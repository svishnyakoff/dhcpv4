package option

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
)

type optionToStringConverter func(DHCPOption) string

func messageTypeStringConverter(d DHCPOption) string {
	shape := struct {
		ID          string
		MessageType string
	}{
		OptionType(d.Data[0]).String(),
		MessageType(d.Data[2]).String(),
	}

	return toJsonString(shape)
}

func requestIpAddrConverter(d DHCPOption) string {
	shape := struct {
		ID string
		IP string
	}{
		OptionType(d.Data[0]).String(),
		net.IP(d.Data[2:]).String(),
	}

	return toJsonString(shape)
}

func ipAddrLeaseTimeConverter(d DHCPOption) string {
	shape := struct {
		ID        string
		LeaseTime string
	}{
		OptionType(d.Data[0]).String(),
		fmt.Sprint(binary.BigEndian.Uint32(d.Data[2:]), "sec"),
	}

	return toJsonString(shape)
}

func serverIdentifierConverter(d DHCPOption) string {
	shape := struct {
		ID       string
		ServerId string
	}{
		OptionType(d.Data[0]).String(),
		net.IP(d.Data[2:]).String(),
	}

	return toJsonString(shape)
}

func renewTimeConverter(d DHCPOption) string {
	shape := struct {
		ID   string
		Time int
	}{
		OptionType(d.Data[0]).String(),
		dataToInt(d),
	}

	return toJsonString(shape)
}

func rebindTimeConverter(d DHCPOption) string {
	shape := struct {
		ID   string
		Time int
	}{
		OptionType(d.Data[0]).String(),
		dataToInt(d),
	}

	return toJsonString(shape)
}

func subnetMaskConverter(d DHCPOption) string {
	shape := struct {
		ID         string
		SubnetMask net.IP
	}{
		OptionType(d.Data[0]).String(),
		d.Data[2:],
	}

	return toJsonString(shape)
}

func routerConverter(d DHCPOption) string {
	parseRouters := func(bytes []byte) []net.IP {
		res := make([]net.IP, 0, 10)

		for i := 0; i < len(bytes); i += 4 {
			res = append(res, bytes[i:i+4])
		}

		return res
	}

	shape := struct {
		ID      string
		Routers []net.IP
	}{
		OptionType(d.Data[0]).String(),
		parseRouters(d.Data[2:]),
	}

	return toJsonString(shape)
}

func domainNameServerConverter(d DHCPOption) string {
	parseDns := func(bytes []byte) []net.IP {
		res := make([]net.IP, 0, 10)

		j := 0
		for i := 0; i < len(bytes); i += 4 {
			j++
			res = append(res, bytes[i:i+4])
		}

		return res
	}

	shape := struct {
		ID         string
		DnsServers []net.IP
	}{
		OptionType(d.Data[0]).String(),
		parseDns(d.Data[2:]),
	}

	return toJsonString(shape)
}

func domainNameConverter(d DHCPOption) string {
	shape := struct {
		ID  string
		Dns string
	}{
		OptionType(d.Data[0]).String(),
		string(d.Data[2:]),
	}

	return toJsonString(shape)
}

func defaultConverter(d DHCPOption) string {
	shape := struct {
		ID string
	}{
		OptionType(d.Data[0]).String(),
	}

	return toJsonString(shape)
}

func dataToInt(d DHCPOption) int {
	return int(binary.BigEndian.Uint32(d.Data[2:]))
}

func dataToIpv4(d DHCPOption) net.IP {
	return d.Data[2:]
}

func dataToIpMask(d DHCPOption) net.IPMask {
	return d.Data[2:]
}

func toJsonString(s interface{}) string {
	bytes, _ := json.Marshal(s)
	return string(bytes)
}
