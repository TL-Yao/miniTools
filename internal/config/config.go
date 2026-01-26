package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds application configuration
type Config struct {
	AnthropicAPIKey string `yaml:"anthropic_api_key"`
	TranslateModel  string `yaml:"translate_model"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		TranslateModel: "claude-3-haiku-20240307",
	}
}

// Load loads configuration from file and environment
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Try to load from local config.yaml first (in current directory)
	if data, err := os.ReadFile("config.yaml"); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	} else {
		// Fall back to user config directory
		configPath := getConfigPath()
		if data, err := os.ReadFile(configPath); err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		}
	}

	// Environment variables override config file
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		cfg.AnthropicAPIKey = apiKey
	}

	if model := os.Getenv("MINITOOL_TRANSLATE_MODEL"); model != "" {
		cfg.TranslateModel = model
	}

	return cfg, nil
}

func getConfigPath() string {
	// Check XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "minitool", "config.yaml")
	}

	// Fall back to ~/.config
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "minitool", "config.yaml")
}
