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

// TeleportEnvironment holds configuration for a Teleport environment (e.g. production, staging)
type TeleportEnvironment struct {
	Proxy   string `yaml:"proxy"`   // Proxy address (e.g. general-prod.xxx.com:443)
	Cluster string `yaml:"cluster"` // Cluster address (e.g. general.xxx.prod.cdcinternal.com)
}

// TeleportDatabase holds configuration for a single Teleport database
type TeleportDatabase struct {
	Name        string            `yaml:"name"`         // Custom alias for this database (e.g. prod-db)
	Description string            `yaml:"description"`  // Description of this database
	Environment string            `yaml:"environment"`  // Environment name (e.g. production, staging)
	ServiceName string            `yaml:"service_name"` // Teleport registered service name
	DBName      string            `yaml:"db_name"`      // Actual database name (--db-name parameter)
	DBProtocol  string            `yaml:"db_protocol"`  // Protocol (postgres, mysql, etc.)
	DBUser      string            `yaml:"db_user"`      // Database user
	Cluster     string            `yaml:"cluster"`      // Optional: Teleport cluster (deprecated, use environment)
	LocalPort   int               `yaml:"local_port"`   // Local port for proxy mode (0 for auto)
	Labels      map[string]string `yaml:"labels"`       // Optional: Teleport labels
}

// TeleportConfig holds Teleport-related configuration
type TeleportConfig struct {
	Environments map[string]TeleportEnvironment `yaml:"environments"` // Environment configurations
	Databases    []TeleportDatabase             `yaml:"databases"`
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

// GetDatabasesByEnvironment returns all databases for a specific environment
func (c *Config) GetDatabasesByEnvironment(envName string) []TeleportDatabase {
	if c.Teleport == nil {
		return nil
	}
	var result []TeleportDatabase
	for _, db := range c.Teleport.Databases {
		if db.Environment == envName {
			result = append(result, db)
		}
	}
	return result
}

// GetEnvironment returns the environment configuration by name
func (c *Config) GetEnvironment(envName string) *TeleportEnvironment {
	if c.Teleport == nil || c.Teleport.Environments == nil {
		return nil
	}
	if env, ok := c.Teleport.Environments[envName]; ok {
		return &env
	}
	return nil
}

// GetEnvironmentNames returns all configured environment names
func (c *Config) GetEnvironmentNames() []string {
	if c.Teleport == nil || c.Teleport.Environments == nil {
		return nil
	}
	names := make([]string, 0, len(c.Teleport.Environments))
	for name := range c.Teleport.Environments {
		names = append(names, name)
	}
	return names
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		TranslateModel: "claude-3-haiku-20240307",
	}
}

// Load loads configuration from file and environment
// Priority (highest to lowest):
// 1. Environment variables (ANTHROPIC_API_KEY, MINITOOL_TRANSLATE_MODEL)
// 2. Local config.yaml (in current directory)
// 3. User config directory (~/.config/minitool/config.yaml)
// 4. Embedded config.yaml (compiled into binary)
// 5. Embedded ldflags values (backward compatibility)
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Priority 1: Load from embedded config.yaml (lowest priority, load first)
	if len(embeddedConfigYAML) > 0 && embeddedConfigYAML != "# Placeholder\n" {
		if err := yaml.Unmarshal([]byte(embeddedConfigYAML), cfg); err == nil {
			// Embedded config loaded successfully
		}
		// Ignore errors - embedded config is optional
	}

	// Priority 2: Try to load from user config directory (overrides embedded)
	configPath := getConfigPath()
	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// Priority 3: Try to load from local config.yaml (overrides all file-based configs)
	if data, err := os.ReadFile("config.yaml"); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// Priority 4: Apply ldflags embedded values (backward compatibility)
	// Only apply if not already set by config files
	if cfg.AnthropicAPIKey == "" && EmbeddedAPIKey != "" {
		cfg.AnthropicAPIKey = EmbeddedAPIKey
	}
	if cfg.TranslateModel == "" && EmbeddedTranslateModel != "" {
		cfg.TranslateModel = EmbeddedTranslateModel
	}

	// Priority 5: Environment variables (highest priority, overrides everything)
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
