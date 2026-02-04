package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	OutputFormat     string
	OutputWidth      int
	PollInterval     int
	MarqueeEnabled   bool
	MarqueeSpeed     int
	MarqueeSeparator string
	LastFM           LastFMConfig
	Logging          LoggingConfig
	TUI              TUIConfig
}

type TUIConfig struct {
	Enabled     bool   // Enable TUI by default when running daemon
	RefreshRate int    // Refresh rate in milliseconds (default 500)
	Theme       string // Color theme: "default", "minimal", "colorful"
}

type LoggingConfig struct {
	Level string
	File  string
}

type LastFMConfig struct {
	APIKey     string
	APISecret  string
	SessionKey string
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")

	configDir := getConfigDir()
	v.AddConfigPath(configDir)
	v.AddConfigPath(".")

	v.SetDefault("output_format", "{{.Artist}} - {{.Name}}")
	v.SetDefault("output_width", 0)
	v.SetDefault("poll_interval", 3)
	v.SetDefault("marquee_enabled", false)
	v.SetDefault("marquee_speed", 2)
	v.SetDefault("marquee_separator", " • ")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.file", "")
	v.SetDefault("tui.enabled", false)
	v.SetDefault("tui.refresh_rate", 500)
	v.SetDefault("tui.theme", "default")

	_ = v.ReadInConfig()

	v.SetEnvPrefix("SCRIBBLES")
	v.AutomaticEnv()

	cfg := &Config{
		OutputFormat:     v.GetString("output_format"),
		OutputWidth:      v.GetInt("output_width"),
		PollInterval:     v.GetInt("poll_interval"),
		MarqueeEnabled:   v.GetBool("marquee_enabled"),
		MarqueeSpeed:     v.GetInt("marquee_speed"),
		MarqueeSeparator: v.GetString("marquee_separator"),
		LastFM: LastFMConfig{
			APIKey:     v.GetString("lastfm.api_key"),
			APISecret:  v.GetString("lastfm.api_secret"),
			SessionKey: v.GetString("lastfm.session_key"),
		},
		Logging: LoggingConfig{
			Level: v.GetString("logging.level"),
			File:  v.GetString("logging.file"),
		},
		TUI: TUIConfig{
			Enabled:     v.GetBool("tui.enabled"),
			RefreshRate: v.GetInt("tui.refresh_rate"),
			Theme:       v.GetString("tui.theme"),
		},
	}

	return cfg, nil
}

func getConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}

	configDir := filepath.Join(homeDir, ".config", "scribbles")
	_ = os.MkdirAll(configDir, 0755)

	return configDir
}

func GetConfigDir() string {
	return getConfigDir()
}

func GetDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}

	dataDir := filepath.Join(homeDir, ".local", "share", "scribbles")
	_ = os.MkdirAll(dataDir, 0755)

	return dataDir
}

func GetLogDir() string {
	dataDir := GetDataDir()
	logDir := filepath.Join(dataDir, "logs")
	_ = os.MkdirAll(logDir, 0755)

	return logDir
}

func (c *Config) Validate() error {
	if c.PollInterval < 1 {
		return fmt.Errorf("poll_interval must be at least 1 second (got %d)", c.PollInterval)
	}
	if c.PollInterval > 60 {
		return fmt.Errorf("poll_interval should not exceed 60 seconds (got %d)", c.PollInterval)
	}

	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if c.Logging.Level != "" && !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level %q (must be one of: debug, info, warn, error)", c.Logging.Level)
	}

	return nil
}

func (c *Config) ValidateLastFM() error {
	if c.LastFM.APIKey == "" {
		return fmt.Errorf("last.fm API key not configured\n\nTo configure Last.fm:\n  1. Get API credentials from https://www.last.fm/api/account/create\n  2. Run: scribbles auth")
	}
	if c.LastFM.APISecret == "" {
		return fmt.Errorf("last.fm API secret not configured\n\nTo configure Last.fm:\n  1. Get API credentials from https://www.last.fm/api/account/create\n  2. Run: scribbles auth")
	}
	if c.LastFM.SessionKey == "" {
		return fmt.Errorf("last.fm session key not configured\n\nTo authenticate:\n  Run: scribbles auth")
	}
	return nil
}

func (c *Config) Save() error {
	v := viper.New()

	configDir := getConfigDir()
	configFile := filepath.Join(configDir, "config.yaml")

	v.Set("output_format", c.OutputFormat)
	v.Set("output_width", c.OutputWidth)
	v.Set("poll_interval", c.PollInterval)
	v.Set("marquee_enabled", c.MarqueeEnabled)
	v.Set("marquee_speed", c.MarqueeSpeed)
	v.Set("marquee_separator", c.MarqueeSeparator)
	v.Set("lastfm.api_key", c.LastFM.APIKey)
	v.Set("lastfm.api_secret", c.LastFM.APISecret)
	v.Set("lastfm.session_key", c.LastFM.SessionKey)
	v.Set("logging.level", c.Logging.Level)
	v.Set("logging.file", c.Logging.File)
	v.Set("tui.enabled", c.TUI.Enabled)
	v.Set("tui.refresh_rate", c.TUI.RefreshRate)
	v.Set("tui.theme", c.TUI.Theme)

	return v.WriteConfigAs(configFile)
}
