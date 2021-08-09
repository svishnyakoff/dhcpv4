package core

import (
	"fmt"
	. "github.com/svishnyakoff/dhcpv4/packet"
	"log"
	"net"
)

type DHCPClient struct {
	port         int
	conn         *net.UDPConn
	serverIp     net.IP // field exclusively for test purposes to point client to local test server
	serverPort   int    // field exclusively for test purposes to point client to local test server
	useMulticast bool   // if set to true will multicast packets to server instead of broadcast
}

func (c *DHCPClient) Listen() {
	clientPort := c.port

	if c.port == 0 {
		clientPort = 68
	}

	addr := net.UDPAddr{
		Port: clientPort,
		IP:   net.IPv4zero,
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		panic(err)
	}

	c.conn = conn
	log.Println("Listening for dhcp commands from server")
}

func (c *DHCPClient) Stop() {
	if err := c.conn.Close(); err != nil {
		log.Println("Could not stop client gracefully", err)
	}
}

func (c *DHCPClient) Send(packet DHCPPacket, addr net.IP) error {
	if c.useMulticast && addr.To4().Equal(net.IPv4bcast) {
		addr = net.IPv4(224, 0, 0, 1)
	}

	port := 67

	if c.serverPort != 0 {
		port = c.serverPort
	}

	udpAddr := net.UDPAddr{
		IP:   addr,
		Port: port,
	}

	log.Printf("--> %v\n%v\n\n", packet.GetMessageType(), packet)

	if _, err := c.conn.WriteToUDP(packet.Encode(), &udpAddr); err != nil {
		return fmt.Errorf("error writing dhcp data to server. "+
			"It could be problems with the network or router got inaccessible: %v", err)
	}

	return nil
}
