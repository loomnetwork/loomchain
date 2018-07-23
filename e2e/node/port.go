package node

import (
	"net"
	"sync"
)

type portGenerator struct {
	smap sync.Map
}

func (p *portGenerator) Next() int {
	// make sure that we don't reuse the port
	var port int
	for {
		port = GetPort()
		_, ok := p.smap.Load(&port)
		if !ok {
			p.smap.Store(&port, &port)
			break
		}
	}
	return port
}

// GetFreePort asks the kernel for a free open port that is ready to use.
// Token from https://github.com/phayes/freeport/blob/master/freeport.go
func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// GetPort is deprecated, use GetFreePort instead
// Ask the kernel for a free open port that is ready to use
func GetPort() int {
	port, err := GetFreePort()
	if err != nil {
		panic(err)
	}
	return port
}
