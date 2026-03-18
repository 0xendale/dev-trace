package report

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Write writes content to REPORT.md in dir. If the file already exists,
// it overwrites it and writes a warning to warn (typically os.Stderr).
func Write(dir, content string, warn io.Writer) error {
	path := filepath.Join(dir, "REPORT.md")
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintln(warn, "[devtrace] warning: overwriting existing REPORT.md")
	}
	return os.WriteFile(path, []byte(content), 0644)
}
