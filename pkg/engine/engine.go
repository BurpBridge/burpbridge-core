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
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
)

var packetBufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 65535)
		return &buf
	},
}

type Engine struct {
	stack     *stack.Stack
	link      *TunLinkEndpoint
	proxyAddr string
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

type TunLinkEndpoint struct {
	rwc io.ReadWriteCloser

	mu     sync.Mutex
	closed bool

	inbound    chan []byte
	dispatcher stack.NetworkDispatcher
}

func NewTunLinkEndpoint(rwc io.ReadWriteCloser) *TunLinkEndpoint {
	return &TunLinkEndpoint{
		rwc:     rwc,
		inbound: make(chan []byte, 256),
	}
}

func (e *TunLinkEndpoint) ReadPacket() (*stack.PacketBuffer, tcpip.NetworkProtocolNumber, error) {
	pkt, ok := <-e.inbound
	if !ok {
		return nil, 0, io.EOF
	}

	// Handle macOS AF header (4 bytes): 0x00 0x00 0x00 0x02 for IPv4
	// The water library on macOS includes this AF header
	if len(pkt) >= 4 {
		af := uint32(pkt[0])<<24 | uint32(pkt[1])<<16 | uint32(pkt[2])<<8 | uint32(pkt[3])
		if af == 2 {
			// IPv4 AF header detected, strip it
			log.Printf("[Ingest] Stripped macOS AF header, %d -> %d bytes", len(pkt), len(pkt)-4)
			pkt = pkt[4:]
		}
	}

	if len(pkt) < 20 {
		return nil, 0, fmt.Errorf("packet too small")
	}

	version := pkt[0] >> 4
	var protoNum tcpip.NetworkProtocolNumber
	if version == 4 {
		protoNum = ipv4.ProtocolNumber
	} else if version == 6 {
		protoNum = ipv6.ProtocolNumber
	} else {
		return nil, 0, fmt.Errorf("unknown IP version: %d", version)
	}

	pb := stack.NewPacketBuffer(stack.PacketBufferOptions{
		ReserveHeaderBytes: 0,
		Payload:            buffer.MakeWithData(pkt),
	})

	return pb, protoNum, nil
}

func (e *TunLinkEndpoint) WritePacket(proto tcpip.NetworkProtocolNumber, localAddr, remoteAddr tcpip.Address, pb *stack.PacketBuffer) error {
	if pb == nil {
		return nil
	}

	data := pb.AsSlices()
	if len(data) == 0 {
		return nil
	}

	var buf []byte
	for _, chunk := range data {
		buf = append(buf, chunk...)
	}

	if len(buf) == 0 {
		return nil
	}

	// Prepend macOS AF header for IPv4 (0x00 0x00 0x00 0x02)
	// This tells macOS the packet is IPv4
	if len(buf) >= 1 && (buf[0]>>4) == 4 {
		header := []byte{0x00, 0x00, 0x00, 0x02}
		buf = append(header, buf...)
		log.Printf("[Egress] Added macOS AF header, %d -> %d bytes", len(buf)-4, len(buf))
	}

	_, err := e.rwc.Write(buf)
	if err != nil {
		log.Printf("WritePacket error: %v", err)
	}
	return err
}

func (e *TunLinkEndpoint) Attach(dispatcher stack.NetworkDispatcher) {
	e.dispatcher = dispatcher

	go e.readLoop()
}

func (e *TunLinkEndpoint) readLoop() {
	for {
		bufPtr := packetBufferPool.Get().(*[]byte)
		buf := *bufPtr
		defer packetBufferPool.Put(bufPtr)

		n, err := e.rwc.Read(buf)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("TUN read error: %v", err)
			}
			return
		}

		if n == 0 {
			continue
		}

		e.mu.Lock()
		if e.closed {
			e.mu.Unlock()
			return
		}

		packet := make([]byte, n)
		copy(packet, buf[:n])

		// Hex dump for diagnostics
		dumpLen := 20
		if dumpLen > n {
			dumpLen = n
		}
		log.Printf("[Ingest] Hex Dump: %X", packet[:dumpLen])

		select {
		case e.inbound <- packet:
		default:
			log.Println("inbound channel full, dropping packet")
		}
		e.mu.Unlock()
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

func (e *TunLinkEndpoint) ARPHardwareType() header.ARPHardwareType {
	return header.ARPHardwareNone
}

func (e *TunLinkEndpoint) AddHeader(pkt *stack.PacketBuffer) {}

func (e *TunLinkEndpoint) ParseHeader(pkt *stack.PacketBuffer) bool {
	return true
}

func (e *TunLinkEndpoint) SetOnCloseAction(func()) {}

func (e *TunLinkEndpoint) SetMTU(mtu uint32) {}

func (e *TunLinkEndpoint) MaxHeaderLength() uint16 {
	return 0
}

func (e *TunLinkEndpoint) LinkAddress() tcpip.LinkAddress {
	return ""
}

func (e *TunLinkEndpoint) SetLinkAddress(addr tcpip.LinkAddress) {}

func (e *TunLinkEndpoint) IsAttached() bool {
	return e.dispatcher != nil
}

func (e *TunLinkEndpoint) WritePackets(pkts stack.PacketBufferList) (int, tcpip.Error) {
	var written int
	for _, pb := range pkts.AsSlice() {
		if err := e.WritePacket(pb.NetworkProtocolNumber, tcpip.Address{}, tcpip.Address{}, pb); err == nil {
			written++
		}
	}
	return written, nil
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
	st        *stack.Stack
	wg        *sync.WaitGroup
}

func (h *ForwarderHandler) HandleTCP(r *tcp.ForwarderRequest) {
	id := r.ID()
	log.Printf("[TCP] Intercepted connection from %s:%d intended for %s:%d",
		id.RemoteAddress.String(), id.RemotePort, id.LocalAddress.String(), id.LocalPort)

	queue := &waiter.Queue{}
	clientEp, err := r.CreateEndpoint(queue)
	if err != nil {
		log.Printf("[TCP] Failed to create endpoint: %v", err)
		r.Complete(true)
		return
	}

	log.Printf("[Relay] Dialing Burp at %s...", h.proxyAddr)
	proxyConn, dialErr := net.Dial("tcp", h.proxyAddr)
	if dialErr != nil {
		log.Printf("[Relay] Failed to connect to proxy %s: %v", h.proxyAddr, dialErr)
		clientEp.Close()
		r.Complete(true)
		return
	}
	log.Printf("[Relay] Dial to Burp SUCCESS")

	r.Complete(false)

	clientConn := gonet.NewTCPConn(queue, clientEp)

	if h.wg != nil {
		h.wg.Add(2)
	}
	go h.relayTCP(clientConn, proxyConn, "Client->Burp")
	go h.relayTCP(proxyConn, clientConn, "Burp->Client")
}

func (h *ForwarderHandler) relayTCP(dst net.Conn, src net.Conn, direction string) {
	if h.wg != nil {
		defer h.wg.Done()
	}
	defer dst.Close()
	buf := make([]byte, 4096)
	n, err := io.CopyBuffer(dst, src, buf)
	if err != nil && !errors.Is(err, io.EOF) {
		log.Printf("[Relay] %s error after %d bytes: %v", direction, n, err)
	} else {
		log.Printf("[Relay] %s closed. Bytes: %d, err: %v", direction, n, err)
	}
}

func (h *ForwarderHandler) HandleUDP(r *udp.ForwarderRequest) bool {
	queue := &waiter.Queue{}
	ep, err := r.CreateEndpoint(queue)
	if err != nil {
		log.Printf("Failed to create UDP endpoint: %v", err)
		return false
	}

	proxyConn, dialErr := net.Dial("udp", h.proxyAddr)
	if dialErr != nil {
		log.Printf("Failed to connect to UDP proxy %s: %v", h.proxyAddr, dialErr)
		ep.Close()
		return false
	}

	udpConn := gonet.NewUDPConn(queue, ep)

	if h.wg != nil {
		h.wg.Add(2)
	}
	go h.relayUDP(proxyConn, udpConn)
	go h.relayUDP(udpConn, proxyConn)

	return true
}

func (h *ForwarderHandler) relayUDP(dst net.Conn, src net.Conn) {
	if h.wg != nil {
		defer h.wg.Done()
	}
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

	engine.startDispatcher()

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
	nicID := tcpip.NICID(1)
	if err := e.stack.CreateNIC(nicID, e.link); err != nil {
		return fmt.Errorf("failed to create NIC: %v", err)
	}

	// Assign virtual IP 10.0.0.2/24 to gVisor NIC so it has an identity
	localIP := tcpip.AddrFrom4([4]byte{10, 0, 0, 2})
	ipv4Addr := tcpip.ProtocolAddress{
		Protocol:          ipv4.ProtocolNumber,
		AddressWithPrefix: tcpip.AddressWithPrefix{Address: localIP, PrefixLen: 24},
	}
	ipv6Addr := tcpip.ProtocolAddress{
		Protocol:          ipv6.ProtocolNumber,
		AddressWithPrefix: tcpip.AddressWithPrefix{Address: tcpip.Address{}, PrefixLen: 0},
	}
	e.stack.AddProtocolAddress(nicID, ipv4Addr, stack.AddressProperties{})
	e.stack.AddProtocolAddress(nicID, ipv6Addr, stack.AddressProperties{})

	// Enable promiscuous mode to accept ALL packets regardless of destination IP
	if err := e.stack.SetPromiscuousMode(nicID, true); err != nil {
		log.Printf("Warning: SetPromiscuousMode failed: %v", err)
	}

	// Enable IP spoofing - allows sending packets with ANY source IP (spoofed)
	// This is critical for transparent proxy - we reply as the destination server
	if err := e.stack.SetSpoofing(nicID, true); err != nil {
		log.Printf("Warning: SetSpoofing failed: %v", err)
	}

	// Set default catch-all route using IPv4EmptySubnet
	e.stack.SetRouteTable([]tcpip.Route{
		{
			Destination: header.IPv4EmptySubnet,
			NIC:         nicID,
		},
	})

	tcpHandler := &ForwarderHandler{proxyAddr: e.proxyAddr, st: e.stack, wg: &e.wg}
	tcpFwd := tcp.NewForwarder(e.stack, 0, 1024, tcpHandler.HandleTCP)
	e.stack.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpFwd.HandlePacket)

	udpFwd := udp.NewForwarder(e.stack, tcpHandler.HandleUDP)
	e.stack.SetTransportProtocolHandler(udp.ProtocolNumber, udpFwd.HandlePacket)

	return nil
}

func (e *Engine) startDispatcher() {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		for {
			pb, proto, err := e.link.ReadPacket()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					log.Printf("[Ingest] ReadPacket error: %v", err)
				}
				return
			}

			// Log packet info - extract IP header for logging
			if data := pb.AsSlices(); len(data) > 0 {
				var srcIP, dstIP string
				if proto == ipv4.ProtocolNumber && len(data[0]) >= 20 {
					hdr := header.IPv4(data[0])
					srcIP = hdr.SourceAddress().String()
					dstIP = hdr.DestinationAddress().String()
				} else if proto == ipv6.ProtocolNumber && len(data[0]) >= 40 {
					hdr := header.IPv6(data[0])
					srcIP = hdr.SourceAddress().String()
					dstIP = hdr.DestinationAddress().String()
				}
				log.Printf("[Ingest] Raw Packet Read: %d bytes, proto=%v, %s -> %s", pb.Size(), proto, srcIP, dstIP)
			}

			if e.link.dispatcher != nil {
				e.link.dispatcher.DeliverNetworkPacket(proto, pb)
			}
		}
	}()
}

func (e *Engine) Stop() error {
	e.cancel()
	if e.link != nil {
		e.link.Close()
	}
	e.wg.Wait()
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
