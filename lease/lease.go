package lease

import (
	"encoding/hex"
	"encoding/json"
	"github.com/go-ini/ini"
	"github.com/svishnyakoff/dhcpv4/packet"
	netUtils "github.com/svishnyakoff/dhcpv4/util/net-utils"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

type DHCPLease struct {
	State            State
	IpAddr           net.IP
	Dns              net.IP
	SubnetMask       net.IPMask
	ServerIdentifier net.IP
	LeaseInitTime    time.Time
	LeaseDuration    time.Duration
	T1               time.Duration
	T2               time.Duration
	Offer            packet.DHCPPacket
}

var initPrivateAddr sync.Once
var autoPrivateAddr *net.IP

func (l *DHCPLease) ResetLease() {
	initPrivateAddr.Do(func() {
		t := netUtils.AutoPrivateAddr()
		autoPrivateAddr = &t
	})

	l.State = INIT
	l.IpAddr = *autoPrivateAddr
	l.Dns = nil
	l.SubnetMask = nil
	l.ServerIdentifier = nil
	l.LeaseInitTime = time.Time{}
	l.LeaseDuration = 0
	l.T1 = 0
	l.T2 = 0

}

func NewDHCPLease() DHCPLease {
	initPrivateAddr.Do(func() {
		t := netUtils.AutoPrivateAddr()
		autoPrivateAddr = &t
	})

	return DHCPLease{
		State:  INIT,
		IpAddr: *autoPrivateAddr,
	}
}

func LoadLease() DHCPLease {
	var leaseFile = os.Getenv("Lease.File")

	if leaseFile == "" {
		log.Println("File with lease dump is not provided. Use \"Lease." +
			"File\" as a path to file with saved lease information")
		// todo figure out where to store lease if file was not past in params
	}

	cfg, err := ini.Load(leaseFile)

	if err != nil {
		return NewDHCPLease()
	}

	l := DHCPLease{}

	l.State = Parse(cfg.Section("").Key("state").In("INIT", StringCandidates()))
	l.IpAddr = net.ParseIP(cfg.Section("").Key("ip").String()).To4()
	l.Dns = net.ParseIP(cfg.Section("").Key("dns").String()).To4()
	l.SubnetMask, _ = hex.DecodeString(cfg.Section("").Key("subnet.mask").String())
	l.ServerIdentifier = net.ParseIP(cfg.Section("").Key("server.ip").String()).To4()
	l.LeaseInitTime = cfg.Section("timers").Key("lease.start").MustTime()
	l.LeaseDuration = cfg.Section("timers").Key("lease.duration").MustDuration()
	l.T1 = cfg.Section("timers").Key("T1").MustDuration()
	l.T2 = cfg.Section("timers").Key("T2").MustDuration()

	return l
}

func (l DHCPLease) String() string {
	bytes, err := json.Marshal(l)

	if err != nil {
		log.Panic("error serializing lease to string:", err)
	}

	return string(bytes)
}

func (l DHCPLease) GetRebindMoment() time.Time {
	return l.LeaseInitTime.Add(l.T2)
}

func (l DHCPLease) GetLeaseExpirationMoment() time.Time {
	return l.LeaseInitTime.Add(l.LeaseDuration)
}

func (l DHCPLease) IsRenewPeriodExpired() bool {
	return time.Now().After(l.LeaseInitTime.Add(l.T2))
}

func (l DHCPLease) DurationUntilRenew() time.Duration {
	t1Moment := l.LeaseInitTime.Add(l.T1)

	if time.Now().Before(t1Moment) {
		return t1Moment.Sub(time.Now())
	}

	return time.Second * 60
}

func (l DHCPLease) IsRebindPeriodExpired() bool {
	return time.Now().After(l.LeaseInitTime.Add(l.LeaseDuration))
}

func (l *DHCPLease) DurationUntilRebind() time.Duration {
	t2Moment := l.LeaseInitTime.Add(l.T2)

	if time.Now().Before(t2Moment) {
		return t2Moment.Sub(time.Now())
	}

	return time.Second * 60
}
