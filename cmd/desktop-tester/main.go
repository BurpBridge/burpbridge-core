package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"burpbridge-core/pkg/engine"
)

type MockTun struct {
	reader io.Reader
	writer io.Writer
	once   sync.Once
	close  chan struct{}
}

func NewMockTun() *MockTun {
	r, w := io.Pipe()
	return &MockTun{
		reader: r,
		writer: w,
		close:  make(chan struct{}),
	}
}

func (m *MockTun) Read(p []byte) (n int, err error) {
	select {
	case <-m.close:
		return 0, io.EOF
	default:
		return m.reader.Read(p)
	}
}

func (m *MockTun) Write(p []byte) (n int, err error) {
	select {
	case <-m.close:
		return 0, io.EOF
	default:
		return m.writer.Write(p)
	}
}

func (m *MockTun) Close() error {
	m.once.Do(func() {
		close(m.close)
	})
	return nil
}

func main() {
	fmt.Println("BurpBridge Desktop Tester")
	fmt.Println("=========================")

	tun := NewMockTun()

	eng, err := engine.StartEngine(tun, "127.0.0.1:8080")
	if err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}

	log.Println("Engine started, press Ctrl+C to stop")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	eng.Stop()
	tun.Close()
	log.Println("Shutdown complete")
}
