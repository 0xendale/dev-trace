package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/0xendale/devtrace/internal/config"
	"github.com/0xendale/devtrace/internal/git"
	"github.com/0xendale/devtrace/internal/llm"
	"github.com/0xendale/devtrace/internal/recorder"
	"github.com/0xendale/devtrace/internal/report"
)

var version = "dev"

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:     "devtrace",
	Short:   "Record terminal sessions and generate AI-powered reports",
	Version: version,
}

func init() {
	rootCmd.AddCommand(startCmd, stopCmd, genCmd, setupCmd)
	genCmd.Flags().IntVar(&genHours, "hours", 0, "git diff time window in hours (overrides config)")
}

var genHours int

// ── start ──────────────────────────────────────────────────────────────────

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start recording the terminal session",
	RunE:  runStart,
}

func runStart(_ *cobra.Command, _ []string) error {
	dataDir, err := config.DataDir()
	if err != nil {
		return fmt.Errorf("data dir: %w", err)
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	fmt.Fprintln(os.Stderr, "[devtrace] recording started — run 'devtrace stop' to finish")
	return recorder.Run(shell, nil, recorder.LogPath(dataDir), dataDir)
}

// ── stop ───────────────────────────────────────────────────────────────────

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the current recording",
	RunE:  runStop,
}

func runStop(_ *cobra.Command, _ []string) error {
	dataDir, err := config.DataDir()
	if err != nil {
		return fmt.Errorf("data dir: %w", err)
	}

	pid, err := recorder.ReadPID(dataDir)
	if os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "[devtrace] no recording in progress")
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading PID: %w", err)
	}

	if !recorder.IsAlive(pid) {
		fmt.Fprintln(os.Stderr, "[devtrace] stale PID file found, cleaning up")
		return recorder.RemovePID(dataDir)
	}

	proc, _ := os.FindProcess(pid)
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("sending SIGTERM to %d: %w", pid, err)
	}

	// Wait up to 5 seconds for clean exit, then SIGKILL.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !recorder.IsAlive(pid) {
			fmt.Fprintln(os.Stderr, "[devtrace] recording stopped")
			return recorder.RemovePID(dataDir)
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Fprintf(os.Stderr, "[devtrace] recorder did not exit in 5s — sending SIGKILL\n")
	if err := proc.Signal(syscall.SIGKILL); err != nil {
		return fmt.Errorf("sending SIGKILL to %d: %w", pid, err)
	}
	// Brief post-SIGKILL confirmation: wait up to 500ms for the process to actually die.
	postKillDeadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(postKillDeadline) {
		if !recorder.IsAlive(pid) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	fmt.Fprintln(os.Stderr, "[devtrace] recording stopped")
	return recorder.RemovePID(dataDir)
}

// ── gen ────────────────────────────────────────────────────────────────────

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate a Markdown report from the recorded session",
	RunE:  runGen,
}

func runGen(_ *cobra.Command, _ []string) error {
	// Fast pre-condition: must be interactive before doing any I/O work.
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("devtrace gen requires interactive confirmation; run in a terminal")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("no API key found — set DEVTRACE_API_KEY or add api_key to ~/.devtrace/config.toml")
	}

	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}
	logPath := recorder.LogPath(dataDir)

	// Read and validate session log.
	logData, err := os.ReadFile(logPath)
	if err != nil {
		return fmt.Errorf("reading session log: %w", err)
	}
	if len(strings.TrimSpace(string(logData))) == 0 {
		return fmt.Errorf("session log is empty at %s — run 'devtrace start' first", logPath)
	}

	// Determine hours: --hours flag > config > default.
	hours := cfg.Hours
	if genHours > 0 {
		hours = genHours
	}

	// Fetch git diff (best effort — warn if no repo).
	diff, err := git.GetDiff(cwd, hours)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[devtrace] warning: could not get git diff: %v — sending log only\n", err)
		diff = ""
	}

	// Assemble prompt (applies second-pass redaction).
	payload, redactionCount := llm.Assemble(string(logData), diff)

	logLines := strings.Count(string(logData), "\n")
	diffLines := strings.Count(diff, "\n")
	totalKB := float64(len(payload)) / 1024.0

	// Provider-aware endpoint display.
	var endpointDisplay string
	if cfg.Provider == "gemini" {
		endpointDisplay = "generativelanguage.googleapis.com"
	} else {
		endpointDisplay = cfg.Endpoint
		if endpointDisplay == "" {
			endpointDisplay = "api.openai.com"
		}
	}
	fmt.Printf(`About to send to %s:
  Session log: %d lines
  Git diff:    %d lines
  Total:       ~%.1f KB
  Redactions:  %d (across both log write and prompt assembly passes)
Proceed? [y/N] `, endpointDisplay, logLines, diffLines, totalKB, redactionCount)

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("reading confirmation: %w", err)
		}
		// EOF with no input — treat as "no"
		fmt.Fprintln(os.Stderr, "Aborted.")
		return nil
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if answer != "y" {
		fmt.Fprintln(os.Stderr, "Aborted.")
		return nil
	}

	// Select LLM client based on provider.
	var client llm.Completer
	switch cfg.Provider {
	case "gemini":
		client = llm.NewGeminiClient(cfg.APIKey, cfg.Model)
	default:
		if cfg.Endpoint != "" {
			client = llm.NewClientWithURL(cfg.APIKey, cfg.Model, cfg.Endpoint)
		} else {
			client = llm.NewClient(cfg.APIKey, cfg.Model)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	provider := "OpenAI"
	if cfg.Provider == "gemini" {
		provider = "Gemini"
	}
	fmt.Fprintf(os.Stderr, "[devtrace] calling %s API…\n", provider)
	response, err := client.Complete(ctx, llm.SystemPrompt(), payload)
	if err != nil {
		return fmt.Errorf("%s: %w", provider, err)
	}

	// Write REPORT.md to cwd.
	outDir := cwd
	if err := report.Write(outDir, response, os.Stderr); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[devtrace] report written to %s\n", filepath.Join(outDir, "REPORT.md"))
	return nil
}

// ── setup ──────────────────────────────────────────────────────────────────

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup: configure your API key and provider",
	RunE:  runSetup,
}

func runSetup(_ *cobra.Command, _ []string) error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("devtrace setup requires an interactive terminal")
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("devtrace setup")
	fmt.Println()
	fmt.Println("Select provider:")
	fmt.Println("  [1] Gemini  (free tier available — https://aistudio.google.com/apikey)")
	fmt.Println("  [2] OpenAI  (https://platform.openai.com/api-keys)")
	fmt.Println("  [3] Other   (any OpenAI-compatible endpoint)")
	fmt.Print("Choice [1]: ")

	choiceStr, _ := reader.ReadString('\n')
	choiceStr = strings.TrimSpace(choiceStr)
	if choiceStr == "" {
		choiceStr = "1"
	}

	var providerName, model, endpoint string
	switch choiceStr {
	case "2":
		providerName = "openai"
		model = "gpt-4o"
	case "3":
		providerName = "openai"
		fmt.Print("Endpoint URL: ")
		endpoint, _ = reader.ReadString('\n')
		endpoint = strings.TrimSpace(endpoint)
		fmt.Print("Model: ")
		model, _ = reader.ReadString('\n')
		model = strings.TrimSpace(model)
	default:
		providerName = "gemini"
		model = "gemini-2.0-flash"
	}

	fmt.Print("API key: ")
	keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("reading API key: %w", err)
	}
	apiKey := strings.TrimSpace(string(keyBytes))
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dataDir := filepath.Join(home, ".devtrace")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("provider = \"" + providerName + "\"\n")
	sb.WriteString("api_key  = \"" + apiKey + "\"\n")
	sb.WriteString("model    = \"" + model + "\"\n")
	sb.WriteString("hours    = 8\n")
	if endpoint != "" {
		sb.WriteString("endpoint = \"" + endpoint + "\"\n")
	}

	cfgPath := filepath.Join(dataDir, "config.toml")
	if err := os.WriteFile(cfgPath, []byte(sb.String()), 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[devtrace] config written to %s\n", cfgPath)
	return nil
}
