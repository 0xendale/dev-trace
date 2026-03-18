package recorder_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xendale/devtrace/internal/recorder"
)

func TestOpenLogCreatesWithMode0600(t *testing.T) {
	dir := t.TempDir()
	w, err := recorder.OpenLog(filepath.Join(dir, "session.log"))
	if err != nil {
		t.Fatalf("OpenLog: %v", err)
	}
	w.Close()
	info, _ := os.Stat(filepath.Join(dir, "session.log"))
	if info.Mode().Perm() != 0600 {
		t.Errorf("want 0600, got %04o", info.Mode().Perm())
	}
}

func TestOpenLogTightensLoosePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.log")
	os.WriteFile(path, nil, 0644) // too loose
	w, err := recorder.OpenLog(path)
	if err != nil {
		t.Fatalf("OpenLog: %v", err)
	}
	w.Close()
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Errorf("permissions should have been tightened to 0600, got %04o", info.Mode().Perm())
	}
}

func TestWriteAppendsToLog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.log")

	w, err := recorder.OpenLog(path)
	if err != nil {
		t.Fatalf("first OpenLog: %v", err)
	}
	if _, err := w.Write([]byte("first line\n")); err != nil {
		t.Fatalf("first write: %v", err)
	}
	w.Close()

	w2, err := recorder.OpenLog(path)
	if err != nil {
		t.Fatalf("second OpenLog: %v", err)
	}
	if _, err := w2.Write([]byte("second line\n")); err != nil {
		t.Fatalf("second write: %v", err)
	}
	w2.Close()

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "first line") || !strings.Contains(string(data), "second line") {
		t.Errorf("both lines should be present, got: %s", data)
	}
}
