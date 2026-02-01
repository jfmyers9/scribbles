# scribbles v0.2.0

**Release Date**: 2026-01-31

## What's New in v0.2.0

This release focuses on improving tmux status bar integration with fixed-width output and marquee scrolling features.

### New Features

#### Fixed-Width Output
- **`--width` flag**: Set a fixed display width for `scribbles now` output
- **`output_width` config option**: Configure default width in config file
- **Smart truncation**: Long text truncated with "..." indicator
- **Space padding**: Short text padded to exact width
- **Unicode-aware**: Correctly handles emoji (🎵) and CJK characters
- **Stable status bars**: Prevents layout shifts in tmux/status bars

#### Marquee Scrolling
- **`--marquee` flag**: Enable scrolling text for long track names
- **`marquee_enabled` config option**: Enable marquee globally
- **Timestamp-based**: Deterministic scrolling based on current time
- **Configurable speed**: `marquee_speed` (default: 2 chars/second)
- **Custom separator**: `marquee_separator` (default: " • ")
- **Seamless loop**: Text scrolls continuously showing full title over time

### Configuration

New configuration options in `~/.config/scribbles/config.yaml`:

```yaml
# Fixed output width for the "now" command (0=disabled)
# Useful for tmux status bars to prevent layout shifts
output_width: 25

# Enable marquee scrolling for long track names
marquee_enabled: false

# Marquee scroll speed (characters per second)
marquee_speed: 2

# Separator between repetitions in marquee
marquee_separator: " • "
```

### Usage Examples

#### Fixed-Width Output

```bash
# Command line flag
scribbles now --width 25

# In tmux.conf
set -g status-right "♫ #(scribbles now --width 25 2>/dev/null)"
```

#### Marquee Scrolling

```bash
# Enable marquee with flag
scribbles now --width 25 --marquee

# In tmux.conf with 5-second refresh
set -g status-right "♫ #(scribbles now --width 25 --marquee 2>/dev/null)"
set -g status-interval 5
```

### Tmux Integration Guide

**Before v0.2.0** (variable width, layout shifts):
```tmux
set -g status-right "♫ #(scribbles now 2>/dev/null | cut -c1-25)"
# Problem: Short songs show less than 25 chars, causing shifts
```

**After v0.2.0** (fixed width, stable layout):
```tmux
# Option 1: Fixed width with truncation
set -g status-right "♫ #(scribbles now --width 25 2>/dev/null)"

# Option 2: Marquee scrolling for long titles
set -g status-right "♫ #(scribbles now --width 25 --marquee 2>/dev/null)"
set -g status-interval 5
```

### Technical Details

- **Unicode width calculation**: Uses `github.com/mattn/go-runewidth` library
- **Marquee algorithm**: Timestamp-based modulo positioning for stateless scrolling
- **Comprehensive testing**: 36 test cases covering all edge cases
- **Backward compatible**: All flags default to disabled (0/false)

### What's Included

This release maintains all core scribbling functionality from v0.1.0:

- Automatic scrobbling to Last.fm
- Background daemon with launchd integration
- Last.fm rules compliance (50% duration or 4 minutes)
- Offline queue with retry logic
- State persistence across daemon restarts
- CLI for querying current track

### Installation

Download the appropriate binary for your system:
- **Apple Silicon (M1/M2/M3/M4)**: `scribbles-v0.2.0-darwin-arm64.tar.gz`
- **Intel**: `scribbles-v0.2.0-darwin-amd64.tar.gz`
- **Universal** (works on both): `scribbles-v0.2.0-darwin-universal.tar.gz`

```bash
tar -xzf scribbles-v0.2.0-darwin-universal.tar.gz
sudo cp scribbles /usr/local/bin/
scribbles --version
```

### Checksums

Verify download integrity:

```bash
shasum -a 256 -c scribbles-v0.2.0-darwin-universal.sha256
```

### System Requirements

- macOS 10.15 (Catalina) or later
- Apple Music app
- Last.fm account and API credentials

### Changelog

**New Features:**
- Add `--width` flag and `output_width` config for fixed-width output
- Add `--marquee` flag and marquee config options for scrolling text
- Add Unicode-aware width calculation using go-runewidth
- Add comprehensive test suite (36 tests) for padding and marquee

**Documentation:**
- Add fixed-width output examples to README
- Add marquee scrolling guide with tmux integration
- Add inline code documentation for new functions
- Add configuration examples for all new options

**Dependencies:**
- Add `github.com/mattn/go-runewidth v0.0.19` for Unicode support

### Contributors

- Jim Myers (@jfmyers9)

### License

See LICENSE file for details.

---

**Full Changelog**: https://github.com/jfmyers9/scribbles/compare/v0.1.0...v0.2.0
