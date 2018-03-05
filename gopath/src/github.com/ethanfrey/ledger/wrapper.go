package ledger

import (
	"encoding/binary"
	"errors"
)

var codec = binary.BigEndian

// WrapCommandAPDU turns the command into a sequence of 64 byte packets
func WrapCommandAPDU(channel uint16, command []byte, packetSize int, ble bool) []byte {
	if packetSize < 3 {
		panic("packet size must be at least 3")
	}

	var sequenceIdx uint16
	var offset, extraHeaderSize, blockSize int
	var result = make([]byte, 64)
	var buf = result

	if !ble {
		codec.PutUint16(buf, channel)
		extraHeaderSize = 2
		buf = buf[2:]
	}

	buf[0] = 0x05
	codec.PutUint16(buf[1:], sequenceIdx)
	codec.PutUint16(buf[3:], uint16(len(command)))
	sequenceIdx++
	buf = buf[5:]

	blockSize = packetSize - 5 - extraHeaderSize
	copy(buf, command)
	offset += blockSize

	for offset < len(command) {
		// TODO: optimize this
		end := len(result)
		result = append(result, make([]byte, 64)...)
		buf = result[end:]
		if !ble {
			codec.PutUint16(buf, channel)
			buf = buf[2:]
		}
		buf[0] = 0x05
		codec.PutUint16(buf[1:], sequenceIdx)
		sequenceIdx++
		buf = buf[3:]

		blockSize = packetSize - 3 - extraHeaderSize
		copy(buf, command[offset:])
		offset += blockSize
	}

	return result
}

var (
	errTooShort        = errors.New("too short")
	errInvalidChannel  = errors.New("invalid channel")
	errInvalidSequence = errors.New("invalid sequence")
	errInvalidTag      = errors.New("invalid tag")
)

func validatePrefix(buf []byte, channel, sequenceIdx uint16, ble bool) ([]byte, error) {
	if !ble {
		if codec.Uint16(buf) != channel {
			return nil, errInvalidChannel
		}
		buf = buf[2:]
	}

	if buf[0] != 0x05 {
		return nil, errInvalidTag
	}
	if codec.Uint16(buf[1:]) != sequenceIdx {
		return nil, errInvalidSequence
	}
	return buf[3:], nil
}

// UnwrapResponseAPDU parses a response of 64 byte packets into the real data
func UnwrapResponseAPDU(channel uint16, dev <-chan []byte, packetSize int, ble bool) ([]byte, error) {
	var err error
	var sequenceIdx uint16
	var extraHeaderSize int
	if !ble {
		extraHeaderSize = 2
	}
	buf := <-dev
	if len(buf) < 5+extraHeaderSize+5 {
		return nil, errTooShort
	}

	buf, err = validatePrefix(buf, channel, sequenceIdx, ble)
	if err != nil {
		return nil, err
	}

	responseLength := int(codec.Uint16(buf))
	buf = buf[2:]
	result := make([]byte, responseLength)
	out := result

	blockSize := packetSize - 5 - extraHeaderSize
	if blockSize > len(buf) {
		blockSize = len(buf)
	}
	copy(out, buf[:blockSize])

	// if there is anything left to read...
	for len(out) > blockSize {
		out = out[blockSize:]
		buf = <-dev

		sequenceIdx++
		buf, err = validatePrefix(buf, channel, sequenceIdx, ble)
		if err != nil {
			return nil, err
		}

		blockSize = packetSize - 3 - extraHeaderSize
		if blockSize > len(buf) {
			blockSize = len(buf)
		}
		copy(out, buf[:blockSize])
	}
	return result, nil
}
