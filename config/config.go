package config

import (
	_ "github.com/joho/godotenv/autoload"
	"log"
	"os"
)
import "github.com/caarlos0/env"

type DHCPConfig struct {
	OfferWindowSec                int    `env:"OfferWindowSec" envDefault:"1"`
	HardwareType                  int    `env:"HardwareType" envDefault:"1"`
	HardwareAddrLen               int    `env:"HardwareAddrLen" envDefault:"6"`
	InterfaceName                 string `env:"InterfaceName" envDefault:"en0"`
	RetryRequestSec               int    `env:"RetryRequestSec" envDefault:"3"`
	StopOnLeaseAcquisitionFailure bool   `env:"StopOnLeaseAcquisitionFailure" envDefault:"false"`
}

var GlobalDHCPConfig, _ = LoadConfig()

func LoadConfig() (config DHCPConfig, err error) {
	log.Println("Loading DHCP configurations")
	config = DHCPConfig{}

	if err := env.Parse(&config); err != nil {
		return config, err
	}

	return config, nil
}

func printEnv() {
	for _, e := range os.Environ() {
		log.Println(e)
	}
}
