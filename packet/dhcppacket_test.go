package packet

import (
	"github.com/stretchr/testify/assert"
	"github.com/svishnyakoff/dhcpv4/packet/option"
	"github.com/svishnyakoff/dhcpv4/util/converter"
	"net"
	"testing"
)

func createPacket() DHCPPacket {
	hw, _ := net.ParseMAC("00:0a:95:9d:68:16")
	packet := DHCPPacket{
		Op:    REQUEST,
		Htype: 1, // 1 - 10 mb ethernet
		Hlen:  6, // For Ethernet or other networks using IEEE 802 MAC addresses,
		// the value is 6
		// https://datatracker.ietf.org/doc/html/rfc2131#page-34, http://www.tcpipguide.com/free/t_DHCPMessageFormat.htm
		Xid:    1234,
		Secs:   50,
		Ciaddr: converter.IP2Array(net.IPv4bcast),
		Chaddr: converter.Hardware2Array(hw),
	}

	return packet
}

func TestPacketStringRepresentation(t *testing.T) {
	hw, _ := net.ParseMAC("00:0a:95:9d:68:16")
	packet := DHCPPacket{
		Op:    REQUEST,
		Htype: 1, // 1 - 10 mb ethernet
		Hlen:  6, // For Ethernet or other networks using IEEE 802 MAC addresses,
		// the value is 6
		// https://datatracker.ietf.org/doc/html/rfc2131#page-34, http://www.tcpipguide.com/free/t_DHCPMessageFormat.htm
		Xid:    1234,
		Secs:   50,
		Ciaddr: converter.IP2Array(net.IPv4bcast),
		Chaddr: converter.Hardware2Array(hw),
	}

	packet.AddOption(option.NewMessageTypeOpt(option.DHCPREQUEST))

	assert.JSONEq(t, `
				{"Op":"request","Htype":1,"Hlen":6,"Hops":0,"Xid":1234,"Secs":50,"Flags":"0","Ciaddr":"0.0.0.0",
				"Yiaddr":"0.0.0.0","Siaddr":"0.0.0.0","Giaddr":"0.0.0.0","Chaddr":"00:0a:95:9d:68:16","Sname":"",
				"File":"","Options":[{"ID":"ROUTER_OPT"}]}`, packet.String())
}

func TestDHCPPacket_GetOptions(t *testing.T) {
	packet := createPacket()

	packet.AddOption(option.NewMessageTypeOpt(option.DHCPDISCOVER))

	options := packet.GetOptions()
	assert.Contains(t, options, option.NewMessageTypeOpt(option.DHCPDISCOVER))
}
