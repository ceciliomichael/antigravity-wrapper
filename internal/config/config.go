// Package config provides configuration loading and management for the antigravity-wrapper.
package config

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Config holds the application configuration.
type Config struct {
	// Server settings
	Port      int      `yaml:"port"`
	Host      string   `yaml:"host"`
	APIKeys   []string `yaml:"api_keys"`
	RateLimit int      `yaml:"rate_limit"`

	// Security settings
	MasterSecret string `yaml:"master_secret"`
	DataDir      string `yaml:"data_dir"`

	// Proxy settings
	ProxyURL string `yaml:"proxy_url"`

	// Feature flags
	ThinkingAsContent bool `yaml:"thinking_as_content"`

	// Credentials settings
	CredentialsDir string `yaml:"credentials_dir"`

	// Logging settings
	LogLevel string `yaml:"log_level"`
	Debug    bool   `yaml:"debug"`
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Port:           8080,
		Host:           "0.0.0.0",
		DataDir:        "data",
		CredentialsDir: defaultCredentialsDir(),
		LogLevel:       "info",
		Debug:          false,
		RateLimit:      1000,
	}
}

// Load reads configuration from a YAML file.
// If the file doesn't exist, returns default configuration.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debugf("Config file not found at %s, using defaults", path)
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Apply environment variable overrides
	cfg.applyEnvOverrides()

	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides to the configuration.
func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("ANTIGRAVITY_PORT"); v != "" {
		if port, err := parsePort(v); err == nil {
			c.Port = port
		}
	}

	if v := os.Getenv("ANTIGRAVITY_HOST"); v != "" {
		c.Host = v
	}

	if v := os.Getenv("ANTIGRAVITY_MASTER_SECRET"); v != "" {
		c.MasterSecret = v
	}

	if v := os.Getenv("ANTIGRAVITY_DATA_DIR"); v != "" {
		c.DataDir = v
	}

	if v := os.Getenv("ANTIGRAVITY_PROXY_URL"); v != "" {
		c.ProxyURL = v
	}

	if v := os.Getenv("ANTIGRAVITY_THINKING_AS_CONTENT"); v == "true" || v == "1" {
		c.ThinkingAsContent = true
	}

	if v := os.Getenv("ANTIGRAVITY_CREDENTIALS_DIR"); v != "" {
		c.CredentialsDir = v
	}

	if v := os.Getenv("ANTIGRAVITY_API_KEYS"); v != "" {
		keys := strings.Split(v, ",")
		for i, k := range keys {
			keys[i] = strings.TrimSpace(k)
		}
		c.APIKeys = keys
	}

	if v := os.Getenv("ANTIGRAVITY_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}

	if v := os.Getenv("ANTIGRAVITY_DEBUG"); v == "true" || v == "1" {
		c.Debug = true
	}

	if v := os.Getenv("ANTIGRAVITY_RATE_LIMIT"); v != "" {
		if limit, err := parsePort(v); err == nil {
			c.RateLimit = limit
		}
	}
}

// parsePort is a simple port parser.
func parsePort(s string) (int, error) {
	var port int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, os.ErrInvalid
		}
		port = port*10 + int(c-'0')
	}
	return port, nil
}

// defaultCredentialsDir returns the default credentials directory.
func defaultCredentialsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".antigravity"
	}
	return filepath.Join(home, ".antigravity")
}

// EnsureCredentialsDir creates the credentials directory if it doesn't exist.
func (c *Config) EnsureCredentialsDir() error {
	dir := c.CredentialsDir
	if dir == "" {
		dir = defaultCredentialsDir()
	}
	return os.MkdirAll(dir, 0700)
}

// CredentialsPath returns the full path to a credentials file.
func (c *Config) CredentialsPath(filename string) string {
	dir := c.CredentialsDir
	if dir == "" {
		dir = defaultCredentialsDir()
	}
	return filepath.Join(dir, filename)
}

// EnsureDataDir creates the data directory if it doesn't exist.
func (c *Config) EnsureDataDir() error {
	if c.DataDir == "" {
		return nil
	}
	return os.MkdirAll(c.DataDir, 0700)
}
