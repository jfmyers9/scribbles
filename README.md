# Scribbles

A lightweight Apple Music scrobbler for Last.fm that runs as a macOS
daemon.

## Features

- **Automatic Scrobbling**: Monitors Apple Music and scrobbles tracks to
  Last.fm following official scrobbling rules (50% or 4 minutes)
- **Background Daemon**: Runs unobtrusively in the background via launchd
- **Smart Tracking**: Handles pause/resume, track skips, and repeats
  correctly
- **Offline Queue**: Queues scrobbles when offline and retries
  automatically
- **CLI Status**: Query current track for tmux/status bars
- **Easy Setup**: Simple authentication flow and automatic installation

## Installation

### From Source

```bash
git clone https://github.com/jfmyers9/scribbles.git
cd scribbles
go build -o scribbles .
sudo mv scribbles /usr/local/bin/
```

## Quick Start

### 1. Authenticate with Last.fm

First, get your Last.fm API credentials:
1. Visit https://www.last.fm/api/account/create
2. Create an application to get your API key and secret

Then authenticate:

```bash
scribbles auth
```

This will:
- Prompt you for your API key and secret
- Generate an authorization URL for you to visit
- Wait for you to authorize the application
- Save your session key to the config file

### 2. Install the Daemon

Install and start the background daemon:

```bash
scribbles install
```

This will:
- Create a launchd plist at
  `~/Library/LaunchAgents/com.scribbles.daemon.plist`
- Start the daemon automatically
- Configure it to start on login

The daemon will now monitor Apple Music and scrobble tracks to Last.fm.

### 3. Check Current Track

Query what's currently playing:

```bash
scribbles now
```

## Configuration

Configuration is stored in `~/.config/scribbles/config.yaml`.

Example configuration:

```yaml
# Output format template for the "now" command
# Available fields: .Name, .Artist, .Album, .Duration, .Position, .State
output_format: "{{.Artist}} - {{.Name}}"

# Polling interval for the daemon (in seconds)
poll_interval: 3

# Logging configuration
logging:
  level: info  # debug, info, warn, error
  file: ""     # empty for stderr, or path to log file

# Last.fm API credentials (set via "scribbles auth")
lastfm:
  api_key: "your-api-key"
  api_secret: "your-api-secret"
  session_key: "your-session-key"
```

## Commands

### `scribbles daemon`

Run the scrobbling daemon in the foreground.

```bash
scribbles daemon [flags]
```

Flags:
- `--log-file <path>`: Log to a file instead of stderr
- `--log-level <level>`: Set log level (debug, info, warn, error)
- `--data-dir <path>`: Data directory for state and queue (default:
  `~/.local/share/scribbles`)

The daemon:
- Polls Apple Music every 3 seconds (configurable)
- Tracks playback time and handles pause/resume
- Scrobbles tracks when they reach 50% or 4 minutes
- Queues failed scrobbles for retry
- Handles graceful shutdown on SIGINT/SIGTERM

### `scribbles now`

Display the currently playing track.

```bash
scribbles now [flags]
```

Flags:
- `--format <template>`: Override the output format template

Examples:

```bash
# Default format
scribbles now
# Output: Artist Name - Track Name

# Custom format
scribbles now --format "{{.Name}} by {{.Artist}}"
# Output: Track Name by Artist Name

# Full format
scribbles now --format "{{.Artist}} - {{.Name}} ({{.Album}})"
# Output: Artist Name - Track Name (Album Name)
```

Exit codes:
- `0`: Music is playing
- `1`: Music is stopped or paused

### `scribbles auth`

Authenticate with Last.fm.

```bash
scribbles auth
```

Interactive command that:
1. Prompts for API key and secret
2. Generates an authorization URL
3. Opens your browser to authorize the application
4. Saves the session key to your config file

### `scribbles install`

Install the daemon as a launchd agent.

```bash
scribbles install
```

This creates and loads a launchd plist that:
- Starts the daemon automatically on login
- Restarts it if it crashes
- Logs to `~/.local/share/scribbles/logs/`

### `scribbles uninstall`

Uninstall the daemon.

```bash
scribbles uninstall
```

Stops the daemon and removes the launchd plist.

## Integration with tmux

Add the current track to your tmux status line:

```tmux
set -g status-right "#(scribbles now 2>/dev/null || echo '')"
```

Or with a prefix:

```tmux
set -g status-right "♫ #(scribbles now 2>/dev/null || echo 'Not playing')"
```

Update interval (in seconds):

```tmux
set -g status-interval 5
```

## How Scrobbling Works

Scribbles follows the official Last.fm scrobbling rules:

1. **Track must be longer than 30 seconds**
2. **Track must be played for at least**:
   - 50% of its duration, OR
   - 4 minutes (whichever comes first)

Examples:
- 3 minute track: scrobbles at 1:30
- 10 minute track: scrobbles at 4:00
- 20 second track: never scrobbles (too short)

### Handling Edge Cases

- **Pause/Resume**: Playback time is accumulated across pauses
- **Skip**: If you skip before the threshold, the track is not scrobbled
- **Repeat**: Each play of the same track is scrobbled separately
- **Offline**: Scrobbles are queued and submitted when online

## Data Storage

- **Config**: `~/.config/scribbles/config.yaml`
- **State**: `~/.local/share/scribbles/state.json` (daemon runtime state)
- **Queue**: `~/.local/share/scribbles/queue.db` (SQLite database for
  scrobble queue)
- **Logs**: `~/.local/share/scribbles/logs/` (when running via launchd)

## Troubleshooting

### Daemon not scrobbling

1. Check if the daemon is running:
   ```bash
   launchctl list | grep scribbles
   ```

2. Check the logs:
   ```bash
   tail -f ~/.local/share/scribbles/logs/scribbles.log
   tail -f ~/.local/share/scribbles/logs/scribbles.err
   ```

3. Verify Last.fm credentials:
   ```bash
   cat ~/.config/scribbles/config.yaml
   ```

4. Restart the daemon:
   ```bash
   scribbles uninstall
   scribbles install
   ```

### Scrobbles not appearing on Last.fm

- Check that tracks are longer than 30 seconds
- Verify you're playing tracks for at least 50% or 4 minutes
- Check Last.fm is online and accepting scrobbles
- Look for errors in the daemon logs

### "Music app not running" error

The daemon requires Apple Music to be running. Start Music and the daemon
will automatically detect it.

### High CPU usage

- Increase the poll interval in the config (default: 3 seconds)
- Check for errors in the logs that might cause rapid retries

### `scribbles now` is slow

The `now` command calls AppleScript to query Apple Music, which takes
~200-300ms. This is unavoidable due to macOS limitations. For best
performance:
- Set tmux `status-interval` to 5 seconds or more
- Ensure Apple Music is running (faster when app is active)

## Development

### Building

```bash
go build -o scribbles .
```

### Running Tests

```bash
# Unit tests only
go test ./...

# With integration tests (requires Last.fm credentials)
go test -tags=integration ./...
```

### Project Structure

```
scribbles/
├── cmd/                    # CLI commands (Cobra)
│   ├── root.go
│   ├── daemon.go
│   ├── now.go
│   ├── auth.go
│   ├── install.go
│   └── uninstall.go
├── internal/
│   ├── music/              # Apple Music client
│   │   ├── client.go       # Interface
│   │   └── applescript.go  # AppleScript implementation
│   ├── scrobbler/          # Last.fm client
│   │   ├── client.go       # Last.fm API wrapper
│   │   ├── queue.go        # SQLite scrobble queue
│   │   └── rules.go        # Scrobbling rules
│   ├── daemon/             # Daemon implementation
│   │   ├── daemon.go       # Main daemon loop
│   │   ├── state.go        # Track state management
│   │   ├── poller.go       # Music polling
│   │   └── launchd.go      # launchd plist generation
│   └── config/             # Configuration
│       └── config.go
├── go.mod
├── go.sum
└── README.md
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Credits

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management
- [lastfm-go](https://github.com/shkh/lastfm-go) - Last.fm API client
- [zerolog](https://github.com/rs/zerolog) - Structured logging
- [modernc.org/sqlite](https://modernc.org/sqlite) - Pure Go SQLite

## Related Projects

- [rescrobbled](https://github.com/InputUsername/rescrobbled) - MPRIS
  scrobbler for Linux
- [pScrobbler](https://github.com/patrickklaeren/pScrobbler) - Plex
  scrobbler
- [mpdscribble](https://github.com/MusicPlayerDaemon/mpdscribble) - MPD
  scrobbler
