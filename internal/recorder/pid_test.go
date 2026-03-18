package recorder_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xendale/devtrace/internal/recorder"
)

func TestWriteReadPID(t *testing.T) {
	dir := t.TempDir()
	if err := recorder.WritePID(dir, 12345); err != nil {
		t.Fatalf("WritePID: %v", err)
	}
	pid, err := recorder.ReadPID(dir)
	if err != nil {
		t.Fatalf("ReadPID: %v", err)
	}
	if pid != 12345 {
		t.Errorf("want 12345, got %d", pid)
	}
}

func TestPIDFileMode(t *testing.T) {
	dir := t.TempDir()
	recorder.WritePID(dir, 1)
	info, err := os.Stat(filepath.Join(dir, "recorder.pid"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("want 0600, got %04o", info.Mode().Perm())
	}
}

func TestRemovePID(t *testing.T) {
	dir := t.TempDir()
	recorder.WritePID(dir, 1)
	if err := recorder.RemovePID(dir); err != nil {
		t.Fatalf("RemovePID: %v", err)
	}
	if _, err := recorder.ReadPID(dir); !os.IsNotExist(err) {
		t.Errorf("expected not-exist error after remove, got %v", err)
	}
}

func TestIsAliveCurrentProcess(t *testing.T) {
	if !recorder.IsAlive(os.Getpid()) {
		t.Error("current process should be alive")
	}
}

func TestIsAliveDeadProcess(t *testing.T) {
	// PID 0 is never a valid user process
	if recorder.IsAlive(99999999) {
		t.Error("PID 99999999 should not be alive")
	}
}
