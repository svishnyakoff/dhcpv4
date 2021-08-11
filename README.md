# dhcpv4 - DHCP client for IPv4

=============================

Go project that provides DHCP v4 client and implements dhcp client behaviour according to 
[RFC-2131](https://datatracker.ietf.org/doc/html/rfc2131). MIT licence.

Any contribution is very welcome including of issue reporting.  

NOTE: the project was created as a pet project to learn GO language and is not mature enough for use in production.

## Examples:

#### How to launch a dhcp client and obtain DHCP lease(IP addr, DNS info, etc)?

```go
// you can customize behaviour of dhcp client by populating ClientProps structure 
// or you can pass empty structure if you need default behaviour
client := dhcpv4.NewDHCPClient(dhcpv4.ClientProps{})
if err := client.Start(); err != nil {
    log.Panic(err)
}

sigs := make(chan os.Signal, 1)

signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

// client will run in background, so following lines are just to avoid premature termination of go app along with 
// DHCP client
sig := <-sigs

log.Println("Gracefully stop client")
log.Println(sig)
client.Stop()
```


#### How to obtain DHCP lease information(IP addr, DNS info, etc)?

```go
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

// rest of code that wait for interruption signal
```

#### How to retrieve an DHCP option that does not have dedicated field in DHCPLease?
```go
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
```
