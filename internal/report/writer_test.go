package report_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xendale/devtrace/internal/report"
)

func TestWriteCreatesFile(t *testing.T) {
	dir := t.TempDir()
	if err := report.Write(dir, "# My Report\n\nContent here.", os.Stderr); err != nil {
		t.Fatalf("Write: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "REPORT.md"))
	if err != nil {
		t.Fatalf("reading REPORT.md: %v", err)
	}
	if !strings.Contains(string(data), "Content here.") {
		t.Errorf("expected content in REPORT.md, got: %s", data)
	}
}

func TestWriteOverwritesWithWarning(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "REPORT.md"), []byte("old content"), 0644)
	var warnBuf strings.Builder
	if err := report.Write(dir, "new content", &warnBuf); err != nil {
		t.Fatalf("Write: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "REPORT.md"))
	if string(data) != "new content" {
		t.Errorf("expected 'new content', got %q", data)
	}
	if !strings.Contains(warnBuf.String(), "overwriting existing REPORT.md") {
		t.Errorf("expected overwrite warning, got: %q", warnBuf.String())
	}
}

func TestWriteNoWarningWhenNew(t *testing.T) {
	dir := t.TempDir()
	var warnBuf strings.Builder
	if err := report.Write(dir, "content", &warnBuf); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if warnBuf.Len() > 0 {
		t.Errorf("expected no warning for new file, got: %q", warnBuf.String())
	}
}
