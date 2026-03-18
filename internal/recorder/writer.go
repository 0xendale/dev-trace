package recorder

import (
	"fmt"
	"io"
	"os"
)

// OpenLog opens (or creates) the session log at path with mode 0600.
// If the file exists with looser permissions, it tightens them and prints
// a warning to stderr. Always opens in append mode.
func OpenLog(path string) (io.WriteCloser, error) {
	info, statErr := os.Stat(path)
	if statErr != nil && !os.IsNotExist(statErr) {
		return nil, fmt.Errorf("stat session log: %w", statErr)
	}
	if statErr == nil && info.Mode().Perm() != 0600 {
		fmt.Fprintf(os.Stderr, "[devtrace] warning: log file had permissions %04o — tightening to 0600\n", info.Mode().Perm())
		if err := os.Chmod(path, 0600); err != nil {
			return nil, fmt.Errorf("chmod session.log: %w", err)
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("opening session log: %w", err)
	}
	return f, nil
}
