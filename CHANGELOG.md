# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic
Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.0] - 2026-02-16

### Added

- Discord Rich Presence integration
  - Shows currently playing track with artist, album, and artwork
  - Album artwork lookup via iTunes Search API with negative cache TTL
  - Song-title fallback when album artwork is unavailable
  - Configurable via `discord.enabled` and `discord.application_id`
  - Auto-connects/reconnects to Discord IPC socket

### Fixed

- TUI layout corruption and stale content on song switch
- TUI stutter and degradation during playback
- Discord artwork lookup with song fallback and negative cache TTL
- Discord unchecked error returns flagged by errcheck

### Changed

- Bumped Go to 1.26
- Bumped golangci-lint action to v7
- Applied gofmt -s simplifications

## [0.4.0] - 2026-02-04

### Added

- Terminal UI for now playing visualization
  - `scribbles tui` - Standalone TUI for quick visualization
  - `scribbles daemon --tui` - Integrated TUI with scrobble tracking
  - Now playing display with track name, artist, album, and play state
  - Progress bar with real-time position
  - Scrobble progress indicator showing percentage toward scrobble threshold
  - Recent tracks panel with scrobble status
  - Session stats display
  - Keyboard controls: `q` (quit), `space` (play/pause), `n` (next), `p` (prev)
  - Configurable via `tui.enabled`, `tui.refresh_rate`, `tui.theme`
- Music control commands for Apple Music playback control
  - `scribbles play` - Resume playback
  - `scribbles pause` - Pause playback
  - `scribbles playpause` - Toggle play/pause
  - `scribbles next` - Skip to next track
  - `scribbles prev` - Go to previous track
  - `scribbles shuffle [on|off]` - Set shuffle mode
  - `scribbles volume [0-100]` - Set playback volume
- Tmux integration examples for keyboard-controlled music playback
- Control methods in `AppleScriptClient` for playback control

### Dependencies

- Added `github.com/rivo/tview` for terminal UI
- Added `github.com/gdamore/tcell/v2` for terminal handling

## [0.2.0] - 2026-01-31

### Added

- Fixed-width output with `--width` flag for stable tmux status bars
- `output_width` config option for setting default display width
- Marquee scrolling with `--marquee` flag for long track names
- `marquee_enabled`, `marquee_speed`, and `marquee_separator` config options
- Unicode-aware width calculation using `github.com/mattn/go-runewidth`
- Smart truncation with "..." indicator for long text
- Space padding to maintain exact width for short text
- Timestamp-based deterministic scrolling algorithm
- Comprehensive test suite with 36 test cases for padding and marquee

### Changed

- Improved tmux integration with fixed-width output preventing layout shifts

## [0.1.0] - 2026-01-31

### Added

- Initial release with core scrobbling functionality
- Automatic scrobbling to Last.fm with rules compliance (50% duration or 4
  minutes)
- Background daemon with launchd integration
- `scribbles daemon` command to run scrobbling daemon
- `scribbles now` command to display currently playing track
- `scribbles auth` command for Last.fm authentication
- `scribbles install` command to install and start background daemon
- `scribbles uninstall` command to stop and remove background daemon
- Pause/resume tracking with correct timer handling
- Offline queue with SQLite backend for scrobbles when Last.fm is unreachable
- Automatic retry logic for failed scrobbles
- State persistence across daemon restarts
- Configuration management with Viper
- Structured logging with zerolog
- CLI integration for tmux status lines and other displays
- Support for custom output formats with Go templates

[unreleased]: https://github.com/jfmyers9/scribbles/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/jfmyers9/scribbles/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/jfmyers9/scribbles/compare/v0.3.0...v0.4.0
[0.2.0]: https://github.com/jfmyers9/scribbles/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/jfmyers9/scribbles/releases/tag/v0.1.0
