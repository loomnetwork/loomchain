// go build -buildmode=plugin -o contracts/helloworld.so plugin/examples/helloworld.go
package main

import (
	"github.com/loomnetwork/loom/plugin"
)

type HelloWorld struct {
}

func (c *HelloWorld) Meta() plugin.Meta {
	return plugin.Meta{
		Name:    "helloworld",
		Version: "1.0.0",
	}
}

func (c *HelloWorld) Call(ctx plugin.Context, input []byte) ([]byte, error) {
	return []byte("helloworld"), nil
}

func (c *HelloWorld) StaticCall(ctx plugin.StaticContext, input []byte) ([]byte, error) {
	return nil, nil
}

var Contract plugin.Contract = &HelloWorld{}
