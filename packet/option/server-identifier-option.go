package option

import "net"

type ServerIdentifierOpt struct {
	Data []byte
	ID   string
}

func (s ServerIdentifierOpt) ServerIdentifier() net.IP {
	return s.Data
}
