# Release Notes Template

## scribbles vX.Y.Z

**Release Date**: YYYY-MM-DD

### What's New

scribbles is an Apple Music scrobbler for Last.fm that runs as a background daemon on macOS.

### Features

- **Automatic Scrobbling**: Monitors Apple Music playback and scrobbles tracks to Last.fm
- **Last.fm Rules Compliance**: Follows Last.fm scrobbling rules (50% duration or 4 minutes)
- **Background Daemon**: Runs automatically via launchd with minimal resource usage
- **CLI Integration**: Query current track for tmux status lines or other displays
- **Pause/Resume Tracking**: Correctly handles paused playback without resetting timers
- **Offline Queue**: Queues scrobbles when Last.fm is unreachable and retries later
- **State Persistence**: Survives daemon restarts without losing track progress

### Installation

#### Homebrew (Recommended)

```bash
# Coming soon
brew tap jfmyers9/scribbles
brew install scribbles
```

#### Manual Installation

1. Download the appropriate binary for your system:
   - **Apple Silicon (M1/M2/M3)**: `scribbles-darwin-arm64.tar.gz`
   - **Intel**: `scribbles-darwin-amd64.tar.gz`
   - **Universal** (works on both): `scribbles-darwin-universal.tar.gz`

2. Extract and install:

```bash
tar -xzf scribbles-vX.Y.Z-darwin-universal.tar.gz
sudo cp scribbles /usr/local/bin/
```

3. Verify installation:

```bash
scribbles --version
```

### Quick Start

1. **Authenticate with Last.fm**:

```bash
scribbles auth
```

Follow the prompts to enter your Last.fm API credentials and authorize the application.

2. **Install the background daemon**:

```bash
scribbles install
```

This will install and start the scrobbling daemon. It will automatically start on login.

3. **Test the CLI** (optional):

```bash
scribbles now
```

Add to your tmux status line:

```tmux
set -g status-right "#(scribbles now --format '{{.Artist}} - {{.Name}}')"
```

### Configuration

Configuration is stored in `~/.config/scribbles/config.yaml`. The file is created automatically during the auth process.

**Example Configuration**:

```yaml
lastfm:
  api_key: "your_api_key"
  api_secret: "your_api_secret"
  session_key: "your_session_key"

daemon:
  poll_interval: 3s

logging:
  level: info
  file: ~/.local/share/scribbles/logs/daemon.log

now:
  format: "{{.Artist}} - {{.Name}}"
```

### Commands

- `scribbles daemon` - Run the scrobbling daemon (usually managed by launchd)
- `scribbles now` - Display currently playing track
- `scribbles auth` - Authenticate with Last.fm
- `scribbles install` - Install and start background daemon
- `scribbles uninstall` - Stop and remove background daemon
- `scribbles --version` - Show version information

### Data Storage

- **Configuration**: `~/.config/scribbles/config.yaml`
- **Scrobble Queue**: `~/.local/share/scribbles/scrobbles.db` (SQLite)
- **Daemon State**: `~/.local/share/scribbles/daemon-state.json`
- **Logs**: `~/.local/share/scribbles/logs/daemon.log`

### Checksums

Verify download integrity:

```bash
shasum -a 256 -c scribbles-darwin-universal.sha256
```

### System Requirements

- macOS 10.15 (Catalina) or later
- Apple Music app
- Last.fm account and API credentials

### Known Issues

None currently.

### Changelog

- Initial release with core scrobbling functionality
- Background daemon with launchd integration
- CLI for querying current track
- SQLite-backed scrobble queue with retry logic
- Configuration management with Viper
- Structured logging with zerolog

### Contributors

- Jim Myers (@jfmyers9)

### License

See LICENSE file for details.

---

**Full Changelog**: https://github.com/jfmyers9/scribbles/compare/vX.Y.Z-1...vX.Y.Z
