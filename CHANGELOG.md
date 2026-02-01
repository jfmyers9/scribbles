# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic
Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[unreleased]: https://github.com/jfmyers9/scribbles/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/jfmyers9/scribbles/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/jfmyers9/scribbles/releases/tag/v0.1.0
