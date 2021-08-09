package net_utils

import (
	"github.com/stretchr/testify/assert"
	"github.com/svishnyakoff/dhcpv4/config"
	"net"
	"testing"
)

func TestNotUniqueIp(t *testing.T) {
	ip := net.ParseIP("192.168.1.1")
	isUnique := IsUniqueIp(ip, config.DHCPConfig{InterfaceName: "en0"})

	assert.False(t, isUnique)
}

func TestUniqueIp(t *testing.T) {
	ip := net.ParseIP("192.168.1.254")
	isUnique := IsUniqueIp(ip, config.DHCPConfig{InterfaceName: "en0"})

	assert.True(t, isUnique)
}
