// +build evm

package gateway

import (
	loom "github.com/loomnetwork/go-loom"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

type (
	ResetMainnetBlockRequest = tgtypes.TransferGatewayResetMainnetBlockRequest
)

type UnsafeGateway struct {
	Gateway
}

func (gw *UnsafeGateway) ResetMainnetBlock(ctx contract.Context, req *ResetMainnetBlockRequest) error {
	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	if loom.UnmarshalAddressPB(state.Owner).Compare(ctx.Message().Sender) != 0 {
		return ErrNotAuthorized
	}

	state.LastMainnetBlockNum = req.GetLastMainnetBlockNum()

	return saveState(ctx, state)
}
