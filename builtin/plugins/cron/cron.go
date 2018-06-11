package cron

import (
	ctypes "github.com/loomnetwork/go-loom/builtin/types/cron"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

type (
	InitCronRequest  = ctypes.InitCronRequest
	AddCronResponse  = ctypes.AddCronResponse
	AddCronRequest   = ctypes.AddCronRequest
	ListCronResponse = ctypes.ListCronResponse
	ListCronRequest  = ctypes.ListCronRequest
)

var (
	cronKey  = []byte("cron")
	decimals = 18
)

type Cron struct {
}

func (c *Cron) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "cron",
		Version: "1.0.0",
	}, nil
}

func (c *Cron) Init(ctx contract.Context, req *InitCronRequest) error {
	return nil
}

func (c *Cron) AddCronJob(
	ctx contract.StaticContext,
	req *AddCronRequest,
) (*AddCronResponse, error) {
	return &AddCronResponse{
		Error: "",
	}, nil
}

func (c *Cron) ListCronJobs(
	ctx contract.StaticContext,
	req *ListCronRequest,
) (*ListCronResponse, error) {
	return &ListCronResponse{}, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&Cron{})
