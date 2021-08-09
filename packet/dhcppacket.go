package packet

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/svishnyakoff/dhcpv4/packet/option"
	"log"
	"net"
	"strconv"
	"strings"
)

const (
	REQUEST = 1
	REPLY   = 2
)

type DHCPPacket struct {
	Op    byte // 1 = BOOTREQUEST, 2 = BOOTREPLY
	Htype byte // Hardware address type
	Hlen  byte // Hardware address length
	Hops  byte // optionally used by relay agents when booting via a relay agent

	Xid uint32 //a random number chosen by the client,
	// used by the client and server to associate messages and responses between a client and a server.

	Secs  uint16
	Flags uint16 // broadcast bit need to be send to 1 during discover/request steps

	Ciaddr [4]byte // client IP addr if previously assigned
	Yiaddr [4]byte // 'your' (client) IP address.
	Siaddr [4]byte
	Giaddr [4]byte  // Relay agent IP address
	Chaddr [16]byte // client hardware address

	Sname [64]byte // server host name -optional
	File  [128]byte

	// DHCP message type option is required
	options []byte // first 4 octets must be 99, 130, 83, and 99
}

func (packet DHCPPacket) Encode() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, packet.Op)
	binary.Write(buf, binary.BigEndian, packet.Htype)
	binary.Write(buf, binary.BigEndian, packet.Hlen)
	binary.Write(buf, binary.BigEndian, packet.Hops)

	binary.Write(buf, binary.BigEndian, packet.Xid)

	binary.Write(buf, binary.BigEndian, packet.Secs)
	binary.Write(buf, binary.BigEndian, packet.Flags)

	binary.Write(buf, binary.BigEndian, packet.Ciaddr)
	binary.Write(buf, binary.BigEndian, packet.Yiaddr)
	binary.Write(buf, binary.BigEndian, packet.Siaddr)
	binary.Write(buf, binary.BigEndian, packet.Giaddr)
	binary.Write(buf, binary.BigEndian, packet.Chaddr)

	binary.Write(buf, binary.BigEndian, packet.Sname)
	binary.Write(buf, binary.BigEndian, packet.File)

	options := make([]byte, 0, len(packet.options)+7)
	options = append(options, 99, 130, 83, 99) // magic cookie
	options = append(options, packet.options[:]...)
	options = append(options, 255, 0, 0) // "end" tag

	binary.Write(buf, binary.BigEndian, options)

	return buf.Bytes()
}

func Decode(buf []byte, bytesRead int) (packet DHCPPacket, e error) {

	defer func() {
		if r := recover(); r != nil {
			packet = DHCPPacket{}
			if er, ok := r.(error); ok {
				e = errors.New("DHCP command parse error: " + er.Error())
			} else {
				e = fmt.Errorf("DHCP command parse error: %v", r)
			}
		}
	}()

	handleReadError := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	b := bytes.NewReader(buf[:bytesRead])
	packet = DHCPPacket{}

	handleReadError(binary.Read(b, binary.BigEndian, &packet.Op))
	handleReadError(binary.Read(b, binary.BigEndian, &packet.Htype))
	handleReadError(binary.Read(b, binary.BigEndian, &packet.Hlen))
	handleReadError(binary.Read(b, binary.BigEndian, &packet.Hops))

	handleReadError(binary.Read(b, binary.BigEndian, &packet.Xid))

	handleReadError(binary.Read(b, binary.BigEndian, &packet.Secs))
	handleReadError(binary.Read(b, binary.BigEndian, &packet.Flags))

	handleReadError(binary.Read(b, binary.BigEndian, &packet.Ciaddr))
	handleReadError(binary.Read(b, binary.BigEndian, &packet.Yiaddr))
	handleReadError(binary.Read(b, binary.BigEndian, &packet.Siaddr))
	handleReadError(binary.Read(b, binary.BigEndian, &packet.Giaddr))
	handleReadError(binary.Read(b, binary.BigEndian, &packet.Chaddr))

	handleReadError(binary.Read(b, binary.BigEndian, &packet.Sname))
	handleReadError(binary.Read(b, binary.BigEndian, &packet.File))

	packet.options = make([]byte, b.Len())
	handleReadError(binary.Read(b, binary.BigEndian, packet.options))

	// ignore magic cookie and end tag
	packet.options = packet.options[4 : len(packet.options)-3]

	return packet, nil
}

func (packet *DHCPPacket) AddOption(option DHCPOption) {
	if packet.options == nil {
		packet.options = make([]byte, 0)
	}

	packet.options = append(packet.options, option.Data...)
}

func (packet DHCPPacket) GetOptions() []DHCPOption {
	options := packet.options

	decodedOptions := make(map[OptionType]DHCPOption)
	for i := 0; i < len(options); {
		optionId := OptionType(options[i])
		optionLen := int(options[i+1])
		option := DHCPOption{}

		nextOptionIndex := i + 2 + optionLen
		option.ID = TypeToString(optionId)
		//extract option subarray one by one and shift indexes to point next option
		option.Data = options[i:nextOptionIndex]
		i = nextOptionIndex

		existingOption, ok := decodedOptions[optionId]

		if !ok {
			decodedOptions[optionId] = option
		} else {
			// https://datatracker.ietf.org/doc/html/rfc3396
			// In the case that a decoding agent finds a split option, it MUST treat
			// the contents of that option as a single option
			existingOption.Data = append(existingOption.Data, option.Data[2:]...)
			existingOption.Data[1] += option.Data[1]
		}
	}

	result := make([]DHCPOption, 0, 10)

	for _, value := range decodedOptions {
		result = append(result, value)
	}

	return result
}

func (packet DHCPPacket) GetMessageType() MessageType {
	opt := packet.GetOption(DHCP_MESSAGE_TYPE)

	if opt == nil {
		return UNKNOWN
	}

	return MessageType(opt.Data[2])
}

func (packet DHCPPacket) IsPacketOfType(messageType MessageType) bool {
	t := packet.GetMessageType()

	return t == messageType
}

func (packet DHCPPacket) GetOption(optionType OptionType) *DHCPOption {
	for _, opt := range packet.GetOptions() {
		if opt.ID == optionType.String() {
			return &opt
		}
	}

	return nil
}

func (packet DHCPPacket) BroadcastFlag() bool {
	return (1 << 15 & packet.Flags) != 0
}

// Broadcast flag lets server know that client does not have IP configured and server need to broacast reply
func (packet *DHCPPacket) MarkBroadcastFlag() {
	packet.Flags = uint16(1 << 15)
}

func (packet DHCPPacket) String() string {
	asShortString := func(a []byte) string {

		return strings.TrimFunc(string(a), func(r rune) bool {
			return r == 0
		})
	}

	k := toStringStruct{
		Htype:   packet.Htype,
		Hlen:    packet.Hlen,
		Hops:    packet.Hops,
		Xid:     packet.Xid,
		Secs:    packet.Secs,
		Flags:   strconv.FormatInt(int64(packet.Flags), 2),
		Ciaddr:  fmt.Sprintf("%v", net.IP(packet.Ciaddr[:])),
		Yiaddr:  fmt.Sprintf("%v", net.IP(packet.Yiaddr[:])),
		Siaddr:  fmt.Sprintf("%v", net.IP(packet.Siaddr[:])),
		Giaddr:  fmt.Sprintf("%v", net.IP(packet.Giaddr[:])),
		Chaddr:  fmt.Sprintf("%v", net.HardwareAddr(packet.Chaddr[:packet.Hlen])),
		Sname:   asShortString(packet.Sname[:]),
		File:    asShortString(packet.File[:]),
		Options: packet.GetOptions(),
	}

	if packet.Op == 1 {
		k.Op = "request"
	} else {
		k.Op = "response"
	}

	// todo continue readin documentation
	marshal, err := json.Marshal(k)

	if err != nil {
		log.Println("broken DHCP packet", err)
		return "broken DHCP packet"
	}

	return string(marshal)
}

type toStringStruct struct {
	Op    string
	Htype byte
	Hlen  byte
	Hops  byte

	Xid uint32

	Secs  uint16
	Flags string

	Ciaddr string
	Yiaddr string
	Siaddr string
	Giaddr string
	Chaddr string

	Sname string
	File  string

	Options []DHCPOption
}
