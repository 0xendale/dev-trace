package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds all devtrace settings.
type Config struct {
	APIKey   string `toml:"api_key"`
	Model    string `toml:"model"`
	Hours    int    `toml:"hours"`
	Endpoint string `toml:"endpoint"`
	Provider string `toml:"provider"` // "openai" (default), "gemini"
}

func defaults() Config {
	return Config{Model: "gpt-4o", Hours: 8}
}

// Load reads config, checking the current directory first, then ~/.devtrace/.
// This allows per-project config.toml files to override the global default.
func Load() (Config, error) {
	// Per-project config: ./config.toml takes priority.
	if _, err := os.Stat("config.toml"); err == nil {
		cwd, err := os.Getwd()
		if err != nil {
			return Config{}, err
		}
		return LoadFromDir(cwd)
	}

	// Global fallback: ~/.devtrace/config.toml
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}
	return LoadFromDir(filepath.Join(home, ".devtrace"))
}

// LoadFromDir reads config.toml from dir. Exported for testing.
// On first run (file absent), it creates a skeleton config.toml with mode 0600.
func LoadFromDir(dir string) (Config, error) {
	path := filepath.Join(dir, "config.toml")

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		// First run: write a skeleton so the user knows where to put their API key.
		if mkErr := os.MkdirAll(dir, 0700); mkErr != nil {
			return Config{}, mkErr
		}
		skeleton := `# devtrace configuration
# Run 'devtrace setup' for interactive setup.
#
# provider = "gemini"   # or "openai" (default)
# api_key  = ""         # or set DEVTRACE_API_KEY env var
# model    = "gemini-2.0-flash"  # or "gpt-4o" for OpenAI
# hours    = 8          # git diff time window
# endpoint = ""         # custom endpoint (OpenAI-compatible only)
`
		if mkErr := os.WriteFile(path, []byte(skeleton), 0600); mkErr != nil {
			return Config{}, mkErr
		}
		cfg := defaults()
		applyEnv(&cfg)
		return cfg, nil
	}
	if err != nil {
		return Config{}, err
	}
	if info.Mode().Perm()&0177 != 0 {
		return Config{}, fmt.Errorf("config file %s has unsafe permissions (expected 0600, got %04o)", path, info.Mode().Perm())
	}
	cfg := defaults()
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}
	applyEnv(&cfg)
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("DEVTRACE_API_KEY"); v != "" {
		cfg.APIKey = v
	}
}

// DataDir returns ~/.devtrace/, creating it with mode 0700 if absent.
func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".devtrace")
	return dir, os.MkdirAll(dir, 0700)
}
