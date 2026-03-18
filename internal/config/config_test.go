package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xendale/devtrace/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DEVTRACE_API_KEY", "")
	cfg, err := config.LoadFromDir(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Model != "gpt-4o" {
		t.Errorf("want model gpt-4o, got %q", cfg.Model)
	}
	if cfg.Hours != 8 {
		t.Errorf("want hours 8, got %d", cfg.Hours)
	}
	if cfg.APIKey != "" {
		t.Errorf("want empty APIKey by default, got %q", cfg.APIKey)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	content := []byte("api_key = \"file-key\"\nmodel = \"gpt-3.5-turbo\"\nhours = 4\n")
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), content, 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "file-key" {
		t.Errorf("want api_key file-key, got %q", cfg.APIKey)
	}
	if cfg.Model != "gpt-3.5-turbo" {
		t.Errorf("want model gpt-3.5-turbo, got %q", cfg.Model)
	}
	if cfg.Hours != 4 {
		t.Errorf("want hours 4, got %d", cfg.Hours)
	}
}

func TestEnvVarOverridesFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("api_key = \"file-key\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVTRACE_API_KEY", "env-key")
	cfg, err := config.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "env-key" {
		t.Errorf("env var should override file: want env-key, got %q", cfg.APIKey)
	}
}

func TestUnsafePermissionsRejected(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("api_key = \"key\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := config.LoadFromDir(dir)
	if err == nil {
		t.Fatal("expected error for 0644 config file, got nil")
	}
}

func TestFirstRunCreatesSkeleton(t *testing.T) {
	dir := t.TempDir()
	_, err := config.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("unexpected error on first run: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatal("config.toml should have been created on first run")
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("skeleton file should be 0600, got %04o", info.Mode().Perm())
	}
}
