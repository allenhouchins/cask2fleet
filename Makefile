.PHONY: build run clean test help

# Default target
all: build

# Build the program
build:
	@echo "Building generate_fleet_yaml..."
	go build -o generate_fleet_yaml main.go
	@echo "Build complete!"

# Run the program
run: build
	@echo "Running generate_fleet_yaml..."
	./generate_fleet_yaml

# Run directly with Go (no build step)
go-run:
	@echo "Running with go run..."
	go run main.go

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f generate_fleet_yaml
	rm -rf fleet_yaml_files/
	@echo "Clean complete!"

# Run tests (if any are added later)
test:
	@echo "Running tests..."
	go test ./...

# Build for different platforms
build-all: build-macos build-linux build-windows

build-macos:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 go build -o generate_fleet_yaml-macos-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -o generate_fleet_yaml-macos-arm64 main.go

build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -o generate_fleet_yaml-linux-amd64 main.go
	GOOS=linux GOARCH=arm64 go build -o generate_fleet_yaml-linux-arm64 main.go

build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build -o generate_fleet_yaml-windows-amd64.exe main.go

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	@echo "Dependencies installed!"

# Show help
help:
	@echo "Available targets:"
	@echo "  build      - Build the program"
	@echo "  run        - Build and run the program"
	@echo "  go-run     - Run directly with go run"
	@echo "  clean      - Clean build artifacts and output"
	@echo "  test       - Run tests"
	@echo "  build-all  - Build for all platforms"
	@echo "  deps       - Install dependencies"
	@echo "  help       - Show this help message" 