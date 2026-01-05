#!/bin/bash
# Development environment setup script

set -e

echo "Setting up UniEdit Server development environment..."

# Check Go version
GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
REQUIRED_VERSION="1.22"
if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "Error: Go version $REQUIRED_VERSION or higher is required (found $GO_VERSION)"
    exit 1
fi
echo "✓ Go version: $GO_VERSION"

# Install development tools
echo "Installing development tools..."

echo "  Installing wire..."
go install github.com/google/wire/cmd/wire@latest

echo "  Installing mage..."
go install github.com/magefile/mage@latest

echo "  Installing golangci-lint..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

echo "  Installing mockery..."
go install github.com/vektra/mockery/v2@latest

# Download dependencies
echo "Downloading Go dependencies..."
go mod download

# Generate wire code
echo "Generating wire code..."
mage wire

# Copy example config if not exists
if [ ! -f "config.yaml" ]; then
    echo "Creating config.yaml from template..."
    cp configs/config.example.yaml config.yaml
    echo "⚠️  Please update config.yaml with your settings"
fi

# Run tests
echo "Running tests..."
go test ./... -v

echo ""
echo "✓ Setup complete!"
echo ""
echo "Next steps:"
echo "  1. Update config.yaml with your database and Redis settings"
echo "  2. Run 'mage build' to build the server"
echo "  3. Run 'mage dev' to start the development server"
