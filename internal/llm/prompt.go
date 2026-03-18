package llm

import (
	"fmt"
	"os"
	"strings"

	"github.com/0xendale/devtrace/internal/recorder"
)

const (
	maxTokenEstimate = 100_000
	charsPerToken    = 4
)

// SystemPrompt returns the fixed system prompt sent to the model.
func SystemPrompt() string {
	return `You are a developer productivity assistant. Given a terminal session log and git diff, generate a structured Markdown report.

Rules:
- Filter out repetitive errors and noise.
- Ignore trivial commands: ls, cd, pwd, clear, echo, cat (on non-source files).
- Focus on meaningful code changes, test results, and build output.
- Format the report with these sections: ## Summary, ## Changes, ## Test Results, ## Issues.
- Be concise and factual.`
}

// Assemble combines the session log and git diff into a single prompt payload,
// applying a second redaction pass and truncating the log (oldest first) if the
// total exceeds the model context limit.
// Returns the assembled string and the total number of redactions applied.
func Assemble(sessionLog, gitDiff string) (string, int) {
	redactedLog, logRedactions := recorder.RedactWithCount([]byte(sessionLog))
	redactedDiff, diffRedactions := recorder.RedactWithCount([]byte(gitDiff))
	totalRedactions := logRedactions + diffRedactions

	log := string(redactedLog)
	diff := string(redactedDiff)

	// Truncate oldest log lines if we would exceed the token limit.
	log = truncateLog(log, diff)

	var sb strings.Builder
	sb.WriteString("## Terminal Session Log\n\n")
	sb.WriteString(log)
	if diff != "" {
		sb.WriteString("\n## Git Diff\n\n")
		sb.WriteString(diff)
	}
	return sb.String(), totalRedactions
}

// truncateLog removes the oldest complete session blocks (delimited by "---") from
// the top of the log until the combined log+diff fits within maxTokenEstimate.
// A warning is printed to stderr if truncation occurs.
func truncateLog(log, diff string) string {
	if estimateTokens(log+diff) <= maxTokenEstimate {
		return log
	}

	// Split into blocks on the "---" separator.
	// Each block ends with "---\n" or "---" at EOF.
	blocks := strings.SplitAfter(log, "---\n")
	originalCount := len(blocks)

	for estimateTokens(strings.Join(blocks, "")+diff) > maxTokenEstimate && len(blocks) > 1 {
		blocks = blocks[1:] // drop oldest block
	}

	dropped := originalCount - len(blocks)
	fmt.Fprintf(os.Stderr, "[devtrace] warning: session log truncated to fit model context window (dropped %d blocks)\n", dropped)
	return strings.Join(blocks, "")
}

func estimateTokens(s string) int {
	return len(s) / charsPerToken
}
