package mobile

import (
	"fmt"
	"os"

	"burpbridge-core/pkg/engine"
)

var currentEngine *engine.Engine

func StartAndroidProxy(fd int, burpAddr string) error {
	if currentEngine != nil {
		return fmt.Errorf("proxy already running, call StopProxy() first")
	}

	if fd < 0 {
		return fmt.Errorf("invalid file descriptor: %d", fd)
	}

	f := os.NewFile(uintptr(fd), "tun")
	if f == nil {
		return fmt.Errorf("failed to create file from fd: %d", fd)
	}

	eng, err := engine.StartEngine(f, burpAddr)
	if err != nil {
		return fmt.Errorf("failed to start engine: %w", err)
	}

	currentEngine = eng
	return nil
}

func StartIOSProxy(burpAddr string) error {
	return fmt.Errorf("iOS binding not yet implemented - requires packetFlow integration")
}

func StopProxy() error {
	if currentEngine == nil {
		return fmt.Errorf("no proxy running")
	}

	err := currentEngine.Stop()
	currentEngine = nil
	return err
}

func IsProxyRunning() bool {
	return currentEngine != nil
}
