package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// LoadSprootConfig reads and parses a sproot.yaml from the given path.
func LoadSprootConfig(path string) (*SprootConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading sproot.yaml: %w", err)
	}
	var cfg SprootConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing sproot.yaml: %w", err)
	}
	return &cfg, nil
}

// LoadHostConfig reads and parses the host config file at the given path.
// Use DefaultHostConfigPath to get the canonical ~/.sproot/config location.
func LoadHostConfig(path string) (*HostConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading host config: %w", err)
	}
	var cfg HostConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing host config: %w", err)
	}
	if cfg.TokenEnv == "" {
		cfg.TokenEnv = "SPRITE_TOKEN"
	}
	return &cfg, nil
}

// DefaultHostConfigPath returns the canonical path for the host config file.
func DefaultHostConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".sproot", "config"), nil
}

// ExpandTilde replaces a leading ~ with the user's home directory.
func ExpandTilde(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}
