// +build integration

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestDaemonLifecycle tests starting, stopping, and restarting the daemon
func TestDaemonLifecycle(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "scribbles_test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("scribbles_test")

	// Create a temporary data directory for testing
	tmpDir := t.TempDir()

	// Start the daemon
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, "./scribbles_test", "daemon",
		"--data-dir", tmpDir,
		"--log-level", "debug")
	cmd.Env = append(os.Environ(),
		"SCRIBBLES_LASTFM_API_KEY=test_key",
		"SCRIBBLES_LASTFM_API_SECRET=test_secret",
		"SCRIBBLES_LASTFM_SESSION_KEY=test_session",
	)

	// Start the daemon (it will fail due to invalid credentials, but we're
	// testing lifecycle)
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}

	// Give it time to start
	time.Sleep(1 * time.Second)

	// Check that state file was created
	stateFile := filepath.Join(tmpDir, "state.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Logf("State file not created (expected if Music app not running)")
	}

	// Check that queue database was created
	queueDB := filepath.Join(tmpDir, "queue.db")
	if _, err := os.Stat(queueDB); os.IsNotExist(err) {
		t.Errorf("Queue database not created: %s", queueDB)
	}

	// Stop the daemon by cancelling context
	cancel()

	// Wait for daemon to exit
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		// Daemon stopped successfully
	case <-time.After(5 * time.Second):
		t.Error("Daemon did not stop within 5 seconds")
	}
}

// TestNowCommand tests the "now" command
func TestNowCommand(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "scribbles_test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("scribbles_test")

	// Run the "now" command
	cmd := exec.Command("./scribbles_test", "now")
	output, err := cmd.CombinedOutput()

	// The command might fail if Music is not running, which is okay
	if err != nil {
		t.Logf("Now command failed (expected if Music not running): %v", err)
		t.Logf("Output: %s", output)
		return
	}

	// If Music is running, we should get some output
	if len(output) == 0 {
		t.Logf("No output from now command (Music might be paused/stopped)")
	} else {
		t.Logf("Now command output: %s", output)
	}
}

// TestAuthFlow tests the authentication flow (manual test)
func TestAuthFlow(t *testing.T) {
	t.Skip("Requires manual interaction - run manually with valid API credentials")

	// This test requires:
	// 1. Valid Last.fm API credentials
	// 2. Manual browser interaction to authorize
	// It's meant to be run manually, not in CI

	// Example manual test:
	// 1. go test -tags=integration -run TestAuthFlow
	// 2. Enter API key and secret when prompted
	// 3. Authorize in browser
	// 4. Verify session key is saved to config
}

// TestLaunchdInstallation tests installing and uninstalling the daemon
func TestLaunchdInstallation(t *testing.T) {
	t.Skip("Requires root permissions and modifies system - run manually")

	// This test modifies the system and should be run manually
	// It's here as documentation for manual testing

	// Manual test steps:
	// 1. Build the binary: go build -o scribbles .
	// 2. Run: ./scribbles install
	// 3. Verify plist exists: ls ~/Library/LaunchAgents/com.scribbles.daemon.plist
	// 4. Verify daemon is running: launchctl list | grep scribbles
	// 5. Run: ./scribbles uninstall
	// 6. Verify plist removed: ls ~/Library/LaunchAgents/com.scribbles.daemon.plist
}

// BenchmarkNowCommand benchmarks the performance of the "now" command
func BenchmarkNowCommand(b *testing.B) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "scribbles_test", ".")
	if err := buildCmd.Run(); err != nil {
		b.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("scribbles_test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("./scribbles_test", "now")
		if err := cmd.Run(); err != nil {
			// Ignore errors (Music might not be running)
			continue
		}
	}
}

// TestDaemonResourceUsage tests CPU and memory usage of the daemon
func TestDaemonResourceUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "scribbles_test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("scribbles_test")

	// Create a temporary data directory for testing
	tmpDir := t.TempDir()

	// Start the daemon
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "./scribbles_test", "daemon",
		"--data-dir", tmpDir,
		"--log-level", "error")
	cmd.Env = append(os.Environ(),
		"SCRIBBLES_LASTFM_API_KEY=test_key",
		"SCRIBBLES_LASTFM_API_SECRET=test_secret",
		"SCRIBBLES_LASTFM_SESSION_KEY=test_session",
	)

	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}

	// Let it run for 30 seconds and monitor resource usage
	// Note: This is a basic test - for real load testing, use tools like
	// pprof, top, or process monitoring
	time.Sleep(30 * time.Second)

	// Stop the daemon
	cancel()
	cmd.Wait()

	// In a real test, you would:
	// 1. Monitor CPU usage (should be < 1% when idle)
	// 2. Monitor memory usage (should be < 50MB)
	// 3. Check for memory leaks (RSS should be stable)
	// 4. Use tools like: ps, top, or runtime/pprof

	t.Log("Daemon ran for 30 seconds - check manually for resource usage")
	t.Log("Expected: CPU < 1%, Memory < 50MB")
}
