.PHONY: build build-android build-ios clean help setup

help:
	@echo "BurpBridge Build System"
	@echo "\033[0;32m======================\033[0m"
	@echo ""
	@echo "Available targets:"
	@echo "  \033[0;34msetup\033[0m          - Install gomobile and dependencies (first-time only)"
	@echo "  \033[0;34mbuild\033[0m          - Build both Android and iOS libraries"
	@echo "  \033[0;34mbuild-android\033[0m - Build Android .aar library"
	@echo "  \033[0;34mbuild-ios\033[0m     - Build iOS .xcframework library"
	@echo "  \033[0;34mclean\033[0m          - Remove build artifacts"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - Go 1.25+"
	@echo "  - Android NDK 21+ (for Android builds)"
	@echo "  - Xcode (for iOS builds)"
	@echo "  - Run 'make setup' before first build"
	@echo ""

setup:
	@echo "\033[0;34mInstalling gomobile...\033[0m"
	go install golang.org/x/mobile/cmd/gomobile@latest
	@echo "\033[0;34mGetting mobile bind package...\033[0m"
	go get golang.org/x/mobile/bind
	@echo "\033[0;34mInitializing gomobile...\033[0m"
	gomobile init
	@echo ""
	@echo "\033[0;32mSetup complete! Run 'make build' to build the libraries.\033[0m"

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