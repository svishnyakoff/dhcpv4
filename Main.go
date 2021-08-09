package main

import (
	"github.com/svishnyakoff/dhcpv4/core"
	"github.com/svishnyakoff/dhcpv4/lease"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	leaseListener := func(l lease.DHCPLease) {
		log.Println("got new lease\n", l)
	}
	processingEngine := core.NewProcessingEngine(core.ProcessingEngineInitProps{})
	processingEngine.AddLeaseReceivedListener(leaseListener)
	processingEngine.Start()

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigs

	log.Println("Gracefully stop client")
	log.Println(sig)
	processingEngine.Stop()

}
