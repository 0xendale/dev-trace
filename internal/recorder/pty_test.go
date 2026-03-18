package recorder_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xendale/devtrace/internal/recorder"
)

// TestRunCapturesOutput launches a PTY running a non-interactive command
// and verifies its output appears in the session log.
func TestRunCapturesOutput(t *testing.T) {
	if os.Getenv("TERM") == "" {
		t.Setenv("TERM", "xterm-256color")
	}
	dir := t.TempDir()
	logPath := filepath.Join(dir, "session.log")

	done := make(chan error, 1)
	go func() {
		done <- recorder.Run("/bin/sh", []string{"-c", "echo hello-devtrace; exit"}, logPath, dir)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for Run to complete")
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	if !strings.Contains(string(data), "hello-devtrace") {
		t.Errorf("expected 'hello-devtrace' in log, got: %s", data)
	}
}

// TestRunRefusesIfAlreadyRecording verifies that starting a second recorder
// when a live PID file exists returns an error.
func TestRunRefusesIfAlreadyRecording(t *testing.T) {
	dir := t.TempDir()
	// Write a PID file for the current (live) process.
	recorder.WritePID(dir, os.Getpid())
	logPath := filepath.Join(dir, "session.log")
	err := recorder.Run("/bin/sh", []string{"-c", "echo hi; exit"}, logPath, dir)
	if err == nil {
		t.Fatal("expected error when recorder is already running, got nil")
	}
}

// TestRunRefusesIfStalePID verifies that Run returns an error even when the
// PID file references a dead process (stale), since only devtrace stop may
// remove the PID file.
func TestRunRefusesIfStalePID(t *testing.T) {
	dir := t.TempDir()
	// Use an absurdly large PID that cannot exist on any UNIX system.
	recorder.WritePID(dir, 99999999)
	logPath := filepath.Join(dir, "session.log")
	err := recorder.Run("/bin/sh", []string{"-c", "echo hi; exit"}, logPath, dir)
	if err == nil {
		t.Fatal("expected error for stale PID file, got nil")
	}
}
