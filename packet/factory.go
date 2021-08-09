package packet

import (
	. "github.com/svishnyakoff/dhcpv4/config"
	. "github.com/svishnyakoff/dhcpv4/lease"
	"github.com/svishnyakoff/dhcpv4/packet/option"
	. "github.com/svishnyakoff/dhcpv4/transaction"
	"github.com/svishnyakoff/dhcpv4/util/converter"
	netUtils "github.com/svishnyakoff/dhcpv4/util/net-utils"
	"net"
)

var hardwareInterface = netUtils.GetPreferredOutboundInterface()

type DHCPPacketFactory struct {
	Config DHCPConfig
}

func (f *DHCPPacketFactory) Discover() (*DHCPPacket, TxId) {
	tx := RandomTransactionId()
	config := f.Config

	packet := DHCPPacket{
		Op:    REQUEST,
		Htype: uint8(config.HardwareType),    // 1 - 10 mb ethernet
		Hlen:  uint8(config.HardwareAddrLen), // For Ethernet or other networks using IEEE 802 MAC addresses,
		// the value is 6
		// https://datatracker.ietf.org/doc/html/rfc2131#page-34, http://www.tcpipguide.com/free/t_DHCPMessageFormat.htm
		Xid:    tx,
		Secs:   0,
		Ciaddr: converter.IP2Array(net.IPv4zero),
		Chaddr: converter.Hardware2Array(netUtils.GetHardwareAddr(hardwareInterface.Name)),
	}

	packet.MarkBroadcastFlag()
	packet.AddOption(option.NewMessageTypeOpt(option.DHCPDISCOVER))

	return &packet, tx
}

func (f *DHCPPacketFactory) RequestForRenew(lease DHCPLease) (*DHCPPacket, TxId) {
	tx := RandomTransactionId()
	config := f.Config

	packet := DHCPPacket{
		Op:    REQUEST,
		Htype: uint8(config.HardwareType),    // 1 - 10 mb ethernet
		Hlen:  uint8(config.HardwareAddrLen), // For Ethernet or other networks using IEEE 802 MAC addresses,
		// the value is 6
		// https://datatracker.ietf.org/doc/html/rfc2131#page-34, http://www.tcpipguide.com/free/t_DHCPMessageFormat.htm
		Xid:    tx,
		Secs:   0,
		Ciaddr: converter.IP2Array(lease.IpAddr),
		Chaddr: converter.Hardware2Array(netUtils.GetHardwareAddr(hardwareInterface.Name)),
	}

	packet.AddOption(option.NewMessageTypeOpt(option.DHCPREQUEST))

	return &packet, tx
}

func (f *DHCPPacketFactory) RequestForRebind(lease DHCPLease) (*DHCPPacket, TxId) {
	request, tx := f.RequestForRenew(lease)

	return request, tx
}

func (f *DHCPPacketFactory) RequestForOffer(offer DHCPPacket) (*DHCPPacket, TxId) {
	// https://datatracker.ietf.org/doc/html/rfc2131#page-34
	// The DHCPREQUEST message contains the same 'xid' as the DHCPOFFER message.
	tx := offer.Xid
	config := f.Config

	packet := DHCPPacket{
		Op:    REQUEST,
		Htype: uint8(config.HardwareType),
		Hlen:  uint8(config.HardwareAddrLen),
		Xid:   tx,
		// https://datatracker.ietf.org/doc/html/rfc2131#section-3.1
		//"To help
		//     ensure that any BOOTP relay agents forward the DHCPREQUEST message
		//     to the same set of DHCP servers that received the original
		//     DHCPDISCOVER message, the DHCPREQUEST message MUST use the same
		//     value in the DHCP message header's 'secs' field and be sent to the
		//     same IP broadcast address as the original DHCPDISCOVER message."
		Secs: 0,

		Ciaddr: converter.IP2Array(net.IPv4zero), // 0.0.0.0
		Chaddr: converter.Hardware2Array(netUtils.GetHardwareAddr(hardwareInterface.Name)),
	}

	packet.MarkBroadcastFlag()

	packet.AddOption(option.NewRequestIpAddrOpt(offer.Yiaddr[:]))
	packet.AddOption(option.NewMessageTypeOpt(option.DHCPREQUEST))
	packet.AddOption(*offer.GetOption(option.SERVER_IDENTIFIER))

	return &packet, tx
}

func (f *DHCPPacketFactory) RequestForReboot(lease DHCPLease) (*DHCPPacket, TxId) {
	// https://datatracker.ietf.org/doc/html/rfc2131#page-34
	// The DHCPREQUEST message contains the same 'xid' as the DHCPOFFER message.
	tx := RandomTransactionId()
	config := f.Config

	packet := DHCPPacket{
		Op:    REQUEST,
		Htype: uint8(config.HardwareType),
		Hlen:  uint8(config.HardwareAddrLen),
		Xid:   tx,
		// https://datatracker.ietf.org/doc/html/rfc2131#section-3.1
		//"To help
		//     ensure that any BOOTP relay agents forward the DHCPREQUEST message
		//     to the same set of DHCP servers that received the original
		//     DHCPDISCOVER message, the DHCPREQUEST message MUST use the same
		//     value in the DHCP message header's 'secs' field and be sent to the
		//     same IP broadcast address as the original DHCPDISCOVER message."
		Secs: 0,

		Ciaddr: converter.IP2Array(net.IPv4zero), // 0.0.0.0
		Chaddr: converter.Hardware2Array(netUtils.GetHardwareAddr(hardwareInterface.Name)),
	}

	packet.MarkBroadcastFlag()

	packet.AddOption(option.NewRequestIpAddrOpt(lease.IpAddr))
	packet.AddOption(option.NewMessageTypeOpt(option.DHCPREQUEST))

	return &packet, tx
}

func (f *DHCPPacketFactory) Decline(ack DHCPPacket) (*DHCPPacket, TxId) {
	tx := RandomTransactionId()

	packet := DHCPPacket{
		Op:     REQUEST,
		Xid:    tx,
		Chaddr: converter.Hardware2Array(netUtils.GetHardwareAddr(hardwareInterface.Name)),
	}

	packet.AddOption(option.NewRequestIpAddrOpt(ack.Yiaddr[:]))

	return &packet, tx
}
