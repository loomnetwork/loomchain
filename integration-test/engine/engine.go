package engine

import (
	"context"
	"sync"

	"github.com/loomnetwork/loomchain/integration-test/lib"
	"github.com/loomnetwork/loomchain/integration-test/node"
)

type Engine interface {
	Run(ctx context.Context) error
}

type engine struct {
	conf lib.Config
	wg   *sync.WaitGroup
	errC chan error
}

func New(conf lib.Config) Engine {
	return &engine{
		conf: conf,
		wg:   &sync.WaitGroup{},
		errC: make(chan error),
	}
}

func (e *engine) Run(ctx context.Context) error {
	for _, n := range e.conf.Nodes {
		go func(n *node.Node) {
			select {
			case e.errC <- n.Run(ctx):
			}
		}(n)
	}

	select {
	case err := <-e.errC:
		if err != nil {
			return err
		}
	}
	return nil
}
