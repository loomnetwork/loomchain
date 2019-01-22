// +build evm

package gateway

import (
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
	if ok, _ := ctx.HasPermission(resetMainnetBlockPerm, []string{oracleRole}); !ok {
		return ErrNotAuthorized
	}

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	state.LastMainnetBlockNum = req.GetLastMainnetBlockNum()

	return saveState(ctx, state)
}
