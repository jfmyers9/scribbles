package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jfmyers9/scribbles/internal/config"
	"github.com/jfmyers9/scribbles/internal/scrobbler"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Last.fm",
	Long: `Authenticate with Last.fm to enable scrobbling.

This command will guide you through the Last.fm authentication process:
1. You'll be prompted to enter your Last.fm API key and secret
2. A browser URL will be provided for you to authorize the application
3. After authorization, a session key will be saved to your config file

You can get API credentials from: https://www.last.fm/api/account/create`,
	RunE: runAuth,
}

func init() {
	rootCmd.AddCommand(authCmd)
}

func promptCredentials(reader *bufio.Reader, cfg *config.Config) error {
	if cfg.LastFM.APIKey != "" && cfg.LastFM.APISecret != "" {
		fmt.Println("Found existing API credentials.")
		fmt.Printf("API Key: %s\n", cfg.LastFM.APIKey)
		fmt.Print("\nUse existing credentials? [Y/n]: ")
		response, err := reader.ReadString('\n')
		if err != nil {
			response = "y"
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "" && response != "y" && response != "yes" {
			cfg.LastFM.APIKey = ""
			cfg.LastFM.APISecret = ""
		}
	}

	if cfg.LastFM.APIKey == "" {
		fmt.Print("Enter your Last.fm API Key: ")
		apiKey, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read API key: %w", err)
		}
		cfg.LastFM.APIKey = strings.TrimSpace(apiKey)
	}

	if cfg.LastFM.APISecret == "" {
		fmt.Print("Enter your Last.fm API Secret: ")
		apiSecret, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read API secret: %w", err)
		}
		cfg.LastFM.APISecret = strings.TrimSpace(apiSecret)
	}

	if cfg.LastFM.APIKey == "" || cfg.LastFM.APISecret == "" {
		return fmt.Errorf("API key and secret are required")
	}

	return nil
}

func getSessionWithRetries(ctx context.Context, client *scrobbler.Client, token string) (string, error) {
	const (
		maxRetries = 3
		retryDelay = 2 * time.Second
	)

	var sessionKey string
	var err error
	for i := range maxRetries {
		sessionKey, err = client.GetSession(ctx, token)
		if err == nil {
			return sessionKey, nil
		}

		if i < maxRetries-1 {
			fmt.Printf("Failed to retrieve session (attempt %d/%d). Retrying in %v...\n",
				i+1, maxRetries, retryDelay)
			time.Sleep(retryDelay)
		}
	}

	return "", fmt.Errorf("failed to get session key after %d attempts: %w", maxRetries, err)
}

func runAuth(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	reader := bufio.NewReader(os.Stdin)

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("Last.fm Authentication")
	fmt.Println("======================")
	fmt.Println()
	fmt.Println("You can get API credentials from: https://www.last.fm/api/account/create")
	fmt.Println()

	if err := promptCredentials(reader, cfg); err != nil {
		return err
	}

	client := scrobbler.New(cfg.LastFM.APIKey, cfg.LastFM.APISecret)

	fmt.Println("\nGenerating authentication token...")
	token, authURL, err := client.AuthenticateWithToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate auth token: %w", err)
	}

	fmt.Println("\nPlease visit this URL to authorize scribbles:")
	fmt.Printf("\n  %s\n\n", authURL)
	fmt.Println("After authorizing, press Enter to continue...")
	_, _ = reader.ReadString('\n')

	fmt.Println("Retrieving session key...")
	sessionKey, err := getSessionWithRetries(ctx, client, token)
	if err != nil {
		return err
	}

	cfg.LastFM.SessionKey = sessionKey
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	configPath := config.GetConfigDir()
	fmt.Printf("\n✓ Authentication successful!\n")
	fmt.Printf("✓ Session key saved to %s/config.yaml\n", configPath)
	fmt.Println("\nYou can now use 'scribbles daemon' to start scrobbling.")

	return nil
}
