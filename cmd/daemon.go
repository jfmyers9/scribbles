package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jfmyers9/scribbles/internal/config"
	"github.com/jfmyers9/scribbles/internal/daemon"
	"github.com/jfmyers9/scribbles/internal/music"
	"github.com/jfmyers9/scribbles/internal/scrobbler"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	daemonLogFile  string
	daemonLogLevel string
	daemonDataDir  string
)

// daemonCmd represents the daemon command
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the scrobbling daemon",
	Long: `Run the scrobbling daemon that monitors Apple Music and scrobbles tracks to Last.fm.

The daemon will:
- Poll Apple Music every few seconds to detect track changes
- Track playback time and handle pause/resume correctly
- Scrobble tracks to Last.fm when they meet the scrobbling threshold (50% or 4 minutes)
- Queue failed scrobbles for retry
- Handle graceful shutdown on SIGINT/SIGTERM

The daemon runs in the foreground and logs to stderr by default.
Use the --log-file flag to log to a file (useful for launchd).`,
	RunE: runDaemon,
}

func init() {
	rootCmd.AddCommand(daemonCmd)

	// Command-line flags
	daemonCmd.Flags().StringVar(&daemonLogFile, "log-file", "", "Log file path (default: stderr)")
	daemonCmd.Flags().StringVar(&daemonLogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	daemonCmd.Flags().StringVar(&daemonDataDir, "data-dir", "", "Data directory for state and queue (default: ~/.local/share/scribbles)")
}

func runDaemon(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate Last.fm credentials
	if cfg.LastFM.APIKey == "" || cfg.LastFM.APISecret == "" || cfg.LastFM.SessionKey == "" {
		return fmt.Errorf("Last.fm credentials not configured. Run 'scribbles auth' first")
	}

	// Set up logging
	logger := setupLogger(daemonLogFile, daemonLogLevel)

	logger.Info().
		Str("version", "dev").
		Msg("Starting scribbles daemon")

	// Determine data directory
	dataDir := daemonDataDir
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".local", "share", "scribbles")
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	logger.Info().Str("data_dir", dataDir).Msg("Using data directory")

	// Create music client
	musicClient := music.NewAppleScriptClient()

	// Create scrobbler client
	scrobblerClient := scrobbler.NewWithSession(
		cfg.LastFM.APIKey,
		cfg.LastFM.APISecret,
		cfg.LastFM.SessionKey,
	)

	// Create daemon config
	daemonCfg := daemon.Config{
		PollInterval:      time.Duration(cfg.PollInterval) * time.Second,
		StateFile:         filepath.Join(dataDir, "state.json"),
		QueueDB:           filepath.Join(dataDir, "queue.db"),
		ProcessInterval:   30 * time.Second, // Process queue every 30 seconds
		ScrobbleThreshold: 0.5,              // 50% threshold (from Last.fm rules)
	}

	// Create daemon
	d, err := daemon.New(daemonCfg, musicClient, scrobblerClient, logger)
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	// Run daemon (blocks until shutdown signal)
	if err := d.Run(); err != nil {
		return fmt.Errorf("daemon error: %w", err)
	}

	// Graceful shutdown
	if err := d.Shutdown(); err != nil {
		logger.Error().Err(err).Msg("Error during shutdown")
		return err
	}

	logger.Info().Msg("Daemon stopped")
	return nil
}

// setupLogger creates a logger with the specified configuration
func setupLogger(logFile, logLevel string) zerolog.Logger {
	// Parse log level
	level := zerolog.InfoLevel
	switch logLevel {
	case "debug":
		level = zerolog.DebugLevel
	case "info":
		level = zerolog.InfoLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	}

	// Set up output
	var output *os.File
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
			output = os.Stderr
		} else {
			output = f
		}
	} else {
		output = os.Stderr
	}

	// Create logger
	logger := zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Logger()

	// Use pretty console output if logging to stderr
	if output == os.Stderr {
		logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}

	return logger
}
