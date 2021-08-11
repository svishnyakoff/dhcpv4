package option

import "net"

func DnsParser(o DHCPOption) []net.IP {
	parseDns := func(bytes []byte) []net.IP {
		res := make([]net.IP, 0, 10)

		j := 0
		for i := 0; i < len(bytes); i += 4 {
			j++
			res = append(res, bytes[i:i+4])
		}

		return res
	}

	return parseDns(o.Data[2:])
}
