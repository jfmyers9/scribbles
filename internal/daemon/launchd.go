package daemon

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.scribbles.daemon</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{.BinaryPath}}</string>
		<string>daemon</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>{{.LogPath}}/scribbles.log</string>
	<key>StandardErrorPath</key>
	<string>{{.LogPath}}/scribbles.err</string>
	<key>WorkingDirectory</key>
	<string>{{.WorkingDirectory}}</string>
	<key>EnvironmentVariables</key>
	<dict>
		<key>PATH</key>
		<string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
	</dict>
</dict>
</plist>
`

// PlistConfig holds the configuration for generating a launchd plist
type PlistConfig struct {
	BinaryPath       string
	LogPath          string
	WorkingDirectory string
}

// GeneratePlist generates a launchd plist file from the template
func GeneratePlist(config PlistConfig) (string, error) {
	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse plist template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return "", fmt.Errorf("failed to execute plist template: %w", err)
	}

	return buf.String(), nil
}

// GetPlistPath returns the path where the plist should be installed
func GetPlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(home, "Library", "LaunchAgents", "com.scribbles.daemon.plist"), nil
}

// GetDefaultLogPath returns the default path for daemon logs
func GetDefaultLogPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(home, ".local", "share", "scribbles", "logs"), nil
}
