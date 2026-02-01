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

## Last.fm SDK

This project includes a modern, reusable Last.fm API client library at
`pkg/lastfm/`. It can be used independently in your own Go projects.

### Installation

```bash
go get github.com/jfmyers9/scribbles/pkg/lastfm
```

### Quick Example

```go
import "github.com/jfmyers9/scribbles/pkg/lastfm"

// Create client
client, err := lastfm.NewClient(lastfm.Config{
    APIKey:    "your-api-key",
    APISecret: "your-api-secret",
})

// Authenticate
token, _ := client.Auth().GetToken(ctx)
fmt.Println("Visit:", client.Auth().GetAuthURL(token.Token))
session, _ := client.Auth().GetSession(ctx, token.Token)
client.SetSessionKey(session.Key)

// Scrobble
track := lastfm.Track{
    Artist: "The Beatles",
    Track:  "Yesterday",
}
client.Scrobble().Scrobble(ctx, track, time.Now())
```

### Features

- Clean, type-safe API with context support
- Automatic retry with exponential backoff
- Batch scrobbling (up to 50 tracks)
- Structured error types
- Comprehensive godoc and examples
- Zero dependencies outside stdlib

For detailed documentation, see
[pkg/lastfm/README.md](pkg/lastfm/README.md) or view on
[pkg.go.dev](https://pkg.go.dev/github.com/jfmyers9/scribbles/pkg/lastfm).

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

# Fixed output width for the "now" command (0=disabled)
# Useful for tmux status bars to prevent layout shifts
output_width: 0

# Marquee scrolling for long track names (requires output_width > 0)
marquee_enabled: false      # Enable marquee scrolling
marquee_speed: 2            # Scroll speed in characters per second
marquee_separator: " • "    # Separator between text repetitions

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
- `--width <n>`: Set fixed output width (0=disabled, overrides config)
- `--marquee`: Enable marquee scrolling for long text (requires --width)

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

# Fixed width output (useful for tmux status bars)
scribbles now --width 30
# Output: Artist Name - Track Name
# (padded to exactly 30 characters)

# Truncate long output
scribbles now --width 20
# Output: Artist Name - Tra...
# (truncated with "..." if longer than 20 characters)

# Marquee scrolling for long text
scribbles now --width 25 --marquee
# Output: (scrolls left to reveal full text over time)
# t=0s:  Artist Name - Very L
# t=5s:  y Long Track Name • A
# t=10s: Track Name • Artist N
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

### Fixed-Width Output for Stable Status Bars

To prevent the status bar from shifting as track names change length, use the
`--width` flag or set `output_width` in your config:

```tmux
# Using the --width flag (25 characters fixed width)
set -g status-right "♫ #(scribbles now --width 25 2>/dev/null || echo '—                       ')"
set -g status-interval 5
```

Or configure it globally in `~/.config/scribbles/config.yaml`:

```yaml
output_format: "🎵 {{.Name}} - {{.Artist}}"
output_width: 25
```

Then use in tmux:

```tmux
set -g status-right "#(scribbles now 2>/dev/null || echo '—                       ')"
```

The width is measured in display columns, accounting for Unicode characters
like emoji. When output is longer than the specified width, it's truncated with
"...". When shorter, it's padded with spaces.

### Marquee Scrolling for Long Track Names

For track names longer than the fixed width, you can enable marquee scrolling
to reveal the full text over time instead of truncating it:

```tmux
# Enable marquee scrolling with the --marquee flag
set -g status-right "♫ #(scribbles now --width 25 --marquee 2>/dev/null || echo '—                       ')"
set -g status-interval 5
```

Or configure it globally in `~/.config/scribbles/config.yaml`:

```yaml
output_format: "🎵 {{.Name}} - {{.Artist}}"
output_width: 25
marquee_enabled: true
marquee_speed: 2
marquee_separator: " • "
```

Then use in tmux:

```tmux
set -g status-right "#(scribbles now 2>/dev/null || echo '—                       ')"
```

#### How Marquee Scrolling Works

- **Short text** (fits within width): Displayed statically with padding (no
  scrolling)
- **Long text** (exceeds width): Scrolls left to reveal the full text over time
- **Continuous loop**: Text wraps around with a separator (default: " • ")
- **Deterministic**: Same timestamp produces same output (consistent across tmux
  refreshes)

#### Configuration Options

- `marquee_enabled` (boolean, default: `false`): Enable marquee scrolling
  globally
- `marquee_speed` (integer, default: `2`): Scroll speed in characters per
  second
  - With tmux `status-interval: 5`, speed 2 = 10 characters per refresh
  - Higher values scroll faster but may be harder to read
  - Lower values scroll slower but are more readable
- `marquee_separator` (string, default: `" • "`): Separator shown between the
  end and beginning of the text

#### Speed Tuning

The visual scrolling effect depends on your tmux `status-interval`:

- **status-interval: 5s**, **speed: 2** → 10 chars per update (recommended)
- **status-interval: 5s**, **speed: 1** → 5 chars per update (slower, more
  readable)
- **status-interval: 5s**, **speed: 3** → 15 chars per update (faster, less
  readable)

Adjust `marquee_speed` based on your preferred refresh interval for optimal
readability.

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
- [zerolog](https://github.com/rs/zerolog) - Structured logging
- [modernc.org/sqlite](https://modernc.org/sqlite) - Pure Go SQLite

## Related Projects

- [rescrobbled](https://github.com/InputUsername/rescrobbled) - MPRIS
  scrobbler for Linux
- [pScrobbler](https://github.com/patrickklaeren/pScrobbler) - Plex
  scrobbler
- [mpdscribble](https://github.com/MusicPlayerDaemon/mpdscribble) - MPD
  scrobbler
