package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {

	config, err := LoadConfig()

	assert.NoError(t, err)
	assert.Equal(t, DHCPConfig{
		OfferWindowSec:  1,
		HardwareAddrLen: 6,
		HardwareType:    1,
		InterfaceName:   "en0",
	}, config)
}

func TestLoadConfigWhenEnvSet(t *testing.T) {
	os.Setenv("OfferWindowSec", "2")
	os.Setenv("HardwareType", "3")
	os.Setenv("HardwareAddrLen", "7")
	os.Setenv("InterfaceName", "lo0")
	config, err := LoadConfig()

	assert.NoError(t, err)
	assert.Equal(t, DHCPConfig{
		OfferWindowSec:  2,
		HardwareAddrLen: 7,
		HardwareType:    3,
		InterfaceName:   "lo0",
	}, config)
}
