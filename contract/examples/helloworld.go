// go build -buildmode=plugin -o contracts/helloworld.so contract/examples/helloworld.go
package main

import (
	"github.com/hashicorp/go-version"
	"github.com/loomnetwork/loom/contract"
)

type HelloWorld struct {
}

func (c *HelloWorld) Meta() contract.PluginMeta {
	return contract.PluginMeta{
		Name:    "helloworld",
		Version: version.Must(version.NewVersion("1.0.0")),
	}
}

func (c *HelloWorld) Call(ctx contract.Context, input []byte) ([]byte, error) {
	return []byte("helloworld"), nil
}

func (c *HelloWorld) StaticCall(ctx contract.StaticContext, input []byte) ([]byte, error) {
	return nil, nil
}

var Contract contract.PluginContract = &HelloWorld{}
