package node

import (
	"net"
	"sync"
)

type portGenerator struct {
	lock    sync.Mutex
	start   int
	current int
}

func (p *portGenerator) Next() int {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.current = GetPort()
	return p.current
}

// Token from https://github.com/phayes/freeport/blob/master/freeport.go
// GetFreePort asks the kernel for a free open port that is ready to use.
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
