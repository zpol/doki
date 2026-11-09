package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Theme != "cyberpunk" {
		t.Fatalf("theme=%q", cfg.Theme)
	}
	if cfg.Refresh.Containers != time.Second {
		t.Fatalf("refresh=%v", cfg.Refresh.Containers)
	}
	if !cfg.Confirm.DestructiveActions {
		t.Fatal("expected destructive confirmations enabled")
	}
}

func TestLoadMissingFileUsesDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "cyberpunk" {
		t.Fatalf("theme=%q", cfg.Theme)
	}
}

func TestLoadValidFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfgDir := filepath.Join(dir, "doki")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("theme: nord\nrefresh:\n  containers: 5s\ndisplay:\n  show_stopped: false\n")
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "nord" {
		t.Fatalf("theme=%q", cfg.Theme)
	}
	if cfg.Refresh.Containers != 5*time.Second {
		t.Fatalf("refresh=%v", cfg.Refresh.Containers)
	}
	if cfg.Display.ShowStopped {
		t.Fatal("expected show_stopped false")
	}
}
