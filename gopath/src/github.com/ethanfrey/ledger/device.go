package ledger

import (
	"errors"

	"github.com/ethanfrey/hid"
)

const (
	VendorLedger = 0x2c97
	ProductNano  = 1
	Channel      = 0x0101
	PacketSize   = 64
)

type Ledger struct {
	device Device
}

func NewLedger(dev Device) *Ledger {
	return &Ledger{
		device: dev,
	}
}

func FindLedger() (*Ledger, error) {
	devs, err := hid.Devices()
	if err != nil {
		return nil, err
	}
	for _, d := range devs {
		// TODO: ProductId filter
		if d.VendorID == VendorLedger {
			ledger, err := d.Open()
			if err != nil {
				return nil, err
			}
			return NewLedger(ledger), nil
		}
	}
	return nil, errors.New("no ledger connected")
}

// A Device provides access to a HID device.
type Device interface {
	// Close closes the device and associated resources.
	Close()

	// Write writes an output report to device. The first byte must be the
	// report number to write, zero if the device does not use numbered reports.
	Write([]byte) error

	// ReadCh returns a channel that will be sent input reports from the device.
	// If the device uses numbered reports, the first byte will be the report
	// number.
	ReadCh() <-chan []byte

	// ReadError returns the read error, if any after the channel returned from
	// ReadCh has been closed.
	ReadError() error
}
