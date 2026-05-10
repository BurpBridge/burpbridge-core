# Development Guide

This guide covers setting up your development environment and understanding the codebase.

## Prerequisites

### Required Software

| Software    | Version | Purpose                |
|-------------|---------|------------------------|
| Go          | 1.25+   | Language runtime       |
| Android NDK | 21+     | Android native builds  |
| Xcode       | Latest  | iOS builds             |
| gomobile    | Latest  | Go mobile binding tool |

### Installing Prerequisites

#### 1. Go

Download from [golang.org](https://golang.org/dl/) or use a package manager:

```bash
# macOS
brew install go

# Linux
sudo apt-get install golang-go
```

#### 2. gomobile

```bash
# Install gomobile
go install golang.org/x/mobile/cmd/gomobile@latest

# Get mobile bind package (required for gomobile bind)
go get golang.org/x/mobile/bind

# Initialize gomobile
gomobile init
```

#### 3. Android NDK

If building for Android, ensure you have the Android NDK installed:

```bash
# Check existing NDK
ls ~/Library/Android/sdk/ndk/

# If no ndk-bundle, create symlink to your NDK version
ln -sf ~/Library/Android/sdk/ndk/27.1.12297006 ~/Library/Android/sdk/ndk-bundle
```

#### 4. Xcode

Install Xcode from the App Store, then ensure command-line tools are available:

```bash
xcode-select --install
```

## Building the Project

### Quick Build

```bash
# First-time setup (only needed once)
make setup

# Build both platforms
make build
```

### Platform-Specific Builds

```bash
# Android only
make build-android

# iOS only
make build-ios
```

### Clean Build

```bash
make clean && make build
```

## Project Structure

```
burpbridge-core/
├── cmd/
│   └── desktop-tester/
│       └── main.go          # Desktop CLI for testing the engine
├── pkg/
│   ├── engine/
│   │   └── engine.go        # Core gVisor-based TCP/IP stack
│   └── mobile/
│       └── bind.go          # gomobile bindings for Android/iOS
├── Makefile                 # Build automation
└── go.mod                   # Go module definition
```

### Key Components

#### `pkg/engine/engine.go`

The core networking engine. Key functions:

- `StartEngine(rwc io.ReadWriteCloser, proxyAddr string) (*Engine, error)` - Initialize the engine
- `Stop() error` - Stop the engine gracefully
- Uses gVisor for user-space TCP/IP stack
- Implements transparent TCP/UDP relay to Burp Suite

#### `pkg/mobile/bind.go`

Mobile-specific bindings:

- `StartAndroidProxy(fd int, burpAddr string) error` - Android entry point
- `StartIOSProxy(burpAddr string) error` - iOS entry point (pending)
- `StopProxy() error` - Stop the proxy
- `IsProxyRunning() bool` - Check status

#### `cmd/desktop-tester/main.go`

Desktop testing harness with TUN interface support.

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test ./pkg/engine/
```

## Desktop Testing

The desktop tester allows you to test the engine locally using a TUN interface.

### macOS Testing

1. Build and run the desktop tester:

```bash
sudo go run cmd/desktop-tester/main.go
```

2. In a separate terminal, route traffic through the TUN:

```bash
# Route specific IP through TUN
sudo route add 1.1.1.1 10.0.0.1

# Test connectivity
curl --interface 10.0.0.2 http://example.com
```

3. Check Burp Suite for captured traffic

### Cleanup

```bash
# Remove route when done
sudo route delete 1.1.1.1
```

## Debugging Tips

### Enable Logging

The engine uses standard Go `log` package. Set `LOG_LEVEL` environment variable:

```bash
# Verbose logging
LOG_LEVEL=debug go run cmd/desktop-tester/main.go
```

### Memory Profiling

Check memory usage on iOS:

```go
import "runtime/debug"

// Set memory limit (must be called early in main)
debug.SetMemoryLimit(10 * 1024 * 1024) // 10MB
```

### Network Debugging

Use Wireshark or tcpdump to inspect TUN interface traffic:

```bash
# Capture on utun interface
sudo tcpdump -i utunX -w capture.pcap
```

## Common Issues

### gomobile Not Found

```bash
# Ensure Go bin is in PATH
export PATH=$PATH:$(go env GOPATH)/bin

# Verify installation
which gomobile
```

### NDK Not Found (Android)

```bash
# Check NDK location
ls ~/Library/Android/sdk/ndk/

# Create symlink if needed
ln -sf ~/Library/Android/sdk/ndk/YOUR_VERSION ~/Library/Android/sdk/ndk-bundle
```

### iOS Build Fails

- Ensure Xcode is installed and selected
- Check that you have accepted Xcode licenses:
  ```bash
  sudo xcodebuild -license
  ```

### Memory Limit Exceeded (iOS)

If your app is killed by iOS:

- Reduce `SetMemoryLimit` value
- Disable unused protocol handlers
- Minimize logging output

## IDE Setup

### VS Code

Install Go extension:

```bash
code --install-extension golang.go
```

### GoLand

Ensure Go SDK is configured in Project Structure settings.

## Next Steps

- Review [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines
- Explore the codebase in `pkg/engine/` and `pkg/mobile/`