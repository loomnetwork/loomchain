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

	state.LastMainnetBlockNum = req.GetLastMainnetBlockNum()

	return saveState(ctx, state)
}

func (gw *UnsafeGateway) UnsafeAddOracle(ctx contract.Context, req *AddOracleRequest) error {
	oracleAddr := loom.UnmarshalAddressPB(req.Oracle)
	if ctx.Has(oracleStateKey(oracleAddr)) {
		return ErrOracleAlreadyRegistered
	}

	return addOracle(ctx, oracleAddr)
}

func (gw *UnsafeGateway) UnsafeRemoveOracle(ctx contract.Context, req *RemoveOracleRequest) error {
	oracleAddr := loom.UnmarshalAddressPB(req.Oracle)
	if !ctx.Has(oracleStateKey(oracleAddr)) {
		return ErrOracleNotRegistered
	}

	return removeOracle(ctx, oracleAddr)
}

func (gw *UnsafeGateway) ResetOwnerKey(ctx contract.Context, req *AddOracleRequest) error {
	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	// Revoke permissions from old owner
	oldOwnerAddr := loom.UnmarshalAddressPB(state.Owner)
	ctx.RevokePermissionFrom(oldOwnerAddr, changeOraclesPerm, ownerRole)

	// Update owner and grant permissions
	state.Owner = req.Oracle
	ownerAddr := loom.UnmarshalAddressPB(req.Oracle)
	ctx.GrantPermissionTo(ownerAddr, changeOraclesPerm, ownerRole)

	return saveState(ctx, state)
}
