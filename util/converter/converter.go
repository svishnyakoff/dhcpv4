package converter

import "net"

func IP2Array(ip net.IP) [4]byte {
	var arr [4]byte

	copy(arr[:], ip)

	return arr
}

func Hardware2Array(addr net.HardwareAddr) [16]byte {
	var arr [16]byte

	copy(arr[:], addr)

	return arr
}

func slice2Array(ip net.IP) [4]byte {
	var arr [4]byte

	copy(arr[:], ip)

	return arr
}
