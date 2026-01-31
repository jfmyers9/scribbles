.PHONY: build clean test install uninstall release help

# Default target
.DEFAULT_GOAL := help

## build: Build binaries for all platforms
build:
	@./scripts/build.sh

## clean: Remove build artifacts
clean:
	@rm -rf ./dist
	@echo "✓ Build artifacts removed"

## test: Run all tests
test:
	@echo "Running unit tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@echo ""
	@echo "Test coverage:"
	@go tool cover -func=coverage.out | tail -1

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -v -tags=integration ./...

## install: Install scribbles locally
install: build
	@echo "Installing scribbles to /usr/local/bin/..."
	@cp ./dist/scribbles /usr/local/bin/scribbles
	@echo "✓ scribbles installed successfully"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Run 'scribbles auth' to authenticate with Last.fm"
	@echo "  2. Run 'scribbles install' to install the background daemon"

## uninstall: Uninstall scribbles and remove daemon
uninstall:
	@echo "Stopping and removing daemon..."
	@scribbles uninstall 2>/dev/null || true
	@echo "Removing binary..."
	@rm -f /usr/local/bin/scribbles
	@echo "✓ scribbles uninstalled successfully"

## release: Create a release (requires VERSION)
release:
	@if [ -z "$(VERSION)" ]; then \
		echo "ERROR: VERSION not set. Usage: make release VERSION=v1.0.0"; \
		exit 1; \
	fi
	@echo "Creating release $(VERSION)..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@git push origin $(VERSION)
	@./scripts/build.sh
	@echo ""
	@echo "✓ Release $(VERSION) tagged and built"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Create GitHub release at: https://github.com/jfmyers9/scribbles/releases/new?tag=$(VERSION)"
	@echo "  2. Upload artifacts from ./dist/"
	@echo "  3. Use RELEASE_NOTES.md as release description"

## lint: Run linters
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install from: https://golangci-lint.run/welcome/install/" && exit 1)
	@golangci-lint run

## fmt: Format code
fmt:
	@go fmt ./...
	@echo "✓ Code formatted"

## help: Show this help message
help:
	@echo "scribbles - Apple Music scrobbler for Last.fm"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/  /'
