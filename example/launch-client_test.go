package example

import (
	"github.com/svishnyakoff/dhcpv4"
	"github.com/svishnyakoff/dhcpv4/lease"
	"github.com/svishnyakoff/dhcpv4/packet/option"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestLaunchClientExample(t *testing.T) {
	// you can customize behaviour of dhcp client by populating ClientProps structure or you can pass empty structure
	// if you need default behaviour
	client := dhcpv4.NewDHCPClient(dhcpv4.ClientProps{})
	if err := client.Start(); err != nil {
		log.Panic(err)
	}

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigs

	log.Println("Gracefully stop client")
	log.Println(sig)
	client.Stop()
}

func TestRetrieveLease(t *testing.T) {
	client := dhcpv4.NewDHCPClient(dhcpv4.ClientProps{})
	if err := client.Start(); err != nil {
		log.Panic(err)
	}

	// OnLeaseReceived allows to specify callback that get executed once lease is obtained from server
	client.OnLeaseReceived(func(l lease.DHCPLease) {
		log.Println("IP address", l.IpAddr)
		log.Println("Subnet mask", l.SubnetMask)
		log.Println("Lease duration", l.LeaseDuration)
		log.Println("Raw lease offer", l.Offer)
	})

	// OnLeaseRenewed set callback that will be executed when lease is successfully renewed
	client.OnLeaseRenewed(func(l lease.DHCPLease) {
		log.Println("IP address", l.IpAddr)
		log.Println("Subnet mask", l.SubnetMask)
		log.Println("Lease duration", l.LeaseDuration)
		log.Println("Raw lease offer", l.Offer)
	})

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigs

	log.Println("Gracefully stop client")
	log.Println(sig)
	client.Stop()
}

func TestRetrieveLeaseOption(t *testing.T) {
	client := dhcpv4.NewDHCPClient(dhcpv4.ClientProps{})
	if err := client.Start(); err != nil {
		log.Panic(err)
	}

	client.OnLeaseReceived(func(l lease.DHCPLease) {
		// say DHCP server sends us ROUTER option, then we can extract it from DHCP offer as following
		routerOption := l.Offer.GetOption(option.ROUTER_OPT)

		// according to https://datatracker.ietf.org/doc/html/rfc1533#section-3.5
		// router option contains a list of routers ip addresses. Each DHCPOption contains raw byte slice representing
		// option content and a set of utility methods to help convert option content to more convenient format
		log.Println("Routers:", routerOption.GetDataAsIP4Slice())
	})

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigs

	log.Println("Gracefully stop client")
	log.Println(sig)
	client.Stop()
}
