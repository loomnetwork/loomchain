package sample_go_contract

import (
	types "github.com/loomnetwork/go-loom/builtin/types/sample_go_contract"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
)

type SampleGoContract struct {
}

func (k *SampleGoContract) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "sample-go-contract",
		Version: "1.0.0",
	}, nil
}

func (k *SampleGoContract) Init(ctx contractpb.Context, req *types.SampleGoContractInitRequest) error {
	return nil
}

var Contract plugin.Contract = contractpb.MakePluginContract(&SampleGoContract{})
