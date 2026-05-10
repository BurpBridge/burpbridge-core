.PHONY: build build-android build-ios clean help

help:
	@echo "BurpBridge Build System"
	@echo "======================"
	@echo "Available targets:"
	@echo "  build          - Build both Android and iOS libraries"
	@echo "  build-android - Build Android .aar library"
	@echo "  build-ios     - Build iOS .xcframework library"
	@echo "  clean          - Remove build artifacts"
	@echo ""

build: build-android build-ios

build-android: clean
	@mkdir -p build
	@echo "Building Android AAR..."
	gomobile bind -target=android -o build/burpbridge.aar ./pkg/mobile
	@echo "Android build complete: build/burpbridge.aar"

build-ios: clean
	@mkdir -p build
	@echo "Building iOS XCFramework..."
	gomobile bind -target=ios -o build/BurpBridge.xcframework ./pkg/mobile
	@echo "iOS build complete: build/BurpBridge.xcframework"

clean:
	rm -rf build
	@echo "Cleaned build directory"