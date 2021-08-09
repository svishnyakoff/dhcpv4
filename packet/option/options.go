package option

import (
	"encoding/binary"
	"net"
	"strconv"
	"time"
)

type OptionType int

const (
	SUBNET_MASK OptionType = iota + 1
	TIME_OFFSET
	ROUTER_OPT
	TIME_SERVER_OPT
	NAME_SERVER_OPT
	DOMAIN_NAME_SERVER_OPT
	LOG_SERVER_OPT
	COOKIE_SERVER_OPT
	LPR_SERVER_OPT
	IMPRESS_SERVER_OPT
	RESOURCE_LOCATION_SERVER_OPT
	HOST_NAME_OPT
	BOOT_FILE_SIZE_OPT
	MERIT_DUMP_FILE
	DOMAIN_NAME
	SWAP_SERVER
	ROOT_PATH
	EXTENSION_PATH
	IP_FORWARDING_OPT
	NON_LOCAL_SOURCE_ROUTING_OPT
	POLICY_FILTER_OPT
	MAX_DATAGRAM_REASSEMBLY_SIZE
	DEFAULT_IP_TTL
	PATH_MTU_AGING_TIMEOUT_OPT
	PATH_MTU_PLATEAU_TABLE_OPT

	// ip layer parameters per interface
	INTERFACE_MTU_OPT
	ALL_SUBNET_ARE_LOCAL_OPT
	BROADCAST_ADDR_OPT
	PERFORM_MASK_DISCOVERY_OPT
	MASK_SUPPLIER_OPT
	PERFORM_ROUTER_DICOVERY_OPT
	ROUTER_SOLICITATION_ADDR_OPT
	STATIC_ROUTE_OPT

	// link layer parameters per interface
	TRAILER_ENCAPSULATION_OPT
	ARP_CACHE_TIMEOUT_OPT
	ETHERNET_ENCAPSULATION_OPT

	// tcp parameters
	TCP_DEFAULT_TTL_OPT
	TCP_KEAPALIVE_INTERVAL_OPT
	TCP_KEEPALIVE_GARBAGE_OPT

	// application and service parameters
	NET_INFORMATION_SERVICE_DOMAIN_OPT
	NET_INFORMATION_SERVERS_OPT
	NET_TIME_PROTOCOL_SERVERS_OPT
	VENDOR_SPECIFIC_INFORMATION
	NETBIOS_OVER_TCPIP_NAME_SERVER_OP
	NETBIOS_OVER_TCPIP_DATAGRAM_DISTRIBUTION_SERVER_OPT
	NETBIOS_OVER_TCPIP_NODE_TYPE_OPT
	NETBIOS_OVER_TCPIP_SCOPE_OPT
	XWINDOW_SYSTEM_FONT_SERVER_OPT
	XWINDOW_SYSTEM_DISPLAY_MANAGER_OPT

	// dhcp extensions
	REQUEST_IP_ADDR
	IP_ADDR_LEASE_TIME
	OPT_OVERLOAD
	DHCP_MESSAGE_TYPE
	SERVER_IDENTIFIER
	PARAMETER_REQUEST_LIST
	MESSAGE
	MAX_DHCP_MESSAGE_SIZE
	RENEWAL_TIME_VALUE
	REBINDING_TIME_VALUE
	CLASS_IDENTIFIER
	CLIENT_IDENTIFIER
)

var toString = map[OptionType]string{
	SUBNET_MASK:                  "SUBNET_MASK",
	TIME_OFFSET:                  "TIME_OFFSET",
	ROUTER_OPT:                   "ROUTER_OPT",
	TIME_SERVER_OPT:              "TIME_SERVER_OPT",
	NAME_SERVER_OPT:              "NAME_SERVER_OPT",
	DOMAIN_NAME_SERVER_OPT:       "DOMAIN_NAME_SERVER_OPT",
	LOG_SERVER_OPT:               "LOG_SERVER_OPT",
	COOKIE_SERVER_OPT:            "COOKIE_SERVER_OPT",
	LPR_SERVER_OPT:               "LPR_SERVER_OPT",
	IMPRESS_SERVER_OPT:           "IMPRESS_SERVER_OPT",
	RESOURCE_LOCATION_SERVER_OPT: "RESOURCE_LOCATION_SERVER_OPT",
	HOST_NAME_OPT:                "HOST_NAME_OPT",
	BOOT_FILE_SIZE_OPT:           "BOOT_FILE_SIZE_OPT",
	MERIT_DUMP_FILE:              "MERIT_DUMP_FILE",
	DOMAIN_NAME:                  "DOMAIN_NAME",
	SWAP_SERVER:                  "SWAP_SERVER",
	ROOT_PATH:                    "ROOT_PATH",
	EXTENSION_PATH:               "EXTENSION_PATH",
	IP_FORWARDING_OPT:            "IP_FORWARDING_OPT",
	NON_LOCAL_SOURCE_ROUTING_OPT: "NON_LOCAL_SOURCE_ROUTING_OPT",
	POLICY_FILTER_OPT:            "POLICY_FILTER_OPT",
	MAX_DATAGRAM_REASSEMBLY_SIZE: "MAX_DATAGRAM_REASSEMBLY_SIZE",
	DEFAULT_IP_TTL:               "DEFAULT_IP_TTL",
	PATH_MTU_AGING_TIMEOUT_OPT:   "PATH_MTU_AGING_TIMEOUT_OPT",
	PATH_MTU_PLATEAU_TABLE_OPT:   "PATH_MTU_PLATEAU_TABLE_OPT",

	INTERFACE_MTU_OPT:            "INTERFACE_MTU_OPT",
	ALL_SUBNET_ARE_LOCAL_OPT:     "ALL_SUBNET_ARE_LOCAL_OPT",
	BROADCAST_ADDR_OPT:           "BROADCAST_ADDR_OPT",
	PERFORM_MASK_DISCOVERY_OPT:   "PERFORM_MASK_DISCOVERY_OPT",
	MASK_SUPPLIER_OPT:            "MASK_SUPPLIER_OPT",
	PERFORM_ROUTER_DICOVERY_OPT:  "PERFORM_ROUTER_DICOVERY_OPT",
	ROUTER_SOLICITATION_ADDR_OPT: "ROUTER_SOLICITATION_ADDR_OPT",
	STATIC_ROUTE_OPT:             "STATIC_ROUTE_OPT",

	// link layer parameters per interface
	TRAILER_ENCAPSULATION_OPT:  "TRAILER_ENCAPSULATION_OPT",
	ARP_CACHE_TIMEOUT_OPT:      "ARP_CACHE_TIMEOUT_OPT",
	ETHERNET_ENCAPSULATION_OPT: "ETHERNET_ENCAPSULATION_OPT",

	// tcp parameters
	TCP_DEFAULT_TTL_OPT:        "TCP_DEFAULT_TTL_OPT",
	TCP_KEAPALIVE_INTERVAL_OPT: "TCP_KEAPALIVE_INTERVAL_OPT",
	TCP_KEEPALIVE_GARBAGE_OPT:  "TCP_KEEPALIVE_GARBAGE_OPT",

	// application and service parameters
	NET_INFORMATION_SERVICE_DOMAIN_OPT:                  "NET_INFORMATION_SERVICE_DOMAIN_OPT",
	NET_INFORMATION_SERVERS_OPT:                         "NET_INFORMATION_SERVERS_OPT",
	NET_TIME_PROTOCOL_SERVERS_OPT:                       "NET_TIME_PROTOCOL_SERVERS_OPT",
	VENDOR_SPECIFIC_INFORMATION:                         "VENDOR_SPECIFIC_INFORMATION",
	NETBIOS_OVER_TCPIP_NAME_SERVER_OP:                   "NETBIOS_OVER_TCPIP_NAME_SERVER_OP",
	NETBIOS_OVER_TCPIP_DATAGRAM_DISTRIBUTION_SERVER_OPT: "NETBIOS_OVER_TCPIP_DATAGRAM_DISTRIBUTION_SERVER_OPT",

	NETBIOS_OVER_TCPIP_NODE_TYPE_OPT:   "NETBIOS_OVER_TCPIP_NODE_TYPE_OPT",
	NETBIOS_OVER_TCPIP_SCOPE_OPT:       "NETBIOS_OVER_TCPIP_SCOPE_OPT",
	XWINDOW_SYSTEM_FONT_SERVER_OPT:     "XWINDOW_SYSTEM_FONT_SERVER_OPT",
	XWINDOW_SYSTEM_DISPLAY_MANAGER_OPT: "XWINDOW_SYSTEM_DISPLAY_MANAGER_OPT",

	// dhcp extensions
	REQUEST_IP_ADDR:        "REQUEST_IP_ADDR",
	IP_ADDR_LEASE_TIME:     "IP_ADDR_LEASE_TIME",
	OPT_OVERLOAD:           "OPT_OVERLOAD",
	DHCP_MESSAGE_TYPE:      "DHCP_MESSAGE_TYPE",
	SERVER_IDENTIFIER:      "SERVER_IDENTIFIER",
	PARAMETER_REQUEST_LIST: "PARAMETER_REQUEST_LIST",
	MESSAGE:                "MESSAGE",
	MAX_DHCP_MESSAGE_SIZE:  "MAX_DHCP_MESSAGE_SIZE",
	RENEWAL_TIME_VALUE:     "RENEWAL_TIME_VALUE",
	REBINDING_TIME_VALUE:   "REBINDING_TIME_VALUE",
	CLASS_IDENTIFIER:       "CLASS_IDENTIFIER",
	CLIENT_IDENTIFIER:      "CLIENT_IDENTIFIER",
}

var toStringConverters = map[OptionType]optionToStringConverter{
	REQUEST_IP_ADDR:        requestIpAddrConverter,
	IP_ADDR_LEASE_TIME:     ipAddrLeaseTimeConverter,
	DHCP_MESSAGE_TYPE:      messageTypeStringConverter,
	SERVER_IDENTIFIER:      serverIdentifierConverter,
	RENEWAL_TIME_VALUE:     renewTimeConverter,
	REBINDING_TIME_VALUE:   rebindTimeConverter,
	SUBNET_MASK:            subnetMaskConverter,
	ROUTER_OPT:             routerConverter,
	DOMAIN_NAME_SERVER_OPT: domainNameServerConverter,
	DOMAIN_NAME:            domainNameConverter,
}

func (t OptionType) String() string {
	return TypeToString(t)
}

type DHCPOption struct {
	Data []byte
	ID   string
}

type DHCPOptionI interface {
	GetData() []byte
	GetId() string
}

func (option DHCPOption) String() string {
	converter, found := toStringConverters[OptionType(option.Data[0])]

	if !found {
		converter = defaultConverter
	}

	return converter(option)
}

func (option DHCPOption) MarshalJSON() ([]byte, error) {
	converter, found := toStringConverters[OptionType(option.Data[0])]

	if !found {
		converter = defaultConverter
	}

	return []byte(converter(option)), nil
}

func (o DHCPOption) GetDataAsUint() uint {
	return uint(binary.BigEndian.Uint32(o.GetRawOptionValue()))
}

func (o DHCPOption) GetDataAsIP4() net.IP {
	return o.GetRawOptionValue()
}

func (o DHCPOption) GetDataAsSecDuration() time.Duration {
	return time.Duration(binary.BigEndian.Uint32(o.GetRawOptionValue())) * time.Second
}

func (o DHCPOption) GetRawOptionValue() []byte {
	return o.Data[2:]
}

type MessageType int

const (
	DHCPDISCOVER MessageType = iota + 1
	DHCPOFFER
	DHCPREQUEST
	DHCPDECLINE
	DHCPACK
	DHCPNAK
	DHCPRELEASE
	UNKNOWN
)

func NewMessageTypeOpt(messageType MessageType) DHCPOption {
	return DHCPOption{
		Data: []byte{53, 1, byte(messageType)},
		ID:   DHCP_MESSAGE_TYPE.String(),
	}
}

func NewRequestIpAddrOpt(ipAddr net.IP) DHCPOption {
	return DHCPOption{
		Data: append([]byte{50, 4}, ipAddr[:]...),
		ID:   REQUEST_IP_ADDR.String(),
	}
}

func NewT1Opt(i int) DHCPOption {
	return DHCPOption{
		Data: append([]byte{58, 4}, encodeInt(i, 4)...),
		ID:   RENEWAL_TIME_VALUE.String(),
	}
}

func NewT2Opt(i int) DHCPOption {
	return DHCPOption{
		Data: append([]byte{59, 4}, encodeInt(i, 4)...),
		ID:   REBINDING_TIME_VALUE.String(),
	}
}

func NewIpAddrLeaseTime(leaseTime uint32) DHCPOption {
	buffer := make([]byte, 4)
	binary.BigEndian.PutUint32(buffer, leaseTime)
	return DHCPOption{
		Data: append([]byte{51, 4}, buffer...),
		ID:   IP_ADDR_LEASE_TIME.String(),
	}
}

func NewServerIdentifierOpt(ip net.IP) DHCPOption {
	return DHCPOption{
		Data: append([]byte{54, 4}, ip.To4()...),
		ID:   SERVER_IDENTIFIER.String(),
	}
}

func TypeToString(id OptionType) string {
	v, ok := toString[id]

	if !ok {
		return "UNKNOWN " + strconv.Itoa(int(id))
	}

	return v
}

func encodeInt(i int, s int) []byte {
	buffer := make([]byte, s)
	binary.BigEndian.PutUint32(buffer, uint32(i))

	return buffer
}

func (m MessageType) String() string {
	t := []string{"DHCPDISCOVER", "DHCPOFFER", "DHCPREQUEST", "DHCPDECLINE", "DHCPACK", "DHCPNAK", "DHCPRELEASE", "UNKNOWN"}

	return t[m-1]
}
