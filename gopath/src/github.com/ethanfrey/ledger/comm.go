package ledger

import "fmt"

func (l *Ledger) Exchange(command []byte, timeout int) ([]byte, error) {
	adpu := WrapCommandAPDU(Channel, command, PacketSize, false)

	// write all the packets
	err := l.device.Write(adpu[:PacketSize])
	if err != nil {
		return nil, err
	}
	for len(adpu) > PacketSize {
		adpu = adpu[PacketSize:]
		err = l.device.Write(adpu[:PacketSize])
		if err != nil {
			return nil, err
		}
	}

	input := l.device.ReadCh()
	response, err := UnwrapResponseAPDU(Channel, input, PacketSize, false)

	swOffset := len(response) - 2
	sw := codec.Uint16(response[swOffset:])
	if sw != 0x9000 {
		return nil, fmt.Errorf("Invalid status %04x", sw)
	}
	return response[:swOffset], nil
}
