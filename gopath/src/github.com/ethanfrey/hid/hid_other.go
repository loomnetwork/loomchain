// +build !cgo !linux
// +build !cgo !windows
// +build !cgo !darwin

/*
hid_other provides an empty implementation for unsupported
platforms that compiles but returns errors when used.

In my use case, the software should function with reduced
functionality if hid is not available and the previous state
limited us from compiling on platforms such as FreeBSD.
Ideally, they are fully supported when possible.
*/

package hid

import "errors"

var errNotSupported = errors.New("Platform not supported")

// Devices always retuns an error on unsupported platforms
func Devices() ([]*DeviceInfo, error) {
	return nil, errNotSupported
}

// ByPath always retuns an error on unsupported platforms
func ByPath(path string) (*DeviceInfo, error) {
	return nil, errNotSupported
}

// Open always retuns an error on unsupported platforms
func (d *DeviceInfo) Open() (Device, error) {
	return nil, errNotSupported
}
