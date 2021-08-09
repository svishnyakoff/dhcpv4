package net_utils

import (
	"context"
	"github.com/mdlayher/arp"
	"github.com/mdlayher/ethernet"
	"github.com/svishnyakoff/dhcpv4/config"
	"log"
	"math/rand"
	"net"
	"os"
	"time"
)

func GetHardwareAddr(interfaceName string) net.HardwareAddr {
	interFace, err := net.InterfaceByName(interfaceName)

	if err != nil {
		log.Panic("Could not find interface by name", interfaceName, err)
	}

	return interFace.HardwareAddr
}

// GetPreferredOutboundInterface provides network interface that is used by host system by default. The implementation
// implicitly rely on go internal implementation to figure out  best suitable interface to reach default gateway.
func GetPreferredOutboundInterface() *net.Interface {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	localIp := localAddr.IP

	interfaces, _ := net.Interfaces()
	for _, ifi := range interfaces {
		addresses, _ := ifi.Addrs()
		for _, addr := range addresses {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if localIp.Equal(ip) {
				return &ifi
			}
		}
	}

	return nil
}

// AutoPrivateAddr creates Automatic Private IP address (169.254.x.x) that should be mainly used if
// client just started and did not  get valid IP address from server, or by some reason,
// lease expired and client was not able to get new one.
// The address is generated using following algorithm:
// 1. Randomly generate last two octets
// 2. Emit ARP request to validate if devices with generated IP addr present on LAN segment. If not, we are done.
func AutoPrivateAddr() net.IP {
	log.Println("Generate private IP addr")
	i := GetPreferredOutboundInterface()

	client, err := arp.Dial(i)
	if client != nil {
		defer client.Close()
	}

	if err != nil {
		log.Panic("could not create arp client", err)
	}

	var privateIp = generatePrivateIp()

	for arpCheck(privateIp, i, client) {
		privateIp = generatePrivateIp()
	}

	return privateIp
}

func IsApipaAddr(ip net.IP) bool {
	ip = ip.To4()
	return len(ip) == 4 && ip[0] == 169 && ip[1] == 254
}

func IsUniqueIp(addr net.IP, config config.DHCPConfig) bool {
	// todo should use automatic interface detection GetPreferredOutboundInterface?
	i, err := net.InterfaceByName(config.InterfaceName)

	if err != nil {
		// todo replace with error. Panic is to aggressive
		log.Panic("could retrieve interface by name: ", config.InterfaceName, err)
	}

	client, err := arp.Dial(i)
	if client != nil {
		defer client.Close()
	}

	if err != nil {
		log.Panic("could not create arp client", err)
	}

	return !arpCheck(addr, i, client)
}

func arpCheck(ip net.IP, i *net.Interface, client *arp.Client) bool {
	packet, _ := arp.NewPacket(arp.OperationRequest, i.HardwareAddr, net.IPv4zero, ethernet.Broadcast, ip)

	if err := client.WriteTo(packet, ethernet.Broadcast); err != nil {
		log.Println("error sending arp request", err)
		return false
	}

	timeout := 200 * time.Millisecond // todo timer should be configurable
	arpResolutionCh := make(chan bool)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				arpResolutionCh <- false
				return
			default:
				client.SetReadDeadline(time.Now().Add(time.Second))
				replyPacket, _, err := client.Read()

				if err != nil && !os.IsTimeout(err) {
					log.Println("error reading arp reply.", err)
					continue
				}

				if err != nil {
					arpResolutionCh <- false
					return
				}

				if replyPacket.Operation != arp.OperationReply && !replyPacket.SenderIP.Equal(ip) {
					continue
				}

				arpResolutionCh <- true
				return
			}

		}
	}()

	time.AfterFunc(timeout, func() {
		cancel()
		log.Println("Did not get arp reply in ", timeout, "interval. ")
	})

	return <-arpResolutionCh
}

func generatePrivateIp() net.IP {
	return net.IPv4(169, 254, byte(rand.Int()%254)+1, byte(rand.Int()%253)+1).To4()
}
