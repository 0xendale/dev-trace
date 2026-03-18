package llm_test

import (
	"strings"
	"testing"

	"github.com/0xendale/devtrace/internal/llm"
)

func TestAssembleIncludesBothParts(t *testing.T) {
	log := "[2026-03-17T10:00:00] $ go test ./...\nok  github.com/foo/bar\n---\n"
	diff := "diff --git a/main.go b/main.go\n+func main() {}\n"

	payload, redactions := llm.Assemble(log, diff)
	if !strings.Contains(payload, "go test") {
		t.Errorf("payload missing log content")
	}
	if !strings.Contains(payload, "func main") {
		t.Errorf("payload missing diff content")
	}
	_ = redactions
}

func TestAssembleTruncatesOldestLines(t *testing.T) {
	// Build a log that is large enough to exceed the token limit.
	// Each line is ~100 chars; 100k tokens * 4 chars/token = 400k chars; use 5000 lines.
	var sb strings.Builder
	for i := 0; i < 5000; i++ {
		sb.WriteString("[2026-03-17T10:00:00] $ echo line-number-")
		for j := 0; j < 70; j++ {
			sb.WriteByte('x')
		}
		sb.WriteString("\n---\n")
	}
	bigLog := sb.String()

	payload, _ := llm.Assemble(bigLog, "")
	// The first (oldest) lines should be gone, but the structure should be preserved.
	if strings.Contains(payload, "line-number-"+strings.Repeat("x", 70)) &&
		len(payload) > 400_000 {
		t.Errorf("log was not truncated: payload length = %d", len(payload))
	}
}

func TestSystemPromptContent(t *testing.T) {
	sp := llm.SystemPrompt()
	for _, keyword := range []string{"repetitive", "ls", "cd", "Summary", "Changes"} {
		if !strings.Contains(sp, keyword) {
			t.Errorf("system prompt missing expected keyword %q", keyword)
		}
	}
}

func TestAssembleRedactsSecrets(t *testing.T) {
	log := "[2026-03-17T10:00:00] $ export KEY=sk-abcdefghijklmnopqrstuvwxyz1234567890\n---\n"
	payload, count := llm.Assemble(log, "")
	if strings.Contains(payload, "sk-abc") {
		t.Errorf("secret not redacted in assembled payload")
	}
	if count == 0 {
		t.Errorf("expected at least 1 redaction, got 0")
	}
}
