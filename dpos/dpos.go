package dpos

import (
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv2"
)

type DPOSHandler struct {
	dposContractCtx contractpb.Context
}

func NewDPOSHandler(dposContractCtx contractpb.Context) loomchain.DPOSHandler {
	return &DPOSHandler{
		dposContractCtx: dposContractCtx,
	}
}

func (dpos *DPOSHandler) ValidatorsList() ([]*types.Validator, error) {
	return dposv2.ValidatorList(dpos.dposContractCtx)
}
