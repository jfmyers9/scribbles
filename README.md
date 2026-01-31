# scribbles

A lightweight Apple Music to Last.fm scrobbler for the terminal.

## What it does

scribbles monitors Apple Music playback and automatically scrobbles
tracks to Last.fm. It runs as a background daemon and provides a CLI
for displaying the currently playing track.

**Features:**

- Automatic Last.fm scrobbling from Apple Music
- Display current track info for tmux status lines
- Runs silently in the background
- Handles pauses, skips, and repeats correctly
- No GUI required

## Installation

```bash
go install github.com/jfmyers9/scribbles@latest
```

Or build from source:

```bash
git clone https://github.com/jfmyers9/scribbles.git
cd scribbles
go build
```

## Setup

### 1. Authenticate with Last.fm

Before running the daemon, authenticate with your Last.fm account:

```bash
scribbles auth
```

This will guide you through the Last.fm authentication flow and store
your session credentials in `~/.config/scribbles/`.

### 2. Install the daemon

Install the daemon to run at startup:

```bash
scribbles install
```

This sets up the background service to automatically scrobble tracks.

## Usage

### Display current track

```bash
scribbles now
```

Shows the currently playing track. Exit codes:
- `0` - Track is playing
- `1` - No track playing, paused, or Music app not running

**Custom format:**

```bash
scribbles now --format "{{.Artist}} - {{.Name}}"
```

Available template fields: `.Name`, `.Artist`, `.Album`, `.Duration`,
`.Position`

### Run the daemon manually

```bash
scribbles daemon
```

Normally the daemon runs automatically via the install command.

## Configuration

Configuration is stored in `~/.config/scribbles/config.yaml`.

Example:

```yaml
output_format: "{{.Artist}} - {{.Name}}"
```

## How it works

- Polls Apple Music every few seconds using AppleScript
- Tracks playback state and timestamps
- Submits scrobbles to Last.fm according to their rules:
  - Track must play for at least 50% of duration OR 4 minutes
  - Tracks under 30 seconds are not scrobbled

## License

See LICENSE file.
