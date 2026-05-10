.PHONY: build build-android build-ios clean help setup

help:
	@echo "BurpBridge Build System"
	@echo "======================"
	@echo ""
	@echo "Available targets:"
	@echo "  setup          - Install gomobile and dependencies (first-time only)"
	@echo "  build          - Build both Android and iOS libraries"
	@echo "  build-android - Build Android .aar library"
	@echo "  build-ios     - Build iOS .xcframework library"
	@echo "  clean          - Remove build artifacts"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - Go 1.25+"
	@echo "  - Android NDK 21+ (for Android builds)"
	@echo "  - Xcode (for iOS builds)"
	@echo "  - Run 'make setup' before first build"
	@echo ""

setup:
	@echo "Installing gomobile..."
	go install golang.org/x/mobile/cmd/gomobile@latest
	@echo "Getting mobile bind package..."
	go get golang.org/x/mobile/bind
	@echo "Initializing gomobile..."
	gomobile init
	@echo ""
	@echo "Setup complete! Run 'make build' to build the libraries."

build: build-android build-ios
	@echo ""
	@echo "======================"
	@echo "Build complete!"
	@echo "  Android: build/burpbridge.aar"
	@echo "  iOS:     build/BurpBridge.xcframework"

build-android: clean
	@mkdir -p build
	@echo "Building Android AAR..."
	gomobile bind -target=android -androidapi=35 -o build/burpbridge.aar ./pkg/mobile
	@echo "Android build complete: build/burpbridge.aar"

build-ios: clean
	@mkdir -p build
	@echo "Building iOS XCFramework..."
	gomobile bind -target=ios -o build/BurpBridge.xcframework ./pkg/mobile
	@echo "iOS build complete: build/BurpBridge.xcframework"

clean:
	rm -rf build
	@echo "Cleaned build directory"