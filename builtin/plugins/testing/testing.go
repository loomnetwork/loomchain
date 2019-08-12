package testing

import (
	types "github.com/loomnetwork/go-loom/builtin/types/testing"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
)

type Testing struct {
}

func (k *Testing) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "testing",
		Version: "1.0.0",
	}, nil
}

func (k *Testing) Init(ctx contractpb.Context, req *types.TestingInitRequest) error {
	return nil
}

var Contract plugin.Contract = contractpb.MakePluginContract(&Testing{})
