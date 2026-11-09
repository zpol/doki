package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds user preferences for DOKI.
type Config struct {
	Theme   string         `yaml:"theme"`
	Refresh RefreshConfig  `yaml:"refresh"`
	Display DisplayConfig  `yaml:"display"`
	Confirm ConfirmConfig  `yaml:"confirmations"`
}

// RefreshConfig controls polling intervals.
type RefreshConfig struct {
	Containers time.Duration `yaml:"containers"`
}

// DisplayConfig controls UI presentation defaults.
type DisplayConfig struct {
	ShowStopped bool `yaml:"show_stopped"`
	Mouse       bool `yaml:"mouse"`
}

// ConfirmConfig controls confirmation prompts.
type ConfirmConfig struct {
	DestructiveActions bool `yaml:"destructive_actions"`
	RemoteActions      bool `yaml:"remote_actions"`
}

// Default returns sensible built-in defaults.
func Default() Config {
	return Config{
		Theme: "cyberpunk",
		Refresh: RefreshConfig{
			Containers: time.Second,
		},
		Display: DisplayConfig{
			ShowStopped: true,
			Mouse:       false,
		},
		Confirm: ConfirmConfig{
			DestructiveActions: true,
			RemoteActions:      true,
		},
	}
}

// Load reads configuration from the user config directory.
// Missing files are not an error; defaults are returned.
func Load() (Config, error) {
	cfg := Default()

	dir, err := os.UserConfigDir()
	if err != nil {
		return cfg, nil
	}
	path := filepath.Join(dir, "doki", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}

	type fileConfig struct {
		Theme   string `yaml:"theme"`
		Refresh struct {
			Containers string `yaml:"containers"`
		} `yaml:"refresh"`
		Display DisplayConfig `yaml:"display"`
		Confirm ConfirmConfig `yaml:"confirmations"`
	}

	var raw fileConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}

	if raw.Theme != "" {
		cfg.Theme = raw.Theme
	}
	if raw.Refresh.Containers != "" {
		d, err := time.ParseDuration(raw.Refresh.Containers)
		if err != nil {
			return cfg, fmt.Errorf("invalid refresh.containers duration: %w", err)
		}
		if d < 500*time.Millisecond {
			return cfg, fmt.Errorf("refresh.containers must be at least 500ms")
		}
		cfg.Refresh.Containers = d
	}
	cfg.Display = raw.Display
	// YAML zeros overwrite defaults when the confirmations block is present.
	cfg.Confirm = raw.Confirm
	return cfg, nil
}
