package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	devgit "github.com/0xendale/devtrace/internal/git"
)

func makeTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")

	// Initial commit
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644)
	run("add", ".")
	run("commit", "-m", "initial")

	// A second commit (recent change)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644)
	run("add", ".")
	run("commit", "-m", "add main func")

	return dir
}

func TestGetDiffReturnsChanges(t *testing.T) {
	dir := makeTestRepo(t)
	diff, err := devgit.GetDiff(dir, 24) // last 24 hours
	if err != nil {
		t.Fatalf("GetDiff: %v", err)
	}
	if !strings.Contains(diff, "func main") {
		t.Errorf("expected diff to contain 'func main', got:\n%s", diff)
	}
}

func TestGetDiffNoRepoReturnsError(t *testing.T) {
	_, err := devgit.GetDiff(t.TempDir(), 8)
	if err == nil {
		t.Fatal("expected error for non-git directory, got nil")
	}
}

func TestGetDiffRootCommitOnly(t *testing.T) {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644)
	run("add", ".")
	run("commit", "-m", "root only")

	diff, err := devgit.GetDiff(dir, 24)
	if err != nil {
		t.Fatalf("GetDiff single-commit repo: %v", err)
	}
	if !strings.Contains(diff, "package main") {
		t.Errorf("expected diff to contain 'package main', got:\n%s", diff)
	}
}

func makeOldTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(env []string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	base := []string{
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@test.com",
	}
	old := append(base,
		"GIT_AUTHOR_DATE=2020-01-01T00:00:00",
		"GIT_COMMITTER_DATE=2020-01-01T00:00:00",
	)

	run(base, "init")
	run(base, "config", "user.email", "test@test.com")
	run(base, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644)
	run(old, "add", ".")
	run(old, "commit", "-m", "old initial")
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644)
	run(old, "add", ".")
	run(old, "commit", "-m", "old second")
	return dir
}

func TestGetDiffNoCommitsInWindow(t *testing.T) {
	dir := makeOldTestRepo(t)
	// Use 0 hours — since == now(), so no commits (dated 2020) fall in the window.
	diff, err := devgit.GetDiff(dir, 0)
	if err != nil {
		t.Fatalf("GetDiff zero-hour window: %v", err)
	}
	if diff != "" {
		t.Errorf("expected empty diff for zero-hour window, got: %s", diff)
	}
}
