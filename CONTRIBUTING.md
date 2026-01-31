# Contributing to scribbles

Thank you for your interest in contributing to scribbles! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- Go 1.23 or later
- macOS (for testing AppleScript integration)
- Apple Music app
- Last.fm account and API credentials (for integration testing)

### Getting Started

1. Fork the repository
2. Clone your fork:

```bash
git clone https://github.com/YOUR_USERNAME/scribbles.git
cd scribbles
```

3. Install dependencies:

```bash
go mod download
```

4. Build the project:

```bash
go build
```

## Project Structure

```
scribbles/
├── cmd/                    # Cobra command implementations
│   ├── root.go            # Root command with version info
│   ├── daemon.go          # Daemon command
│   ├── now.go             # Now playing command
│   ├── auth.go            # Authentication command
│   ├── install.go         # Installation command
│   └── uninstall.go       # Uninstallation command
├── internal/              # Internal packages
│   ├── music/             # Apple Music client
│   ├── scrobbler/         # Last.fm client and queue
│   ├── daemon/            # Daemon logic and state
│   └── config/            # Configuration management
├── scripts/               # Build and installation scripts
├── .github/workflows/     # GitHub Actions CI/CD
└── main.go               # Entry point
```

## Making Changes

### Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Use meaningful variable and function names
- Keep functions small and focused
- Write tests for new functionality

### Testing

Run unit tests:

```bash
go test -v -race ./...
```

Run integration tests (requires Apple Music and Last.fm setup):

```bash
go test -v -tags=integration ./...
```

Generate coverage report:

```bash
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Building

Build for your current platform:

```bash
go build
```

Build for all platforms:

```bash
./scripts/build.sh
```

### Linting

We use golangci-lint for code quality. Install it:

```bash
brew install golangci-lint
```

Run linter:

```bash
golangci-lint run
```

Or use the Makefile:

```bash
make lint
```

## Pull Request Process

1. Create a new branch for your changes:

```bash
git checkout -b feature/your-feature-name
```

2. Make your changes and commit with clear messages:

```bash
git commit -m "Add feature: description of what you added"
```

3. Push to your fork:

```bash
git push origin feature/your-feature-name
```

4. Open a Pull Request with:
   - Clear description of changes
   - Reference to any related issues
   - Test results
   - Any breaking changes noted

5. Wait for review and address feedback

## Commit Message Guidelines

Use conventional commit format:

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `test:` - Test changes
- `refactor:` - Code refactoring
- `chore:` - Build/tooling changes

Examples:

```
feat: add support for custom scrobble threshold
fix: handle AppleScript errors when Music app is not running
docs: update README with installation instructions
test: add integration tests for daemon lifecycle
```

## Testing Guidelines

### Unit Tests

- Test all public functions
- Use table-driven tests where appropriate
- Mock external dependencies (AppleScript, Last.fm API)
- Test error cases

Example:

```go
func TestShouldScrobble(t *testing.T) {
    tests := []struct {
        name           string
        duration       time.Duration
        played         time.Duration
        expectScrobble bool
    }{
        {"50% of 4min track", 4 * time.Minute, 2 * time.Minute, true},
        {"too short", 20 * time.Second, 15 * time.Second, false},
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := ShouldScrobble(tt.duration, tt.played)
            if result != tt.expectScrobble {
                t.Errorf("got %v, want %v", result, tt.expectScrobble)
            }
        })
    }
}
```

### Integration Tests

- Use `//go:build integration` tag
- Test against real Apple Music (with warnings)
- Clean up resources after tests

## Documentation

- Update README.md for user-facing changes
- Add godoc comments for exported functions
- Update RELEASE_NOTES.md for releases
- Include code examples in documentation

## Reporting Issues

When reporting bugs, please include:

- scribbles version (`scribbles --version`)
- macOS version
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs from `~/.local/share/scribbles/logs/`

## Feature Requests

We welcome feature requests! Please:

- Check existing issues first
- Describe the use case
- Explain why it would be useful
- Consider submitting a PR

## Code of Conduct

- Be respectful and inclusive
- Provide constructive feedback
- Help others learn and grow
- Keep discussions focused and on-topic

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (see LICENSE file).

## Questions?

Feel free to open an issue for questions or reach out to the maintainers.

Thank you for contributing! 🎵
