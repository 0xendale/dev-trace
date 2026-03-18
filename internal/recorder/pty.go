package recorder

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// Run launches shell with args inside a PTY, tees stdout/stderr through the
// redaction pipeline to the session log at logPath, and writes its PID to
// dataDir/recorder.pid. Stdin is forwarded to the PTY but never logged.
//
// Run blocks until the child shell exits.
// The caller (devtrace start) is responsible for not calling Run when a live
// recording already exists.
func Run(shell string, args []string, logPath, dataDir string) error {
	if os.Getenv("DEVTRACE_ACTIVE") == "1" {
		return fmt.Errorf("devtrace is already recording in this session. Open a new terminal or run 'devtrace stop' first.")
	}

	// Concurrent-start guard: refuse if a live recording is in progress.
	if existingPID, err := ReadPID(dataDir); err == nil {
		// Refuse whether the PID is live or stale — all PID removal is devtrace stop's job.
		if IsAlive(existingPID) {
			return fmt.Errorf("a recording is already in progress (PID %d). Run devtrace stop first.", existingPID)
		}
		return fmt.Errorf("stale PID file found for PID %d. Run devtrace stop to clean up.", existingPID)
	}

	logFile, err := OpenLog(logPath)
	if err != nil {
		return err
	}
	defer logFile.Close()

	c := exec.Command(shell, args...)
	c.Env = os.Environ()
	c.Env = append(c.Env, "DEVTRACE_ACTIVE=1")

	ptmx, err := pty.Start(c)
	if err != nil {
		return fmt.Errorf("starting PTY: %w", err)
	}
	defer ptmx.Close()

	// Write our own PID so devtrace stop can find us.
	// The recorder process must never remove this file — that is devtrace stop's job.
	if err := WritePID(dataDir, os.Getpid()); err != nil {
		_ = c.Wait() // reap the child we already started
		return fmt.Errorf("writing PID: %w", err)
	}

	// Handle terminal resize signals — only when stdin is a real terminal.
	stdinIsTTY := term.IsTerminal(int(os.Stdin.Fd()))

	if stdinIsTTY {
		winchCh := make(chan os.Signal, 1)
		signal.Notify(winchCh, syscall.SIGWINCH)
		go func() {
			for range winchCh {
				_ = pty.InheritSize(os.Stdin, ptmx)
			}
		}()
		winchCh <- syscall.SIGWINCH // apply initial size
		defer func() {
			signal.Stop(winchCh)
			close(winchCh)
		}()

		// Set stdin to raw mode so the PTY gets unprocessed keystrokes.
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("setting raw mode: %w", err)
		}
		defer term.Restore(int(os.Stdin.Fd()), oldState)
	}

	// stdin → PTY slave (forwarded only, not logged).
	go io.Copy(ptmx, os.Stdin)

	// PTY output → stdout AND redact pipeline → log file.
	io.Copy(io.MultiWriter(os.Stdout, &redactWriter{w: logFile}), ptmx)

	return c.Wait()
}

// redactWriter passes each Write through the redaction pipeline before
// forwarding to the underlying writer. It always reports the original
// byte count to avoid confusing callers.
type redactWriter struct {
	w io.Writer
}

func (r *redactWriter) Write(p []byte) (int, error) {
	_, err := r.w.Write(Redact(p))
	return len(p), err
}

// LogPath returns the canonical session log path for the given data directory.
func LogPath(dataDir string) string {
	return filepath.Join(dataDir, "session.log")
}
