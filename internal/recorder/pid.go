package recorder

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func pidPath(dataDir string) string {
	return filepath.Join(dataDir, "recorder.pid")
}

// WritePID writes pid to dataDir/recorder.pid with mode 0600.
// The recorder process must never call RemovePID — that is devtrace stop's job.
func WritePID(dataDir string, pid int) error {
	return os.WriteFile(pidPath(dataDir), []byte(strconv.Itoa(pid)), 0600)
}

// ReadPID returns the PID stored in dataDir/recorder.pid.
// Returns an os.IsNotExist error if no PID file exists.
func ReadPID(dataDir string) (int, error) {
	data, err := os.ReadFile(pidPath(dataDir))
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in recorder.pid: %w", err)
	}
	return pid, nil
}

// RemovePID deletes dataDir/recorder.pid. Only called by devtrace stop.
func RemovePID(dataDir string) error {
	return os.Remove(pidPath(dataDir))
}

// IsAlive reports whether a process with the given PID is running.
func IsAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	return errors.Is(err, syscall.EPERM)
}
