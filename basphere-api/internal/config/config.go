package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Storage     StorageConfig     `yaml:"storage"`
	Provisioner ProvisionerConfig `yaml:"provisioner"`
}

// ServerConfig represents the HTTP server configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// StorageConfig represents the storage configuration
type StorageConfig struct {
	// Path to store pending requests
	PendingDir string `yaml:"pending_dir"`
}

// ProvisionerConfig represents the provisioner configuration
type ProvisionerConfig struct {
	// Path to basphere-admin script
	AdminScript string `yaml:"admin_script"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Storage: StorageConfig{
			PendingDir: "/var/lib/basphere/pending",
		},
		Provisioner: ProvisionerConfig{
			AdminScript: "/usr/local/bin/basphere-admin",
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// GetEnvOrDefault returns the environment variable value or a default
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
