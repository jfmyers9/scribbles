package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jfmyers9/scribbles/internal/config"
	"github.com/jfmyers9/scribbles/internal/daemon"
	"github.com/jfmyers9/scribbles/internal/discord"
	"github.com/jfmyers9/scribbles/internal/music"
	"github.com/jfmyers9/scribbles/internal/scrobbler"
	"github.com/jfmyers9/scribbles/internal/tui"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	daemonLogFile  string
	daemonLogLevel string
	daemonDataDir  string
	daemonTUI      bool
	daemonDiscord  bool
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
- Optionally show the current track via Discord Rich Presence (--discord)
- Handle graceful shutdown on SIGINT/SIGTERM

The daemon runs in the foreground and logs to stderr by default.
Use the --log-file flag to log to a file (useful for launchd).`,
	RunE: runDaemon,
}

func init() {
	rootCmd.AddCommand(daemonCmd)

	daemonCmd.Flags().StringVar(&daemonLogFile, "log-file", "", "Log file path (default: stderr)")
	daemonCmd.Flags().StringVar(&daemonLogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	daemonCmd.Flags().StringVar(&daemonDataDir, "data-dir", "", "Data directory for state and queue (default: ~/.local/share/scribbles)")
	daemonCmd.Flags().BoolVar(&daemonTUI, "tui", false, "Enable terminal UI for now playing display")
	daemonCmd.Flags().BoolVar(&daemonDiscord, "discord", false, "Enable Discord Rich Presence")
}

func runDaemon(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	if err := cfg.ValidateLastFM(); err != nil {
		return err
	}

	// Resolve data directory early so TUI mode can use it for log redirection
	dataDir := daemonDataDir
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".local", "share", "scribbles")
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	logFile := daemonLogFile
	if logFile == "" {
		logFile = cfg.Logging.File
	}

	// When TUI is active, logging to stderr corrupts tview's terminal.
	// Redirect logs to a file so they are preserved but don't interfere.
	enableTUI := daemonTUI || cfg.TUI.Enabled
	if enableTUI && logFile == "" {
		logFile = filepath.Join(dataDir, "daemon.log")
	}

	logLevel := daemonLogLevel
	if !cmd.Flags().Changed("log-level") && cfg.Logging.Level != "" {
		logLevel = cfg.Logging.Level
	}

	logger := setupLogger(logFile, logLevel)

	logger.Info().
		Str("version", "dev").
		Msg("Starting scribbles daemon")

	logger.Info().Str("data_dir", dataDir).Msg("Using data directory")

	musicClient := music.NewAppleScriptClient()
	scrobblerClient := scrobbler.NewWithSession(
		cfg.LastFM.APIKey,
		cfg.LastFM.APISecret,
		cfg.LastFM.SessionKey,
	)

	daemonCfg := daemon.Config{
		PollInterval:      time.Duration(cfg.PollInterval) * time.Second,
		StateFile:         filepath.Join(dataDir, "state.json"),
		QueueDB:           filepath.Join(dataDir, "queue.db"),
		ProcessInterval:   30 * time.Second,
		ScrobbleThreshold: 0.5,
	}

	d, err := daemon.New(daemonCfg, musicClient, scrobblerClient, logger)
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	// Start Discord Rich Presence if enabled
	enableDiscord := daemonDiscord || cfg.Discord.Enabled
	if enableDiscord && cfg.Discord.AppID != "" {
		discordUpdates := d.EnableDiscord()
		presence := discord.New(cfg.Discord.AppID, logger)
		discordCtx, discordCancel := context.WithCancel(context.Background())
		defer discordCancel()
		go presence.Run(discordCtx, discordUpdates)
	}

	if enableTUI {
		return runDaemonWithTUI(d, musicClient, cfg, logger)
	}

	if err := d.Run(); err != nil {
		return fmt.Errorf("daemon error: %w", err)
	}

	if err := d.Shutdown(); err != nil {
		logger.Error().Err(err).Msg("Error during shutdown")
		return err
	}

	logger.Info().Msg("Daemon stopped")
	return nil
}

func runDaemonWithTUI(d *daemon.Daemon, musicClient music.Client, cfg *config.Config, logger zerolog.Logger) error {
	// Enable TUI updates channel
	updates := d.EnableTUI()

	// Create TUI config from app config
	tuiCfg := tui.Config{
		RefreshRate: time.Duration(cfg.TUI.RefreshRate) * time.Millisecond,
		Theme:       cfg.TUI.Theme,
	}

	// Create TUI application with config
	tuiApp := tui.NewWithConfig(tuiCfg)
	tuiApp.SetMusicClient(musicClient)

	// Create context for daemon
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup

	// Run daemon in background
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := d.Run(); err != nil {
			logger.Error().Err(err).Msg("Daemon error")
		}
		// When daemon stops, stop TUI
		tuiApp.Stop()
	}()

	// Run TUI (blocks until user quits)
	err := tuiApp.Run(ctx, updates, d.GetState, d.GetPlayedDuration)

	// Cancel context to signal daemon to stop
	cancel()

	// Wait for daemon to finish
	wg.Wait()

	// Shutdown daemon
	if shutdownErr := d.Shutdown(); shutdownErr != nil {
		logger.Error().Err(shutdownErr).Msg("Error during shutdown")
		if err == nil {
			err = shutdownErr
		}
	}

	logger.Info().Msg("Daemon stopped")
	return err
}

func setupLogger(logFile, logLevel string) zerolog.Logger {
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

	logger := zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Logger()

	if output == os.Stderr {
		logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}

	return logger
}
