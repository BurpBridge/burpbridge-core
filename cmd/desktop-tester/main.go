package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"burpbridge-core/pkg/engine"

	"github.com/songgao/water"
)

func main() {
	fmt.Println("BurpBridge Desktop Tester")
	fmt.Println("=========================")

	ifce, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		log.Fatalf("Failed to create TUN interface: %v", err)
	}

	ifceName := ifce.Name()
	fmt.Printf("Created TUN interface: %s\n", ifceName)

	if err := runCommand("ifconfig", ifceName, "10.0.0.1", "10.0.0.1", "up"); err != nil {
		log.Printf("Warning: Failed to set IP: %v", err)
	}

	eng, err := engine.StartEngine(ifce, "127.0.0.1:8080")
	if err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}

	log.Println("Engine started, press Ctrl+C to stop")
	log.Printf("Tunnel: %s -> Burp at 127.0.0.1:8080", ifceName)

	log.Println("")
	log.Println("=== ROUTING INSTRUCTIONS ===")
	log.Println("To test the tunnel, open another terminal and run:")
	log.Println("")
	log.Printf("  sudo route add 1.1.1.1 10.0.0.1   # Add test route\n")
	log.Printf("  curl http://1.1.1.1              # Test the connection\n")
	log.Printf("  sudo route delete 1.1.1.1         # Remove route when done\n")
	log.Println("")
	log.Println("IMPORTANT: In Burp Suite, enable 'Invisible proxying' in Proxy -> Proxy Settings")
	log.Println("===========================")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	err = eng.Stop()
	if err != nil {
		return
	}

	if err := ifce.Close(); err != nil {
		log.Printf("Error closing TUN interface: %v", err)
	}

	log.Println("Shutdown complete")
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
