package dhcpv4

import (
	"fmt"
	"github.com/svishnyakoff/dhcpv4/core"
	"github.com/svishnyakoff/dhcpv4/lease"
)

type ClientProps struct {
	core.ProcessingEngineInitProps
}

type DHCPClient struct {
	engine *core.ProcessingEngine
}

func (c *DHCPClient) Start() error {
	err := c.engine.Start()

	if err != nil {
		return fmt.Errorf("error starting dhcp client: %v", err)
	}

	return nil
}

func (c *DHCPClient) Stop() {
	c.engine.Stop()
}

func (c *DHCPClient) OnLeaseReceived(listener func(l lease.DHCPLease)) {
	c.engine.AddLeaseReceivedListener(listener)
}

func (c *DHCPClient) OnLeaseRenewed(listener func(l lease.DHCPLease)) {
	c.engine.AddLeaseRenewedListener(listener)
}

func NewDHCPClient(props ClientProps) *DHCPClient {
	return &DHCPClient{engine: core.NewProcessingEngine(props.ProcessingEngineInitProps)}
}
