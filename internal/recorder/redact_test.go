package recorder_test

import (
	"bytes"
	"testing"

	"github.com/0xendale/devtrace/internal/recorder"
)

func TestRedactOpenAIKey(t *testing.T) {
	input := []byte("export KEY=sk-abcdefghijklmnopqrstuvwxyz1234567890")
	out := recorder.Redact(input)
	if bytes.Contains(out, []byte("sk-abc")) {
		t.Errorf("OpenAI key not redacted, got: %s", out)
	}
	if !bytes.Contains(out, []byte("[REDACTED:")) {
		t.Errorf("expected [REDACTED:...] marker, got: %s", out)
	}
}

func TestRedactGitHubPAT(t *testing.T) {
	input := []byte("token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij")
	out := recorder.Redact(input)
	if bytes.Contains(out, []byte("ghp_")) {
		t.Errorf("GitHub PAT not redacted: %s", out)
	}
}

func TestRedactAWSKey(t *testing.T) {
	input := []byte("AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE")
	out := recorder.Redact(input)
	if bytes.Contains(out, []byte("AKIA")) {
		t.Errorf("AWS key not redacted: %s", out)
	}
}

func TestRedactBearerToken(t *testing.T) {
	input := []byte("Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig")
	out := recorder.Redact(input)
	if bytes.Contains(out, []byte("eyJ")) {
		t.Errorf("Bearer token not redacted: %s", out)
	}
}

func TestRedactGenericSecrets(t *testing.T) {
	cases := []struct {
		input     string
		wantClean bool // true = no secret value should remain
	}{
		{"password=hunter2", true},
		{"secret: mysecretvalue", true},
		{"api_key=abc123xyz", true},
		{"TOKEN=somesecrettoken", true},
		{"api-key: xyz789", true},
		{"export password=hunter2 more", true}, // mid-line: space before keyword must be preserved
	}
	for _, tc := range cases {
		out := recorder.Redact([]byte(tc.input))
		if bytes.Equal(out, []byte(tc.input)) {
			t.Errorf("generic secret not redacted: %q", tc.input)
		}
	}
}

func TestRedactPreservesDelimiter(t *testing.T) {
	input := []byte("export password=hunter2")
	out := recorder.Redact(input)
	// "export " prefix should still be present (space before "password" preserved)
	if !bytes.HasPrefix(out, []byte("export ")) {
		t.Errorf("delimiter before secret was dropped; got: %s", out)
	}
	if bytes.Contains(out, []byte("hunter2")) {
		t.Errorf("secret value not redacted; got: %s", out)
	}
}

func TestRedactSafeContent(t *testing.T) {
	input := []byte("go build ./...\ngo test ./...\nAll tests passed.")
	out := recorder.Redact(input)
	if !bytes.Equal(out, input) {
		t.Errorf("safe content was modified: got %s", out)
	}
}

func TestRedactCount(t *testing.T) {
	input := []byte("sk-abcdefghijklmnopqrstuvwxyz1234567890\nghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij")
	_, count := recorder.RedactWithCount(input)
	if count != 2 {
		t.Errorf("want 2 redactions, got %d", count)
	}
}
