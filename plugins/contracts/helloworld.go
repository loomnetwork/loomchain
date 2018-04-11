//go build -buildmode=plugin -o out/helloworld.so plugins/contracts/helloworld.go
package main

import (
	"fmt"

	"github.com/loomnetwork/loom"
)

type HelloWorld struct {
}

func (k *HelloWorld) Name() string {
	return "HelloWorld"
}

func (k *HelloWorld) Init() error {
	fmt.Printf("Init contract \n")
	return nil
}

func (k *HelloWorld) Routes() []int {
	return []int{4, 5, 6} //TODO attach a function
}

func (k *HelloWorld) HandleRoutes(state loom.State, path string, data []byte) ([]byte, error) {
	if path == "5" {
		return k.Handle(state, path, data)
	}
	return nil, nil
}
func (q *HelloWorld) Handle(state loom.State, path string, data []byte) ([]byte, error) {
	return []byte("helloworld"), nil
}

var Contract HelloWorld
