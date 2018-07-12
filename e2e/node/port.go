package node

import (
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
	p.current++
	return p.current % 65535
}
