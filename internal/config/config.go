package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// These variables are set at build time via -ldflags
var (
	EmbeddedAPIKey         string
	EmbeddedTranslateModel string
)

// TeleportDatabase holds configuration for a single Teleport database
type TeleportDatabase struct {
	Name        string            `yaml:"name"`         // Custom alias for this database (e.g. prod-db)
	Description string            `yaml:"description"`  // Description of this database
	ServiceName string            `yaml:"service_name"` // Teleport registered service name
	DBName      string            `yaml:"db_name"`      // Actual database name (--db-name parameter)
	DBProtocol  string            `yaml:"db_protocol"`  // Protocol (postgres, mysql, etc.)
	DBUser      string            `yaml:"db_user"`      // Database user
	Cluster     string            `yaml:"cluster"`      // Optional: Teleport cluster
	LocalPort   int               `yaml:"local_port"`   // Local port for proxy mode (0 for auto)
	Labels      map[string]string `yaml:"labels"`       // Optional: Teleport labels
}

// TeleportConfig holds Teleport-related configuration
type TeleportConfig struct {
	Databases []TeleportDatabase `yaml:"databases"`
}

// Config holds application configuration
type Config struct {
	AnthropicAPIKey string          `yaml:"anthropic_api_key"`
	TranslateModel  string          `yaml:"translate_model"`
	Teleport        *TeleportConfig `yaml:"teleport"`
}

// GetDatabase finds a database by name from the Teleport config
func (c *Config) GetDatabase(name string) *TeleportDatabase {
	if c.Teleport == nil {
		return nil
	}
	for i := range c.Teleport.Databases {
		if c.Teleport.Databases[i].Name == name {
			return &c.Teleport.Databases[i]
		}
	}
	return nil
}

// GetDatabases returns all configured databases
func (c *Config) GetDatabases() []TeleportDatabase {
	if c.Teleport == nil {
		return nil
	}
	return c.Teleport.Databases
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

	// Priority: env var > config file > embedded value
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		cfg.AnthropicAPIKey = apiKey
	} else if cfg.AnthropicAPIKey == "" && EmbeddedAPIKey != "" {
		cfg.AnthropicAPIKey = EmbeddedAPIKey
	}

	if model := os.Getenv("MINITOOL_TRANSLATE_MODEL"); model != "" {
		cfg.TranslateModel = model
	} else if cfg.TranslateModel == "" && EmbeddedTranslateModel != "" {
		cfg.TranslateModel = EmbeddedTranslateModel
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
