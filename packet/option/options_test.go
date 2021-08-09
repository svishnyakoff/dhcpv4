package option

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestConvertingOptionsToString(t *testing.T) {
	messageTypeOpt := NewMessageTypeOpt(DHCPDISCOVER)
	assert.JSONEq(t, "{\"ID\":\"DHCP_MESSAGE_TYPE\",\"MessageType\":\"DHCPDISCOVER\"}", messageTypeOpt.String())

	requestIpAddOpt := NewRequestIpAddrOpt(net.IPv4bcast)
	assert.JSONEq(t, "{\"ID\":\"REQUEST_IP_ADDR\", \"IP\":\"255.255.255.255\"}", requestIpAddOpt.String())
}
