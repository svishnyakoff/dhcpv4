package test

import (
	"errors"
	"github.com/svishnyakoff/dhcpv4/packet"
	"github.com/svishnyakoff/dhcpv4/packet/option"
	"golang.org/x/net/context"
	"golang.org/x/net/ipv4"
	"golang.org/x/sys/unix"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type Reply func(dhcpPacket packet.DHCPPacket) (packet.DHCPPacket, bool)

var emptyIncomingData = incomingData{}

type incomingData struct {
	data    []byte
	addr    net.Addr
	err     error
	unicast bool
}

type DHCPServer struct {
	replies         chan Reply
	ReceivedPackets chan packet.DHCPPacket
	SentPackets     chan packet.DHCPPacket
	stopped         *atomic.Value
	stopSignal      chan int
	port            int
	addr            net.IP
	conn            *ipv4.PacketConn
	multicastConn   *ipv4.PacketConn
	packetChan      chan *incomingData
	stopAction      *sync.Once
}

func NewDHCPServer(addr net.IP, port int) DHCPServer {
	return DHCPServer{
		replies:         make(chan Reply, 100),
		ReceivedPackets: make(chan packet.DHCPPacket, 100),
		SentPackets:     make(chan packet.DHCPPacket, 100),
		stopSignal:      make(chan int, 1),
		stopped:         &atomic.Value{},
		port:            port,
		addr:            addr,
		packetChan:      make(chan *incomingData, 100),
		stopAction:      &sync.Once{},
	}
}

func (s *DHCPServer) Listen() {
	addr := net.UDPAddr{
		Port: s.port,
		IP:   s.addr.To4(),
	}

	log.Println("start listening:", addr.IP, addr.Port)

	// By some reason I was not able to bind single connection to both unicast and multicast addresses. When i tried
	// bind to unicast and join to multicast group, server did not receive multicast traffic.
	//
	// My second attempt was to spawn two connections: one for multicast and one for unicast.
	// But then again when I tried to run multiple servers I got error that address is already used.
	//
	// Eventually I succeeded with two connections by setting special SO_REUSEPORT option that lets reuse port.
	// That should not be an issue as DHCP server is intended for tests only.
	s.multicastConn = s.initConnection(net.UDPAddr{
		Port: s.port,
		IP:   net.IPv4(224, 0, 0, 1),
	})
	s.conn = s.initConnection(net.UDPAddr{
		Port: s.port,
		IP:   s.addr,
	})

	log.Println("DHCP server is listening")

	go s.listenConnection(s.conn, true)
	go s.listenConnection(s.multicastConn, false)

	go func() {
		for !s.isServerStopped() {
			data := <-s.packetChan

			if data.unicast {
				log.Println("received data on unicast IP")
			} else {
				log.Println("received data on multicast IP")
			}

			var r Reply
			select {
			case r = <-s.replies:
				// we already assigned
			default:
				r = func(dhcpPacket packet.DHCPPacket) (packet.DHCPPacket, bool) {
					return packet.DHCPPacket{}, false
				}
			}

			if data != nil {
				s.HandleCommands(*data, r)
			}

		}
	}()
}

func (s *DHCPServer) initConnection(addr net.UDPAddr) *ipv4.PacketConn {

	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var opErr error
			err := c.Control(func(fd uintptr) {
				opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
			})
			if err != nil {
				return err
			}
			return opErr
		},
	}

	lp, err := lc.ListenPacket(context.Background(), "udp", addr.String())

	conn := lp.(*net.UDPConn)

	if err != nil {
		log.Panic("error binding to multicast group:", err)
	}

	packetConn := ipv4.NewPacketConn(conn)

	if addr.IP.IsMulticast() {
		packetConn.JoinGroup(s.getInterface(), &addr)
	}

	packetConn.SetMulticastInterface(s.getInterface())
	packetConn.SetMulticastLoopback(true)

	return packetConn
}

func (s *DHCPServer) listenConnection(conn *ipv4.PacketConn, isUnicast bool) {
	for {
		data, addr, err := s.readUdp(conn)

		// nil data implicitly signal that connection has been closed
		if data == nil {
			return
		}

		s.packetChan <- &incomingData{data: data, addr: addr, err: err, unicast: isUnicast}
	}

}

func (s *DHCPServer) Stop() {
	if !s.isServerStopped() {
		s.stopAction.Do(func() {
			s.stopped.Store(true)
			if s.conn != nil {
				s.conn.Close()
			}

			if s.multicastConn != nil {
				s.multicastConn.Close()
			}

			close(s.ReceivedPackets)
			close(s.SentPackets)
		})
	}
}

func (s *DHCPServer) HandleCommands(data incomingData, r Reply) {

	bytes, addr, err := data.data, data.addr, data.err

	pack, err := packet.Decode(bytes, len(bytes))

	if err != nil {
		panic(err)
	}

	s.ReceivedPackets <- pack

	answer, ok := r(pack)

	// handle for packet is missing, we will just store the fact we received packet but won't respond back
	if ok {
		_, err = s.conn.WriteTo(answer.Encode(), nil, addr)

		s.SentPackets <- answer

		if err != nil {
			log.Panic("error sending to client", err)
		}
	}
}

func (s *DHCPServer) readUdp(conn *ipv4.PacketConn) ([]byte, net.Addr, error) {
	buffer := make([]byte, 2000)

	conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	bytesRead, _, addr, err := conn.ReadFrom(buffer)

	if os.IsTimeout(err) || bytesRead <= 0 {
		if s.isServerStopped() {
			return nil, nil, errors.New("dhcp server has been stopped")
		}
		return s.readUdp(conn)
	}

	return buffer[:bytesRead], addr, err
}

func (s *DHCPServer) AddReply(p packet.DHCPPacket, options ...option.DHCPOption) {

	reply := func(pack packet.DHCPPacket) (packet.DHCPPacket, bool) {
		var res = packet.DHCPPacket{
			Op:    packet.REPLY,
			Htype: pack.Htype,
			Hlen:  pack.Hlen,

			Xid:    pack.Xid,
			Yiaddr: (orDefault(p.Yiaddr, pack.Yiaddr)).([4]byte),
			Siaddr: (orDefault(p.Siaddr, pack.Siaddr)).([4]byte),
			Giaddr: (orDefault(p.Giaddr, pack.Giaddr)).([4]byte),
			Chaddr: (orDefault(p.Chaddr, pack.Chaddr)).([16]byte),
		}

		for _, v := range options {
			res.AddOption(v)
		}

		return res, true
	}

	s.replies <- reply
}

func (s *DHCPServer) AddReplyWithDelay(p packet.DHCPPacket, delay time.Duration, options ...option.DHCPOption) {

	reply := func(pack packet.DHCPPacket) (packet.DHCPPacket, bool) {
		time.Sleep(delay)

		var res = packet.DHCPPacket{
			Op:    packet.REPLY,
			Htype: pack.Htype,
			Hlen:  pack.Hlen,

			Xid:    pack.Xid,
			Yiaddr: (orDefault(p.Yiaddr, pack.Yiaddr)).([4]byte),
			Siaddr: (orDefault(p.Siaddr, pack.Siaddr)).([4]byte),
			Giaddr: (orDefault(p.Giaddr, pack.Giaddr)).([4]byte),
			Chaddr: (orDefault(p.Chaddr, pack.Chaddr)).([16]byte),
		}

		for _, v := range options {
			res.AddOption(v)
		}

		return res, true
	}

	s.replies <- reply
}

func (s *DHCPServer) getInterface() (ifi *net.Interface) {
	ifi, _ = net.InterfaceByName("lo0")
	return
}

func (s *DHCPServer) ReadAllReceivedPackets() []packet.DHCPPacket {
	packets := make([]packet.DHCPPacket, 0, 10)

	for p := range s.ReceivedPackets {
		packets = append(packets, p)
	}

	return packets
}

func (s *DHCPServer) ReadAllSentPackets() []packet.DHCPPacket {
	packets := make([]packet.DHCPPacket, 0, 10)

	for p := range s.SentPackets {
		packets = append(packets, p)
	}

	return packets
}

func (s *DHCPServer) isServerStopped() bool {
	return s.stopped.Load() != nil
}

func (s *DHCPServer) AddIgnoreToReply() {
	reply := func(pack packet.DHCPPacket) (packet.DHCPPacket, bool) {
		var res = packet.DHCPPacket{}

		return res, false
	}

	s.replies <- reply
}

func orDefault(a interface{}, def interface{}) interface{} {
	switch t := a.(type) {
	case byte:
		if t == 0 {
			return def
		}
		return a
	case []byte:
		if t == nil {
			return def
		}
		return a
	case [4]byte:
		{
			if isEmptyArray(t[:]) {
				return def
			}
			return a
		}
	case [16]byte:
		{
			if isEmptyArray(t[:]) {
				return def
			}
			return a
		}

	default:
		panic("unknown type")
	}
}

func isEmptyArray(a []byte) bool {
	for k := range a {
		if k != 0 {
			return false
		}
	}

	return true
}
