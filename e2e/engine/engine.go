package engine

import (
	"context"
	"sync"

	"github.com/loomnetwork/loomchain/e2e/lib"
	"github.com/loomnetwork/loomchain/e2e/node"
)

type Engine interface {
	Run(ctx context.Context, eventC chan *node.Event) error
}

type engine struct {
	conf   lib.Config
	wg     *sync.WaitGroup
	errC   chan error
	eventC chan *node.Event
}

func New(conf lib.Config) Engine {
	return &engine{
		conf:   conf,
		wg:     &sync.WaitGroup{},
		errC:   make(chan error),
		eventC: make(chan *node.Event),
	}
}

func (e *engine) Run(ctx context.Context, eventC chan *node.Event) error {
	for _, n := range e.conf.Nodes {
		go func(n *node.Node) {
			e.errC <- n.Run(ctx, eventC)
		}(n)
	}

	err := <-e.errC
	if err != nil {
		return err
	}
	return nil
}
