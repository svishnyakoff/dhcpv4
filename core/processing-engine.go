package core

import (
	"fmt"
	"github.com/emirpasic/gods/lists"
	"github.com/emirpasic/gods/lists/arraylist"
	"github.com/svishnyakoff/dhcpv4/client/state"
	configuration "github.com/svishnyakoff/dhcpv4/config"
	. "github.com/svishnyakoff/dhcpv4/lease"
	"github.com/svishnyakoff/dhcpv4/packet"
	"github.com/svishnyakoff/dhcpv4/packet/option"
	"github.com/svishnyakoff/dhcpv4/transaction"
	netUtils "github.com/svishnyakoff/dhcpv4/util/net-utils"
	"github.com/svishnyakoff/dhcpv4/util/timers"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

type decodeError struct {
	msg string
}

func (d decodeError) Error() string {
	return d.msg
}

type ProcessingEngine struct {
	Client                 *DHCPClient
	Config                 configuration.DHCPConfig
	Lease                  DHCPLease
	lock                   *sync.Mutex
	terminate              chan int
	stopped                bool
	renewTimer             *time.Timer
	rebindTimer            *time.Timer
	leaseReceivedListeners []func(lease DHCPLease)
	leaseRenewedListeners  []func(lease DHCPLease)
}

type ProcessingEngineInitProps struct {
	Client *DHCPClient
	Config *configuration.DHCPConfig
	Lease  *DHCPLease
}

func NewProcessingEngine(initProps ProcessingEngineInitProps) *ProcessingEngine {
	if initProps.Client == nil {
		initProps.Client = &DHCPClient{}
	}

	if initProps.Lease == nil {
		l := LoadLease()
		initProps.Lease = &l
	}

	if initProps.Config == nil {
		conf, err := configuration.LoadConfig()
		if err != nil {
			panic(err)
		}

		initProps.Config = &conf
	}

	lock := &sync.Mutex{}

	return &ProcessingEngine{
		Client:      initProps.Client,
		Config:      *initProps.Config,
		Lease:       *initProps.Lease,
		lock:        lock,
		terminate:   make(chan int),
		renewTimer:  time.NewTimer(9999 * time.Hour),
		rebindTimer: time.NewTimer(9999 * time.Hour),
	}
}

func (p *ProcessingEngine) GetLease() DHCPLease {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.Lease
}

func (p *ProcessingEngine) Stop() {
	if p.stopped {
		return
	}

	log.Println("Terminating processing engine")
	p.stopped = true
	timers.SafeStop(p.renewTimer)
	timers.SafeStop(p.rebindTimer)
	p.Client.Stop()
	close(p.terminate)
}

func (p *ProcessingEngine) Start() {
	client := p.Client
	client.Listen()

	// todo The client SHOULD wait a random time between one and ten seconds to
	//   desynchronize the use of DHCP at startup.
	// https://datatracker.ietf.org/doc/html/rfc2131#page-34
	p.normalizeStateAfterStart()
	processInput := func() {

		switch p.Lease.State {
		case state.INIT:
			p.Discover()
		case state.BOUND:
			renewTime := timers.SafeReset(p.renewTimer, p.Lease.DurationUntilRenew())
			rebindTime := timers.SafeReset(p.rebindTimer, p.Lease.DurationUntilRebind())
			log.Println("renew is scheduled in", renewTime)
			log.Println("rebind is scheduled in", rebindTime)
			p.RenewOrRebindLease()
		case state.INIT_REBOOT:
			p.RenewAfterReboot()
		case state.RENEWING, state.REBINDING, state.REBOOTING:
			p.UpdateState(state.INIT_REBOOT)
		}
	}

	go func() {
		for {
			select {
			case <-p.terminate:
				// Stop has been called
				log.Println("Processing engine has been stopped")
				return
			default:
				processInput()
			}
		}
	}()
}

func (p *ProcessingEngine) Discover() {
	c := p.Client
	config := p.Config
	maxRetries := 2

	data, tx := p.packetFactory().Discover()

	if err := c.Send(*data, net.IPv4bcast); err != nil {
		// todo integration test - discover sent failed

		log.Println("send of discover command failed:", err)

		return
	}

	p.UpdateState(state.SELECTING)

	offers := p.readOffers(tx)

	if offers.Size() > 0 {
		var err error
		i := 0
		for ; i < maxRetries+1 && !offers.Empty(); i++ {
			err = p.ProcessOffers(offers)
			if err == nil {
				return
			}

			log.Println("error processing offers: attempt", i+1, err)
			waitSec(p.Config.RetryRequestSec)
		}

		log.Printf("offer was not ack by server after %d attempts.\n", i)
		p.UpdateState(state.INIT)
		p.onLeaseAcquisitionFailure()
		return
	}

	log.Println("DHCP client did not receive any offer during time interval:", config.OfferWindowSec, "sec")
	p.UpdateState(state.INIT)
	p.onLeaseAcquisitionFailure()
}

func (p *ProcessingEngine) onLeaseAcquisitionFailure() {
	if p.Config.StopOnLeaseAcquisitionFailure {
		log.Println("stop further attempts to acquire lease due to StopOnLeaseAcquisitionFailure is set to true. ")
		p.Stop()
	}
}

func (p *ProcessingEngine) onLeaseReceived() {
	for _, listener := range p.leaseReceivedListeners {
		listener(p.Lease)
	}
}

func (p *ProcessingEngine) onLeaseRenewed() {
	for _, listener := range p.leaseRenewedListeners {
		listener(p.Lease)
	}
}

func (p *ProcessingEngine) ProcessOffers(offers lists.List) error {

	packetFactory := p.packetFactory()
	offerPacket, _ := offers.Get(0)
	serverIdentifier := offerPacket.(packet.DHCPPacket).GetOption(option.SERVER_IDENTIFIER)

	if serverIdentifier == nil {
		return fmt.Errorf("DHCP server sent offer without server identifier")
	}

	requestTime := time.Now()
	requestPacket, reqTx := packetFactory.RequestForOffer(offerPacket.(packet.DHCPPacket))
	if err := p.Client.Send(*requestPacket, net.IPv4bcast); err != nil {
		log.Println("sending request on offer failed:", err)

		return err
	}

	p.UpdateState(state.REQUESTING)

	var waitForAck func() error

	waitForAck = func() error {
		ack, err := p.WaitForEvent(reqTx, time.Second)

		if err != nil && os.IsTimeout(err) {
			return fmt.Errorf("server did not respond to request within %v\n", time.Second)
		}

		if err != nil {
			return fmt.Errorf("error while waiting for offer ack or nack packets: %v\n", err)
		}
		if !ack.IsPacketOfType(option.DHCPACK) && !ack.IsPacketOfType(option.DHCPNAK) {
			log.Println("expected ACK or NACK but got", ack.GetMessageType(), "keep waiting for either ack or nack")
			return waitForAck()
		}

		if ack.IsPacketOfType(option.DHCPACK) {
			if err = p.FinalizeOffer(&ack, requestTime); err != nil {
				return err
			}
			p.UpdateState(state.BOUND)
			p.onLeaseReceived()
		}

		if ack.IsPacketOfType(option.DHCPNAK) {
			return fmt.Errorf("server revoked the lease:%v\n", offerPacket.(packet.DHCPPacket).GetOption(option.SERVER_IDENTIFIER))
		}

		return nil
	}

	return waitForAck()
}

func (p *ProcessingEngine) FinalizeOffer(ack *packet.DHCPPacket, requestTime time.Time) error {
	if !netUtils.IsUniqueIp(ack.Yiaddr[:], p.Config) {
		declinePacket, _ := p.packetFactory().Decline(*ack)
		if err := p.Client.Send(*declinePacket, ack.GetOption(option.SERVER_IDENTIFIER).GetDataAsIP4()); err != nil {
			log.Println("was not able to send DHCPDECLINE", err)
		}
		return fmt.Errorf("address offerred by DHCP server already present on local network segement: %v\n",
			net.IP(ack.Yiaddr[:]))
	}

	serverIdentifier := ack.GetOption(option.SERVER_IDENTIFIER)
	p.Lease.ServerIdentifier = serverIdentifier.GetDataAsIP4()
	p.Lease.IpAddr = ack.Yiaddr[:]
	p.Lease.LeaseDuration = ack.GetOption(option.IP_ADDR_LEASE_TIME).GetDataAsSecDuration()
	p.Lease.LeaseInitTime = requestTime

	if t1 := ack.GetOption(option.RENEWAL_TIME_VALUE); t1 != nil {
		p.Lease.T1 = t1.GetDataAsSecDuration()
	} else {
		p.Lease.T1 = p.Lease.LeaseDuration / 2
	}

	if t2 := ack.GetOption(option.REBINDING_TIME_VALUE); t2 != nil {
		p.Lease.T2 = t2.GetDataAsSecDuration()
	} else {
		p.Lease.T2 = time.Duration(float64(p.Lease.LeaseDuration) * 0.875)
	}

	return nil
}

func (p *ProcessingEngine) UpdateState(newState state.State) {
	log.Println("State change:", p.Lease.State, "->", newState)
	p.Lease.State = newState
	switch newState {
	case state.INIT:
		p.Lease.ResetLease()
	}
}

func (p *ProcessingEngine) RenewAfterReboot() {
	lease := p.Lease
	packetFactory := p.packetFactory()

	if lease.State != state.INIT_REBOOT {
		log.Println("Cannot renew the lease from state", lease.State)
		return
	}

	requestTime := time.Now()
	requestPacket, tx := packetFactory.RequestForReboot(lease)

	if err := p.Client.Send(*requestPacket, net.IPv4bcast); err != nil {
		log.Printf("request failed when tried to renew lease: %v\n", err)
		p.UpdateState(state.INIT)
		return
	}

	response, err := p.WaitForEvent(tx, time.Second)
	p.handleRenewResponse(response, requestTime, err)
}

func (p *ProcessingEngine) handleRenewResponse(response packet.DHCPPacket, requestTime time.Time, err error) {
	if err != nil {
		log.Println("error reading response for renew", err)
		p.UpdateState(state.INIT)
		return
	}

	if response.IsPacketOfType(option.DHCPNAK) {
		log.Println("server declined to renew lease")
		p.UpdateState(state.INIT)
		return
	} else if response.IsPacketOfType(option.DHCPACK) {
		if err := p.FinalizeOffer(&response, requestTime); err != nil {
			log.Println("Probably BUG: it seems some other host within local network has the same IP address as"+
				" the Lease's IP address we just renewed", err)
			p.UpdateState(state.INIT)
			return
		}

		log.Println("successfully renewed lease")
		p.onLeaseRenewed()
		p.UpdateState(state.BOUND)
		return
	}
}

func (p *ProcessingEngine) RenewOrRebindLease() {
	var s state.State = 0
	var renewed bool = false

	if !p.Lease.IsRenewPeriodExpired() {
		s, renewed = p.RenewLease()
	}

	if renewed || s == state.INIT {
		return
	}

	if !p.Lease.IsRebindPeriodExpired() {
		p.RebindLease()
	} else {
		p.UpdateState(state.INIT)
	}

}

func (p *ProcessingEngine) RenewLease() (state.State, bool) {
	p.waitForTimer(p.renewTimer, p.Lease.GetRebindMoment())
	if p.stopped || p.Lease.IsRenewPeriodExpired() {
		return p.Lease.State, false
	}

	log.Println("renewing lease")
	lease := p.Lease
	leaseInitTime := p.Lease.LeaseInitTime
	packetFactory := p.packetFactory()

	renewExpireMoment := leaseInitTime.Add(lease.T2)
	if time.Now().Before(renewExpireMoment) {
		requestTime := time.Now()
		requestPacket, tx := packetFactory.RequestForRenew(lease)

		// todo don't print timeout errors "network error: read udp [::]:68: i/o timeout"
		if err := p.Client.Send(*requestPacket, lease.ServerIdentifier); err != nil {
			log.Printf("request failed when tried to renew lease: %v\n", err)
		}

		p.UpdateState(state.RENEWING)

		response, err := p.WaitForEvent(tx, time.Second)

		if err != nil {
			log.Println("error reading response for renew", err)
			timers.SafeReset(p.renewTimer, p.Lease.DurationUntilRenew())
			return p.RenewLease()
		}

		if response.IsPacketOfType(option.DHCPNAK) {
			log.Println("server declined to renew lease")
			p.UpdateState(state.INIT)
			return state.INIT, false
		} else if response.IsPacketOfType(option.DHCPACK) {
			err := p.FinalizeOffer(&response, requestTime)
			if err != nil {
				log.Println("Probably BUG: it seems some other host within local network has the same IP address as"+
					" the Lease's IP address we just renewed", err)
				p.UpdateState(state.INIT)
				return state.INIT, false
			} else {
				log.Println("successfully renewed lease")
				p.UpdateState(state.BOUND)
				p.onLeaseRenewed()
				return state.BOUND, true
			}
		}
	}

	return p.Lease.State, false
}

func (p *ProcessingEngine) RebindLease() (state.State, bool) {
	p.waitForTimer(p.rebindTimer, p.Lease.GetLeaseExpirationMoment())
	if p.stopped || p.Lease.IsRebindPeriodExpired() {
		return p.Lease.State, false
	}
	log.Println("rebinding lease")
	lease := p.Lease
	leaseInitTime := p.Lease.LeaseInitTime
	packetFactory := p.packetFactory()
	rebindExpireMoment := leaseInitTime.Add(lease.LeaseDuration)

	for time.Now().Before(rebindExpireMoment) {
		requestTime := time.Now()
		requestPacket, tx := packetFactory.RequestForRebind(lease)

		if err := p.Client.Send(*requestPacket, net.IPv4bcast); err != nil {
			log.Printf("request failed when tried to rebind lease: %v\n", err)
		}

		p.UpdateState(state.REBINDING)

		response, err := p.WaitForEvent(tx, time.Second)

		if err != nil {
			log.Println("error reading response for rebind", err)
			continue
		}

		if response.IsPacketOfType(option.DHCPNAK) {
			log.Println("server declined to rebind lease")
			p.UpdateState(state.INIT)
			return state.INIT, false
		} else if response.IsPacketOfType(option.DHCPACK) {
			err := p.FinalizeOffer(&response, requestTime)
			if err != nil {
				log.Println("Probably BUG: it seems some other host within local network has the same IP address as"+
					" the Lease's IP address we just renewed", err)
				p.UpdateState(state.INIT)
				return state.INIT, false
			} else {
				log.Println("successfully rebind lease")
				p.UpdateState(state.BOUND)
				p.onLeaseRenewed()
				return state.BOUND, true
			}
		}
	}

	return p.Lease.State, false
}

func (p *ProcessingEngine) packetFactory() *packet.DHCPPacketFactory {
	return &packet.DHCPPacketFactory{Config: p.Config}
}

func (p *ProcessingEngine) WaitForEvent(tx transaction.TxId, timeout time.Duration) (packet.DHCPPacket, error) {
	return p.WaitForEventUntil(tx, time.Now().Add(timeout))
}

func (p *ProcessingEngine) WaitForEventUntil(tx transaction.TxId, timeout time.Time) (packet.DHCPPacket, error) {
	data, err := p.readPacket(timeout)

	if err != nil {
		return packet.DHCPPacket{}, err
	}

	if data.Xid != tx {
		return p.WaitForEventUntil(tx, timeout)
	}

	return data, nil
}

func (p *ProcessingEngine) AddLeaseReceivedListener(listener func(lease DHCPLease)) {
	p.leaseReceivedListeners = append(p.leaseReceivedListeners, listener)
}

func (p *ProcessingEngine) AddLeaseRenewedListener(listener func(lease DHCPLease)) {
	p.leaseRenewedListeners = append(p.leaseRenewedListeners, listener)
}

func (p *ProcessingEngine) readPacket(timeout time.Time) (packet.DHCPPacket, error) {
	buf := make([]byte, 2000)
	client := p.Client
	bytesRead := 0
	var err error = nil

	for bytesRead <= 0 && err == nil {
		client.conn.SetReadDeadline(timeout)
		bytesRead, _, err = client.conn.ReadFromUDP(buf)
	}

	if err != nil {
		return packet.DHCPPacket{}, err
	}

	data, err := packet.Decode(buf, bytesRead)
	if err != nil {
		return packet.DHCPPacket{}, decodeError{
			msg: fmt.Sprintf("cannot decode dhcp packet: %v", err),
		}
	}

	log.Printf("<--%v\n%v\n\n", data.GetMessageType(), data)

	return data, err
}

func (p *ProcessingEngine) normalizeStateAfterStart() {
	switch p.Lease.State {
	case state.INIT_REBOOT, state.BOUND, state.RENEWING, state.REBINDING, state.REBOOTING, state.SELECTING, state.REQUESTING:
		p.UpdateState(state.INIT_REBOOT)
	}
}

func (p *ProcessingEngine) waitForTimer(t *time.Timer, timeout time.Time) {

	select {
	case <-t.C:
	case <-p.terminate:
	case <-time.After(timeout.Sub(time.Now())):
	}
}

// readOffers waits for offer commands from server for up to MaxOfferWaitTimeSec seconds.
// The optimistic expectation of the method is that offers  will be received in OfferWindowSec interval.
// If at least one offer received during  OfferWindowSec interval,
// to total execution time will be OfferWindowSec and only offers received during this time interval will be returned.
// If optimistic expectation fails, the method will wait for first offer for up to MaxOfferWaitTimeSec seconds.
func (p *ProcessingEngine) readOffers(tx transaction.TxId) (offers lists.List) {
	offers = arraylist.New()
	config := p.Config
	offerTimeoutMoment := time.Now().Add(time.Second * time.Duration(config.MaxOfferWaitTimeSec))
	offerWindowEndMoment := time.Now().Add(time.Second * time.Duration(config.OfferWindowSec))

	for {
		if time.Now().After(offerTimeoutMoment) {
			return
		}

		responsePacket, err := p.WaitForEventUntil(tx, offerTimeoutMoment)
		if err != nil && os.IsTimeout(err) {
			return
		}

		if err == nil && responsePacket.IsPacketOfType(option.DHCPOFFER) {
			offers.Add(responsePacket)
			if time.Now().After(offerWindowEndMoment) {
				return
			}

			offerTimeoutMoment = offerWindowEndMoment
		} else if err != nil {
			log.Println("error while reading offer packets.", err)
			return
		} else {
			log.Println("have been waiting for offer but got", responsePacket.GetMessageType())
		}
	}
}

func wait(t time.Duration) {
	time.Sleep(t)
}

func waitSec(t int) {
	time.Sleep(time.Duration(t) * time.Second)
}
