package dpos

import (
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv2"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv3"
)

type DPOSHandler struct {
	dposContractCtx contractpb.Context
	state           loomchain.State
}

func NewDPOSHandler(dposContractCtx contractpb.Context, state loomchain.State) loomchain.DPOSHandler {
	return &DPOSHandler{
		dposContractCtx: dposContractCtx,
		state:           state,
	}
}

func (dpos *DPOSHandler) ValidatorsList() ([]*types.Validator, error) {
	if dpos.state.FeatureEnabled(loomchain.DPOSVersion3Feature, false) {
		return dposv3.ValidatorList(dpos.dposContractCtx)
	}
	return dposv2.ValidatorList(dpos.dposContractCtx)
}
