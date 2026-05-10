package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

type Engine struct {
	stack     *stack.Stack
	link      *TunLinkEndpoint
	proxyAddr string
	ctx       context.Context
	cancel    context.CancelFunc
}

type TunLinkEndpoint struct {
	rwc io.ReadWriteCloser

	mu     sync.Mutex
	closed bool

	inbound    chan buffer.View
	dispatcher stack.NetworkDispatcher
}

func NewTunLinkEndpoint(rwc io.ReadWriteCloser) *TunLinkEndpoint {
	return &TunLinkEndpoint{
		rwc:     rwc,
		inbound: make(chan buffer.View, 256),
	}
}

func (e *TunLinkEndpoint) ReadPacket() (stack.PacketBufferPtr, tcpip.NetworkProtocolNumber, error) {
	pkt, ok := <-e.inbound
	if !ok {
		return stack.PacketBufferPtr{}, 0, io.EOF
	}

	pb := stack.NewPacketBuffer(stack.PacketBufferOptions{
		ReserveHeaderSpace: 0,
	})
	pb.Write(pkt)

	return pb, ipv4.ProtocolNumber, nil
}

func (e *TunLinkEndpoint) WritePacket(tcpip.NetworkProtocolNumber, tcpip.Address, tcpip.Address, stack.PacketBufferPtr) error {
	return nil
}

func (e *TunLinkEndpoint) Attach(dispatcher stack.NetworkDispatcher) {
	e.dispatcher = dispatcher

	go e.readLoop()
	go e.dispatchLoop()
}

func (e *TunLinkEndpoint) readLoop() {
	buf := make([]byte, 65535)
	for {
		n, err := e.rwc.Read(buf)
		if err != nil {
			return
		}

		e.mu.Lock()
		if e.closed {
			e.mu.Unlock()
			return
		}

		select {
		case e.inbound <- buffer.NewViewFromSlice(buf[:n]):
		default:
			log.Println("inbound channel full, dropping packet")
		}
		e.mu.Unlock()
	}
}

func (e *TunLinkEndpoint) dispatchLoop() {
	for v := range e.inbound {
		if e.dispatcher == nil {
			continue
		}

		pb := stack.NewPacketBuffer(stack.PacketBufferOptions{
			ReserveHeaderSpace: 0,
		})
		pb.Write(v)

		e.dispatcher.DeliverNetworkPacket("", ipv4.ProtocolNumber, pb)
	}
}

func (e *TunLinkEndpoint) IsLoopback() bool {
	return false
}

func (e *TunLinkEndpoint) MTU() uint32 {
	return 1500
}

func (e *TunLinkEndpoint) Capabilities() stack.LinkEndpointCapabilities {
	return stack.CapabilityNone
}

func (e *TunLinkEndpoint) Wait() {}

func (e *TunLinkEndpoint) Close() {
	e.mu.Lock()
	e.closed = true
	e.mu.Unlock()
	close(e.inbound)
}

type ForwarderHandler struct {
	proxyAddr string
}

func (h *ForwarderHandler) HandleTCP(r *tcp.ForwarderRequest) {
	clientEp := r.CreateEndpoint()

	proxyConn, err := net.Dial("tcp", h.proxyAddr)
	if err != nil {
		log.Printf("Failed to connect to proxy %s: %v", h.proxyAddr, err)
		clientEp.Close()
		r.CompleteHijack(nil)
		return
	}

	r.CompleteHijack(nil)

	go h.relayTCP(clientEp, proxyConn)
	go h.relayTCP(proxyConn, clientEp)
}

func (h *ForwarderHandler) relayTCP(dst io.Writer, src io.ReadCloser) {
	defer dst.Close()
	_, err := io.CopyBuffer(dst, src, make([]byte, 4096))
	if err != nil && !errors.Is(err, io.EOF) {
		log.Printf("Relay error: %v", err)
	}
}

func (h *ForwarderHandler) HandleUDP(e *udp.ForwarderRequest) {
	ep := e.CreateEndpoint()

	proxyConn, err := net.Dial("udp", h.proxyAddr)
	if err != nil {
		log.Printf("Failed to connect to UDP proxy %s: %v", h.proxyAddr, err)
		ep.Close()
		e.CompleteHijack(nil)
		return
	}

	e.CompleteHijack(nil)

	go h.relayUDP(proxyConn, ep)
	go h.relayUDP(ep, proxyConn)
}

func (h *ForwarderHandler) relayUDP(dst io.Writer, src io.ReadCloser) {
	defer dst.Close()
	_, err := io.CopyBuffer(dst, src, make([]byte, 4096))
	if err != nil && !errors.Is(err, io.EOF) {
		log.Printf("UDP relay error: %v", err)
	}
}

func StartEngine(rwc io.ReadWriteCloser, proxyAddr string) (*Engine, error) {
	if proxyAddr == "" {
		return nil, fmt.Errorf("proxyAddr is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	engine := &Engine{
		proxyAddr: proxyAddr,
		ctx:       ctx,
		cancel:    cancel,
	}

	if err := engine.initStack(rwc); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize stack: %w", err)
	}

	log.Printf("Engine started, proxying to %s", proxyAddr)
	return engine, nil
}

func (e *Engine) initStack(rwc io.ReadWriteCloser) error {
	e.stack = stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol,
			ipv6.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			udp.NewProtocol,
		},
	})

	e.link = NewTunLinkEndpoint(rwc)
	nicID := e.stack.AddNIC(e.link, "burpbridge")

	e.stack.SetRouteTable([]tcpip.Route{
		{
			Destination: tcpip.Address(net.ParseIP("0.0.0.0").To4()),
			Gateway:     "",
			NIC:         nicID,
		},
		{
			Destination: tcpip.Address(net.ParseIP("::").To16()),
			Gateway:     "",
			NIC:         nicID,
		},
	})

	tcpHandler := &ForwarderHandler{proxyAddr: e.proxyAddr}
	tcpFwd := tcp.NewForwarder(e.stack, 0, 1024, tcpHandler.HandleTCP)
	e.stack.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpFwd)

	udpFwd := udp.NewForwarder(e.stack, tcpHandler.HandleUDP)
	e.stack.SetTransportProtocolHandler(udp.ProtocolNumber, udpFwd)

	return nil
}

func (e *Engine) Stop() error {
	e.cancel()
	if e.link != nil {
		e.link.Close()
	}
	if e.stack != nil {
		e.stack.Close()
	}
	log.Println("Engine stopped")
	return nil
}

func (e *Engine) Wait() error {
	<-e.ctx.Done()
	return nil
}
