// go build -buildmode=plugin -o contracts/helloworld.so contract/examples/helloworld.go
package main

import (
	"fmt"

	"github.com/loomnetwork/loom"
	"github.com/loomnetwork/loom/contract"
)

type HelloWorld struct {
}

var _ contract.Contract = &HelloWorld{}

func (k *HelloWorld) Init(params []byte) error {
	fmt.Printf("Init contract \n")
	return nil
}

func (k *HelloWorld) Call(state loom.State, method string, data []byte) ([]byte, error) {
	if method == "hello" {
		return k.handleHello(state, data)
	}
	return nil, nil
}

func (q *HelloWorld) handleHello(state loom.State, data []byte) ([]byte, error) {
	return []byte("helloworld"), nil
}

var Contract HelloWorld
