package git

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GetDiff returns the combined git diff of all commits made within the last
// hours hours in the repository rooted at dir.
// Returns an error if dir is not a git repository.
func GetDiff(dir string, hours int) (string, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour).Format(time.RFC3339)

	// Collect commit SHAs in the time window.
	logOut, err := runGit(dir, "log", "--since="+since, "--format=%H")
	if err != nil {
		return "", fmt.Errorf("git log: %w", err)
	}
	shas := strings.Fields(logOut)
	if len(shas) == 0 {
		return "", nil // No commits in window — not an error.
	}

	// Diff from the parent of the oldest commit to HEAD.
	oldest := shas[len(shas)-1]

	// Check whether oldest is a root commit (no parent).
	_, parentErr := runGit(dir, "rev-parse", "--verify", oldest+"^")
	if parentErr != nil {
		// Root commit: diff against the empty tree for consistent output format.
		const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
		diff, err := runGit(dir, "diff", emptyTree, "HEAD")
		if err != nil {
			return "", fmt.Errorf("git diff (root): %w", err)
		}
		return diff, nil
	}

	diff, err := runGit(dir, "diff", oldest+"^", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	return diff, nil
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w\n%s", err, out)
	}
	return string(out), nil
}
