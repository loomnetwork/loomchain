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

func (c *HelloWorld) Init(ctx plugin.Context, req *plugin.Request) (*plugin.Response, error) {
	return nil, nil
}

func (c *HelloWorld) Call(ctx plugin.Context, req *plugin.Request) (*plugin.Response, error) {
	return nil, nil
}

func (c *HelloWorld) StaticCall(ctx plugin.StaticContext, req *plugin.Request) (*plugin.Response, error) {
	return nil, nil
}

var Contract plugin.Contract = &HelloWorld{}
