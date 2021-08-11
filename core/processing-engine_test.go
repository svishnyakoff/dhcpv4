package core

import (
	"github.com/stretchr/testify/assert"
	"github.com/svishnyakoff/dhcpv4/lease"
	"github.com/svishnyakoff/dhcpv4/util/converter"
	netUtils "github.com/svishnyakoff/dhcpv4/util/net-utils"
	"net"
	"os"
	"testing"
	"time"
)
import (
	"github.com/svishnyakoff/dhcpv4/config"
	"github.com/svishnyakoff/dhcpv4/packet"
	"github.com/svishnyakoff/dhcpv4/packet/option"
	"github.com/svishnyakoff/dhcpv4/test"
)

func TestHappyPathToGetLease(t *testing.T) {
	leaseReceiveListener := new(LeaseListener)
	leaseRenewListener := new(LeaseListener)
	server := test.NewDHCPServer(net.ParseIP("127.0.0.1"), 2024)

	server.AddReply(packet.DHCPPacket{
		Yiaddr: converter.IP2Array(net.ParseIP("127.0.0.2").To4()),
	}, option.NewIpAddrLeaseTime(200), option.NewMessageTypeOpt(option.DHCPOFFER),
		option.NewServerIdentifierOpt(net.ParseIP("127.0.0.1").To4()))

	server.AddReply(packet.DHCPPacket{
		Yiaddr: converter.IP2Array(net.ParseIP("127.0.0.2").To4()),
	}, option.NewIpAddrLeaseTime(200), option.NewMessageTypeOpt(option.DHCPACK),
		option.NewServerIdentifierOpt(net.ParseIP("127.0.0.1").To4()))

	server.Listen()
	processingEngine := NewProcessingEngine(ProcessingEngineInitProps{
		Client: &UdpClient{serverPort: 2024, useMulticast: true},
		Config: &config.GlobalDHCPConfig,
	})
	processingEngine.AddLeaseReceivedListener(leaseReceiveListener.listen)
	processingEngine.AddLeaseRenewedListener(leaseRenewListener.listen)
	processingEngine.Start()

	time.Sleep(time.Second * 5)

	processingEngine.Stop()
	server.Stop()

	serverReceivedPackets := server.ReadAllReceivedPackets()

	assert.True(t, serverReceivedPackets[0].Flags != 0)
	assert.Equal(t, 1, leaseReceiveListener.count)
	assert.Equal(t, 0, leaseRenewListener.count)

	l := processingEngine.GetLease()
	assertLease(t, LeaseExpectation{
		State:            lease.BOUND,
		IpAddr:           net.ParseIP("127.0.0.2").To4(),
		LeaseDuration:    time.Second * 200,
		ServerIdentifier: net.ParseIP("127.0.0.1").To4(),
	}, l)
}

func TestClientToRetryRequest(t *testing.T) {
	os.Setenv("RetryRequestSec", "1")
	os.Setenv("StopOnLeaseAcquisitionFailure", "true")

	defer os.Unsetenv("RetryRequestSec")
	defer os.Unsetenv("StopOnLeaseAcquisitionFailure")
	server := test.NewDHCPServer(net.ParseIP("127.0.0.1"), 2024)
	leaseReceiveListener := new(LeaseListener)
	leaseRenewListener := new(LeaseListener)

	server.AddReply(packet.DHCPPacket{
		Yiaddr: converter.IP2Array(net.ParseIP("127.0.0.2").To4()),
	}, option.NewIpAddrLeaseTime(200), option.NewMessageTypeOpt(option.DHCPOFFER),
		option.NewServerIdentifierOpt(net.ParseIP("127.0.0.1").To4()))

	server.Listen()
	conf, _ := config.LoadConfig()
	processingEngine := NewProcessingEngine(ProcessingEngineInitProps{
		Client: &UdpClient{serverPort: 2024, useMulticast: true},
		Config: &conf,
	})
	processingEngine.AddLeaseReceivedListener(leaseReceiveListener.listen)
	processingEngine.AddLeaseRenewedListener(leaseRenewListener.listen)
	processingEngine.Start()

	packetFactory := DHCPPacketFactory{Config: conf}

	time.Sleep(time.Second * 9)

	processingEngine.Stop()
	server.Stop()

	serverReceivedPackets := server.ReadAllReceivedPackets()
	serverSentPackets := server.ReadAllSentPackets()
	l := processingEngine.GetLease()

	discoverPacket, _ := packetFactory.Discover()
	discoverPacket.Xid = serverReceivedPackets[0].Xid

	requestPacket, _ := packetFactory.RequestForOffer(serverSentPackets[0])
	requestPacket.Xid = serverReceivedPackets[1].Xid

	assert.Equal(t, 0, leaseReceiveListener.count)
	assert.Equal(t, 0, leaseRenewListener.count)
	assert.Equal(t, 4, len(serverReceivedPackets))

	assert.ElementsMatch(t, []packet.DHCPPacket{
		*discoverPacket, *requestPacket, *requestPacket, *requestPacket,
	}, serverReceivedPackets, "expected: %v\n, received: %v", []packet.DHCPPacket{
		*discoverPacket, *requestPacket, *requestPacket, *requestPacket,
	}, serverReceivedPackets)

	assertLease(t, LeaseExpectation{
		State:  lease.INIT,
		IpAddr: l.IpAddr,
	}, l)

	assert.True(t, netUtils.IsApipaAddr(l.IpAddr))
}

// TestRenewLeaseAfterReboot verifies INIT_BOOT -> BOUND transition,
// that is situation when lease was previously acquired and for example we restarted host
func TestRenewLeaseAfterReboot(t *testing.T) {
	server := test.NewDHCPServer(net.ParseIP("127.0.0.1"), 2024)
	leaseReceiveListener := new(LeaseListener)
	leaseRenewListener := new(LeaseListener)

	server.AddReply(packet.DHCPPacket{
		Yiaddr: converter.IP2Array(net.ParseIP("127.0.0.2").To4()),
	}, option.NewIpAddrLeaseTime(200), option.NewMessageTypeOpt(option.DHCPACK),
		option.NewServerIdentifierOpt(net.ParseIP("127.0.0.1").To4()))

	server.Listen()
	conf, _ := config.LoadConfig()
	lizz := lease.DHCPLease{
		State:            lease.BOUND, // client itself should switch BOUND to INIT_REBOOT state during boot
		IpAddr:           net.ParseIP("127.0.0.2").To4(),
		ServerIdentifier: net.ParseIP("127.0.0.1").To4(),
		LeaseInitTime:    time.Now(),
		LeaseDuration:    100 * time.Second,
		T1:               50 * time.Second,
		T2:               75 * time.Second,
	}
	processingEngine := NewProcessingEngine(ProcessingEngineInitProps{
		Client: &UdpClient{serverPort: 2024, useMulticast: true},
		Config: &conf,
		Lease:  &lizz,
	})
	processingEngine.AddLeaseReceivedListener(leaseReceiveListener.listen)
	processingEngine.AddLeaseRenewedListener(leaseRenewListener.listen)
	processingEngine.Start()

	packetFactory := DHCPPacketFactory{Config: conf}

	time.Sleep(time.Second * 2)

	processingEngine.Stop()
	server.Stop()

	serverReceivedPackets := server.ReadAllReceivedPackets()
	l := processingEngine.GetLease()

	requestPacket, _ := packetFactory.RequestForReboot(lizz)
	requestPacket.Xid = serverReceivedPackets[0].Xid

	assert.Equal(t, 0, leaseReceiveListener.count)
	assert.Equal(t, 1, leaseRenewListener.count)

	assert.Equal(t, 1, len(serverReceivedPackets))

	assert.ElementsMatch(t, []packet.DHCPPacket{
		*requestPacket,
	}, serverReceivedPackets, "expected: %v\n, received: %v", []packet.DHCPPacket{
		*requestPacket,
	}, serverReceivedPackets)

	assert.Equal(t, lease.BOUND, l.State)
	assert.Equal(t, net.ParseIP("127.0.0.2").To4(), l.IpAddr)
	assert.Equal(t, net.ParseIP("127.0.0.1").To4(), l.ServerIdentifier)
	assert.Equal(t, 200*time.Second, l.LeaseDuration)
}

func TestRenewAfterRebootFailedAndThenRequestNewLease(t *testing.T) {
	server := test.NewDHCPServer(net.ParseIP("127.0.0.1"), 2024)
	leaseReceiveListener := new(LeaseListener)
	leaseRenewListener := new(LeaseListener)

	server.AddReply(packet.DHCPPacket{},
		option.NewMessageTypeOpt(option.DHCPNAK), option.NewServerIdentifierOpt(net.ParseIP("127.0.0.1").To4()))

	server.AddReply(packet.DHCPPacket{
		Yiaddr: converter.IP2Array(net.ParseIP("127.0.0.3").To4()),
	}, option.NewIpAddrLeaseTime(200), option.NewMessageTypeOpt(option.DHCPOFFER),
		option.NewServerIdentifierOpt(net.ParseIP("127.0.0.1").To4()))

	server.AddReply(packet.DHCPPacket{
		Yiaddr: converter.IP2Array(net.ParseIP("127.0.0.3").To4()),
	}, option.NewIpAddrLeaseTime(300), option.NewMessageTypeOpt(option.DHCPACK),
		option.NewServerIdentifierOpt(net.ParseIP("127.0.0.1").To4()))

	server.Listen()
	conf, _ := config.LoadConfig()
	lizz := lease.DHCPLease{
		State:            lease.BOUND, // client itself should switch BOUND to INIT_REBOOT state during boot
		IpAddr:           net.ParseIP("127.0.0.2").To4(),
		ServerIdentifier: net.ParseIP("127.0.0.1").To4(),
		LeaseInitTime:    time.Now(),
		LeaseDuration:    100 * time.Second,
		T1:               50 * time.Second,
		T2:               75 * time.Second,
	}
	processingEngine := NewProcessingEngine(ProcessingEngineInitProps{
		Client: &UdpClient{serverPort: 2024, useMulticast: true},
		Config: &conf,
		Lease:  &lizz,
	})
	processingEngine.AddLeaseReceivedListener(leaseReceiveListener.listen)
	processingEngine.AddLeaseRenewedListener(leaseRenewListener.listen)
	processingEngine.Start()

	packetFactory := DHCPPacketFactory{Config: conf}

	time.Sleep(time.Second * 5)

	processingEngine.Stop()
	server.Stop()

	serverReceivedPackets := server.ReadAllReceivedPackets()
	l := processingEngine.GetLease()

	requestPacket, _ := packetFactory.RequestForReboot(lizz)
	requestPacket.Xid = serverReceivedPackets[0].Xid

	assert.Equal(t, 1, leaseReceiveListener.count)
	assert.Equal(t, 0, leaseRenewListener.count)

	assert.Equal(t, 3, len(serverReceivedPackets))
	assert.Equal(t, option.DHCPREQUEST, serverReceivedPackets[0].GetMessageType())
	assert.Equal(t, option.DHCPDISCOVER, serverReceivedPackets[1].GetMessageType())
	assert.Equal(t, option.DHCPREQUEST, serverReceivedPackets[2].GetMessageType())

	assert.Equal(t, lease.BOUND, l.State)
	assert.Equal(t, net.ParseIP("127.0.0.3").To4(), l.IpAddr)
	assert.Equal(t, net.ParseIP("127.0.0.1").To4(), l.ServerIdentifier)
	assert.Equal(t, 300*time.Second, l.LeaseDuration)
}

func TestReceiveSeveralOffers(t *testing.T) {
	os.Setenv("RetryRequestSec", "0")
	defer os.Unsetenv("RetryRequestSec")

	server1 := test.NewDHCPServer(net.ParseIP("127.0.0.1").To4(), 2024)
	server2 := test.NewDHCPServer(net.ParseIP("127.0.0.2").To4(), 2024)

	leaseReceiveListener := new(LeaseListener)
	leaseRenewListener := new(LeaseListener)

	server1.AddReply(packet.DHCPPacket{
		Yiaddr: converter.IP2Array(net.ParseIP("127.0.0.7").To4()),
	}, option.NewIpAddrLeaseTime(200), option.NewMessageTypeOpt(option.DHCPOFFER),
		option.NewServerIdentifierOpt(net.ParseIP("127.0.0.1").To4()))

	server1.AddReply(packet.DHCPPacket{
		Yiaddr: converter.IP2Array(net.ParseIP("127.0.0.7").To4()),
	}, option.NewIpAddrLeaseTime(200), option.NewMessageTypeOpt(option.DHCPACK),
		option.NewServerIdentifierOpt(net.ParseIP("127.0.0.1").To4()))

	server2.AddReplyWithDelay(packet.DHCPPacket{
		Yiaddr: converter.IP2Array(net.ParseIP("127.0.0.8").To4()),
	}, time.Second,
		option.NewIpAddrLeaseTime(200), option.NewMessageTypeOpt(option.DHCPOFFER),
		option.NewServerIdentifierOpt(net.ParseIP("127.0.0.2").To4()))

	server1.Listen()
	server2.Listen()

	dhcpConfig, _ := config.LoadConfig()
	processingEngine := NewProcessingEngine(ProcessingEngineInitProps{
		Client: &UdpClient{serverPort: 2024, useMulticast: true},
		Config: &dhcpConfig,
	})
	processingEngine.AddLeaseReceivedListener(leaseReceiveListener.listen)
	processingEngine.AddLeaseRenewedListener(leaseRenewListener.listen)
	processingEngine.Start()

	packetFactory := DHCPPacketFactory{Config: dhcpConfig}

	time.Sleep(time.Second * 5)

	processingEngine.Stop()
	server1.Stop()
	server2.Stop()

	server1ReceivedPackets := server1.ReadAllReceivedPackets()
	server2ReceivedPackets := server2.ReadAllReceivedPackets()

	server1SentPackets := server1.ReadAllSentPackets()

	discoverPacket, _ := packetFactory.Discover()
	discoverPacket.Xid = server1ReceivedPackets[0].Xid

	requestPacket, _ := packetFactory.RequestForOffer(server1SentPackets[0])
	// expect that client accept first offer it receives.
	// Server 2 intentionally responds with delay to give first server time to send and client to process the offer.
	// Client then should broadcast REQUEST to all servers.
	requestPacket.Xid = server1ReceivedPackets[1].Xid

	l := processingEngine.GetLease()
	assert.Equal(t, 1, leaseReceiveListener.count)
	assert.Equal(t, 0, leaseRenewListener.count)
	assert.Equal(t, lease.BOUND, l.State)

	assert.ElementsMatch(t, []packet.DHCPPacket{
		*discoverPacket, *requestPacket,
	}, server1ReceivedPackets, "expected: %v\n, received: %v", []packet.DHCPPacket{
		*discoverPacket, *requestPacket,
	}, server1ReceivedPackets)

	assert.ElementsMatch(t, []packet.DHCPPacket{
		*discoverPacket, *requestPacket,
	}, server1ReceivedPackets, "expected: %v\n, received: %v", []packet.DHCPPacket{
		*discoverPacket, *requestPacket,
	}, server2ReceivedPackets)
}

func TestRenewingLease(t *testing.T) {
	server := test.NewDHCPServer(net.ParseIP("127.0.0.1"), 2024)
	leaseReceiveListener := new(LeaseListener)
	leaseRenewListener := new(LeaseListener)

	server.AddReply(packet.DHCPPacket{
		Yiaddr: converter.IP2Array(net.ParseIP("127.0.0.2").To4()),
	}, option.NewIpAddrLeaseTime(200), option.NewMessageTypeOpt(option.DHCPOFFER),
		option.NewT1Opt(3), option.NewServerIdentifierOpt(net.ParseIP("127.0.0.1").To4()))

	server.AddReply(packet.DHCPPacket{
		Yiaddr: converter.IP2Array(net.ParseIP("127.0.0.2").To4()),
	}, option.NewIpAddrLeaseTime(200), option.NewMessageTypeOpt(option.DHCPACK),
		option.NewT1Opt(3),
		option.NewServerIdentifierOpt(net.ParseIP("127.0.0.1").To4()))

	server.AddReply(packet.DHCPPacket{
		Yiaddr: converter.IP2Array(net.ParseIP("127.0.0.2").To4()),
	}, option.NewIpAddrLeaseTime(300), option.NewMessageTypeOpt(option.DHCPACK),
		option.NewServerIdentifierOpt(net.ParseIP("127.0.0.1").To4()))

	server.Listen()
	processingEngine := NewProcessingEngine(ProcessingEngineInitProps{
		Client: &UdpClient{serverPort: 2024, useMulticast: true},
		Config: &config.GlobalDHCPConfig,
	})
	processingEngine.AddLeaseReceivedListener(leaseReceiveListener.listen)
	processingEngine.AddLeaseRenewedListener(leaseRenewListener.listen)
	processingEngine.Start()

	time.Sleep(time.Second * 10)

	processingEngine.Stop()
	server.Stop()

	assert.Equal(t, 1, leaseReceiveListener.count)
	assert.Equal(t, 1, leaseRenewListener.count)

	serverReceivedPackets := server.ReadAllReceivedPackets()
	assert.Equal(t, 3, len(serverReceivedPackets))
	assert.Equal(t, option.DHCPDISCOVER, serverReceivedPackets[0].GetMessageType())
	assert.Equal(t, option.DHCPREQUEST, serverReceivedPackets[1].GetMessageType())
	assert.Equal(t, option.DHCPREQUEST, serverReceivedPackets[2].GetMessageType())

	l := processingEngine.GetLease()
	assertLease(t, LeaseExpectation{
		State:            lease.BOUND,
		IpAddr:           net.ParseIP("127.0.0.2").To4(),
		LeaseDuration:    time.Second * 300,
		ServerIdentifier: net.ParseIP("127.0.0.1").To4(),
	}, l)
}

func TestRebindLease(t *testing.T) {
	server := test.NewDHCPServer(net.ParseIP("127.0.0.1"), 2024)
	leaseReceiveListener := new(LeaseListener)
	leaseRenewListener := new(LeaseListener)

	server.AddReply(packet.DHCPPacket{
		Yiaddr: converter.IP2Array(net.ParseIP("127.0.0.2").To4()),
	}, option.NewIpAddrLeaseTime(200), option.NewMessageTypeOpt(option.DHCPOFFER),
		option.NewServerIdentifierOpt(net.ParseIP("127.0.0.1").To4()))

	server.AddReply(packet.DHCPPacket{
		Yiaddr: converter.IP2Array(net.ParseIP("127.0.0.2").To4()),
	}, option.NewIpAddrLeaseTime(200), option.NewMessageTypeOpt(option.DHCPACK),
		option.NewT1Opt(3),
		option.NewT2Opt(5),
		option.NewServerIdentifierOpt(net.ParseIP("127.0.0.1").To4()))

	// ignoring first attempt to renew lease
	server.AddIgnoreToReply()

	// reply with ACK when client tries to renew lease after moving to rebind state
	server.AddReply(packet.DHCPPacket{
		Yiaddr: converter.IP2Array(net.ParseIP("127.0.0.2").To4()),
	}, option.NewIpAddrLeaseTime(300), option.NewMessageTypeOpt(option.DHCPACK),
		option.NewServerIdentifierOpt(net.ParseIP("127.0.0.1").To4()))

	server.Listen()
	processingEngine := NewProcessingEngine(ProcessingEngineInitProps{
		Client: &UdpClient{serverPort: 2024, useMulticast: true},
		Config: &config.GlobalDHCPConfig,
	})
	processingEngine.AddLeaseReceivedListener(leaseReceiveListener.listen)
	processingEngine.AddLeaseRenewedListener(leaseRenewListener.listen)
	processingEngine.Start()

	time.Sleep(time.Second * 10)

	processingEngine.Stop()
	server.Stop()

	assert.Equal(t, 1, leaseReceiveListener.count)
	assert.Equal(t, 1, leaseRenewListener.count)

	serverReceivedPackets := server.ReadAllReceivedPackets()
	assert.Equal(t, 4, len(serverReceivedPackets))
	assert.Equal(t, option.DHCPDISCOVER, serverReceivedPackets[0].GetMessageType())
	assert.Equal(t, option.DHCPREQUEST, serverReceivedPackets[1].GetMessageType())
	assert.Equal(t, option.DHCPREQUEST, serverReceivedPackets[1].GetMessageType())
	assert.Equal(t, option.DHCPREQUEST, serverReceivedPackets[2].GetMessageType())

	l := processingEngine.GetLease()
	assertLease(t, LeaseExpectation{
		State:            lease.BOUND,
		IpAddr:           net.ParseIP("127.0.0.2").To4(),
		LeaseDuration:    time.Second * 300,
		ServerIdentifier: net.ParseIP("127.0.0.1").To4(),
	}, l)
}

func TestDelayedOffer(t *testing.T) {
	os.Setenv("MaxOfferWaitTimeSec", "5")
	defer os.Unsetenv("MaxOfferWaitTimeSec")

	leaseReceiveListener := new(LeaseListener)
	leaseRenewListener := new(LeaseListener)
	server := test.NewDHCPServer(net.ParseIP("127.0.0.1"), 2024)

	server.AddReplyWithDelay(packet.DHCPPacket{
		Yiaddr: converter.IP2Array(net.ParseIP("127.0.0.2").To4()),
	}, time.Millisecond*4500, option.NewIpAddrLeaseTime(200), option.NewMessageTypeOpt(option.DHCPOFFER),
		option.NewServerIdentifierOpt(net.ParseIP("127.0.0.1").To4()))

	server.AddReply(packet.DHCPPacket{
		Yiaddr: converter.IP2Array(net.ParseIP("127.0.0.2").To4()),
	}, option.NewIpAddrLeaseTime(200), option.NewMessageTypeOpt(option.DHCPACK),
		option.NewServerIdentifierOpt(net.ParseIP("127.0.0.1").To4()))

	server.Listen()
	dhcpConfig, _ := config.LoadConfig()
	processingEngine := NewProcessingEngine(ProcessingEngineInitProps{
		Client: &UdpClient{serverPort: 2024, useMulticast: true},
		Config: &dhcpConfig,
	})
	processingEngine.AddLeaseReceivedListener(leaseReceiveListener.listen)
	processingEngine.AddLeaseRenewedListener(leaseRenewListener.listen)
	processingEngine.Start()

	time.Sleep(time.Second * 7)

	processingEngine.Stop()
	server.Stop()

	serverReceivedPackets := server.ReadAllReceivedPackets()

	assert.True(t, serverReceivedPackets[0].Flags != 0)
	assert.Equal(t, 1, leaseReceiveListener.count)
	assert.Equal(t, 0, leaseRenewListener.count)

	l := processingEngine.GetLease()
	assertLease(t, LeaseExpectation{
		State:            lease.BOUND,
		IpAddr:           net.ParseIP("127.0.0.2").To4(),
		LeaseDuration:    time.Second * 200,
		ServerIdentifier: net.ParseIP("127.0.0.1").To4(),
	}, l)
}

type LeaseListener struct {
	count int
}

func (l *LeaseListener) listen(dhcpLease lease.DHCPLease) {
	l.count++
}

func assertLease(t *testing.T, expectation LeaseExpectation, l lease.DHCPLease) {
	assert.Equal(t, expectation, LeaseExpectation{
		State:            l.State,
		IpAddr:           l.IpAddr,
		Dns:              l.Dns,
		SubnetMask:       l.SubnetMask,
		ServerIdentifier: l.ServerIdentifier,
		LeaseDuration:    l.LeaseDuration,
	})
}

type LeaseExpectation struct {
	State            lease.State
	IpAddr           net.IP
	Dns              net.IP
	SubnetMask       net.IPMask
	ServerIdentifier net.IP
	LeaseDuration    time.Duration
}
