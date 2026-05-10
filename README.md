# BurpBridge

A transparent proxy engine that intercepts mobile network traffic and forwards it to Burp Suite for security testing.

## Overview

BurpBridge provides a Go-based networking engine that can be embedded in Android and iOS applications. It creates a local VPN-like interface that captures all network traffic, transparently relays TCP/UDP connections to a local Burp Suite proxy, and enables comprehensive mobile app security testing without requiring a system-wide VPN.

## Architecture

```
┌─────────────────┐     ┌─────────────────┐       ┌──────────────────┐
│  Mobile App     │     │  BurpBridge     │       │  Burp Suite      │
│  (TUN Interface)──────▶│  (Go Engine)   │──────▶│  (127.0.0.1:8080)│
└─────────────────┘     └─────────────────┘       └──────────────────┘
                              │
                              ▼
                     ┌─────────────────┐
                     │  gVisor Stack   │
                     │  (TCP/UDP Relay)│
                     └─────────────────┘
```

- **Mobile App**: Creates a TUN interface and passes file descriptor to BurpBridge
- **BurpBridge Engine**: User-space TCP/IP stack using gVisor
- **Relay**: Transparent TCP/UDP forwarding to Burp Suite
- **Burp Suite**: Receives all traffic for inspection (with "Invisible proxying" enabled)

## Quick Start

### Prerequisites

- Go 1.25 or later
- Android NDK 21+ (for Android builds)
- Xcode (for iOS builds)

### First-Time Setup

```bash
make setup
```

### Build

```bash
# Build both platforms
make build

# Or build individually
make build-android
make build-ios
```

### Outputs

| Platform | Output | Location |
|----------|--------|----------|
| Android | `.aar` | `build/burpbridge.aar` |
| iOS | `.xcframework` | `build/BurpBridge.xcframework` |

## Key Constraints

### iOS Memory Limit
- iOS Network Extensions are killed at ~15MB memory usage
- The engine uses `runtime/debug.SetMemoryLimit()` to cap memory at ~10MB
- Binary size can be larger; runtime memory is what matters

### Burp Suite Configuration
- Enable **Invisible proxying** in Burp Suite settings
- This allows proxying to non-proxy-aware clients

## Project Structure

```
burpbridge-core/
├── cmd/
│   └── desktop-tester/      # CLI harness for desktop testing
│       └── main.go
├── pkg/
│   ├── engine/               # Core networking engine (pure Go)
│   │   └── engine.go
│   └── mobile/              # Mobile bindings (gomobile)
│       └── bind.go
├── Makefile
├── README.md
├── DEVELOPMENT.md
└── CONTRIBUTING.md
```

## Documentation

- [Development Guide](DEVELOPMENT.md) - Setup and development workflow
- [Contributing Guide](CONTRIBUTING.md) - How to contribute to the project

## Usage

### Android (Kotlin)

```kotlin
// Start proxy with TUN file descriptor
val result = BurpBridge.startAndroidProxy(tunFd, "127.0.0.1:8080")
if (result.isSuccess) {
    Log.d("BurpBridge", "Proxy started")
}

// Stop when done
BurpBridge.stopProxy()
```

### iOS (Swift)

```swift
// Start proxy (implementation pending packetFlow integration)
BurpBridge.startIOSProxy("127.0.0.1:8080") { result in
    switch result {
    case .success:
        print("Proxy started")
    case .failure(let error):
        print("Error: \(error)")
    }
}
```

## License

MIT License - See LICENSE file for details