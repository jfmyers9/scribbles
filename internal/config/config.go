package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds application configuration
type Config struct {
	// Output format template for the now command
	// Default: "{{.Artist}} - {{.Name}}"
	OutputFormat string

	// Poll interval for the daemon (in seconds)
	PollInterval int

	// Last.fm API credentials
	LastFM LastFMConfig
}

// LastFMConfig holds Last.fm specific configuration
type LastFMConfig struct {
	APIKey     string
	APISecret  string
	SessionKey string
}

// Load reads configuration from file and environment
func Load() (*Config, error) {
	v := viper.New()

	// Set config name and paths
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Config file locations (in order of precedence)
	configDir := getConfigDir()
	v.AddConfigPath(configDir)
	v.AddConfigPath(".")

	// Set defaults
	v.SetDefault("output_format", "{{.Artist}} - {{.Name}}")
	v.SetDefault("poll_interval", 3)

	// Read config file (optional - don't fail if missing)
	_ = v.ReadInConfig()

	// Read from environment variables
	v.SetEnvPrefix("SCRIBBLES")
	v.AutomaticEnv()

	// Map config to struct
	cfg := &Config{
		OutputFormat: v.GetString("output_format"),
		PollInterval: v.GetInt("poll_interval"),
		LastFM: LastFMConfig{
			APIKey:     v.GetString("lastfm.api_key"),
			APISecret:  v.GetString("lastfm.api_secret"),
			SessionKey: v.GetString("lastfm.session_key"),
		},
	}

	return cfg, nil
}

// getConfigDir returns the configuration directory path
// Creates the directory if it doesn't exist
func getConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}

	configDir := filepath.Join(homeDir, ".config", "scribbles")

	// Create config directory if it doesn't exist
	_ = os.MkdirAll(configDir, 0755)

	return configDir
}

// GetConfigDir returns the configuration directory path (public helper)
func GetConfigDir() string {
	return getConfigDir()
}

// Save writes configuration to file
func (c *Config) Save() error {
	v := viper.New()

	// Set config file path
	configDir := getConfigDir()
	configFile := filepath.Join(configDir, "config.yaml")

	// Set values in viper
	v.Set("output_format", c.OutputFormat)
	v.Set("poll_interval", c.PollInterval)
	v.Set("lastfm.api_key", c.LastFM.APIKey)
	v.Set("lastfm.api_secret", c.LastFM.APISecret)
	v.Set("lastfm.session_key", c.LastFM.SessionKey)

	// Write to file
	return v.WriteConfigAs(configFile)
}
