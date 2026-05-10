package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"burpbridge-core/pkg/engine"
)

func main() {
	fmt.Println("BurpBridge Desktop Tester")
	fmt.Println("=========================")

	r, w := io.Pipe()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := r.Read(buf)
			if err != nil {
				return
			}
			log.Printf("Received %d bytes from engine", n)
		}
	}()

	eng, err := engine.StartEngine(w, "127.0.0.1:8080")
	if err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}

	log.Println("Engine started, press Ctrl+C to stop")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	eng.Stop()
	log.Println("Shutdown complete")
}
