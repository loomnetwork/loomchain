// +build evm

package gateway

import (
	"github.com/gogo/protobuf/proto"
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

// AddAuthorizedContractMapping allows overriding the existing contract mapping
func (gw *UnsafeGateway) UnsafeAddAuthorizedContractMapping(ctx contract.Context, req *AddContractMappingRequest) error {
	if req.ForeignContract == nil || req.LocalContract == nil {
		return ErrInvalidRequest
	}
	foreignAddr := loom.UnmarshalAddressPB(req.ForeignContract)
	localAddr := loom.UnmarshalAddressPB(req.LocalContract)
	if foreignAddr.ChainID == "" || localAddr.ChainID == "" {
		return ErrInvalidRequest
	}
	if foreignAddr.Compare(localAddr) == 0 {
		return ErrInvalidRequest
	}

	state, err := loadState(ctx)

	if err != nil {
		return err
	}

	callerAddr := ctx.Message().Sender

	// Only the Gateway owner is allowed to bypass contract ownership checks
	if callerAddr.Compare(loom.UnmarshalAddressPB(state.Owner)) != 0 {
		return ErrNotAuthorized
	}

	err = ctx.Set(contractAddrMappingKey(foreignAddr), &ContractAddressMapping{
		From: req.ForeignContract,
		To:   req.LocalContract,
	})
	if err != nil {
		return err
	}

	err = ctx.Set(contractAddrMappingKey(localAddr), &ContractAddressMapping{
		From: req.LocalContract,
		To:   req.ForeignContract,
	})
	if err != nil {
		return err
	}

	payload, err := proto.Marshal(&ContractMappingConfirmed{
		ForeignContract: req.ForeignContract,
		LocalContract:   req.LocalContract,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(payload, contractMappingConfirmedEventTopic)
	return nil
}
