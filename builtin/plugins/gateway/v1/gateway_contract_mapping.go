// +build evm

package gateway

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway/v1"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	ssha "github.com/miguelmota/go-solidity-sha3"
)

type (
	PendingContractMapping             = tgtypes.TransferGatewayPendingContractMapping
	ContractAddressMapping             = tgtypes.TransferGatewayContractAddressMapping
	UnverifiedContractCreator          = tgtypes.TransferGatewayUnverifiedContractCreator
	VerifiedContractCreator            = tgtypes.TransferGatewayVerifiedContractCreator
	ContractMappingConfirmed           = tgtypes.TransferGatewayContractMappingConfirmed
	AddContractMappingRequest          = tgtypes.TransferGatewayAddContractMappingRequest
	UnverifiedContractCreatorsRequest  = tgtypes.TransferGatewayUnverifiedContractCreatorsRequest
	UnverifiedContractCreatorsResponse = tgtypes.TransferGatewayUnverifiedContractCreatorsResponse
	VerifyContractCreatorsRequest      = tgtypes.TransferGatewayVerifyContractCreatorsRequest
)

// AddContractMapping adds a mapping between a DAppChain contract and a Mainnet contract.
func (gw *Gateway) AddContractMapping(ctx contract.Context, req *AddContractMappingRequest) error {
	if req.ForeignContract == nil || req.LocalContract == nil || req.ForeignContractCreatorSig == nil ||
		req.ForeignContractTxHash == nil {
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

	localRec, err := ctx.ContractRecord(localAddr)
	if err != nil {
		return err
	}

	callerAddr := ctx.Message().Sender
	if callerAddr.Compare(localRec.CreatorAddress) != 0 {
		return ErrNotAuthorized
	}

	if contractMappingExists(ctx, foreignAddr, localAddr) {
		return ErrContractMappingExists
	}

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	hash := ssha.SoliditySHA3(
		ssha.Address(common.BytesToAddress(req.ForeignContract.Local)),
		ssha.Address(common.BytesToAddress(req.LocalContract.Local)),
	)

	signerAddr, err := evmcompat.RecoverAddressFromTypedSig(hash, req.ForeignContractCreatorSig)
	if err != nil {
		return err
	}

	err = ctx.Set(pendingContractMappingKey(state.NextContractMappingID),
		&PendingContractMapping{
			ID:              state.NextContractMappingID,
			ForeignContract: req.ForeignContract,
			LocalContract:   req.LocalContract,
			ForeignContractCreator: loom.Address{
				ChainID: foreignAddr.ChainID,
				Local:   loom.LocalAddress(signerAddr.Bytes()),
			}.MarshalPB(),
			ForeignContractTxHash: req.ForeignContractTxHash,
		},
	)
	if err != nil {
		return err
	}

	state.NextContractMappingID++
	return ctx.Set(stateKey, state)
}

func (gw *Gateway) UnverifiedContractCreators(ctx contract.StaticContext,
	req *UnverifiedContractCreatorsRequest) (*UnverifiedContractCreatorsResponse, error) {
	var creators []*UnverifiedContractCreator
	for _, entry := range ctx.Range(pendingContractMappingKeyPrefix) {
		var mapping PendingContractMapping
		if err := proto.Unmarshal(entry.Value, &mapping); err != nil {
			return nil, err
		}
		creators = append(creators, &UnverifiedContractCreator{
			ContractMappingID: mapping.ID,
			ContractTxHash:    mapping.ForeignContractTxHash,
		})
	}
	return &UnverifiedContractCreatorsResponse{
		Creators: creators,
	}, nil
}

func (gw *Gateway) VerifyContractCreators(ctx contract.Context,
	req *VerifyContractCreatorsRequest) error {
	if len(req.Creators) == 0 {
		return ErrInvalidRequest
	}

	if ok, _ := ctx.HasPermission(verifyCreatorsPerm, []string{oracleRole}); !ok {
		return ErrNotAuthorized
	}

	for _, creatorInfo := range req.Creators {
		mappingKey := pendingContractMappingKey(creatorInfo.ContractMappingID)
		mapping := &PendingContractMapping{}
		if err := ctx.Get(mappingKey, mapping); err != nil {
			if err == contract.ErrNotFound {
				// A pending mapping is removed as soon as an oracle submits a confirmation,
				// so it won't be unusual for it to be missing when multiple oracles are running.
				continue
			}
			return err
		}

		if err := confirmContractMapping(ctx, mappingKey, mapping, creatorInfo); err != nil {
			return err
		}
	}

	return nil
}

func confirmContractMapping(ctx contract.Context, pendingMappingKey []byte, mapping *PendingContractMapping,
	confirmation *VerifiedContractCreator) error {
	// Clear out the pending mapping regardless of whether it's successfully confirmed or not
	ctx.Delete(pendingMappingKey)

	if (mapping.ForeignContractCreator.ChainId != confirmation.Creator.ChainId) ||
		(mapping.ForeignContractCreator.Local.Compare(confirmation.Creator.Local) != 0) ||
		(mapping.ForeignContract.ChainId != confirmation.Contract.ChainId) ||
		(mapping.ForeignContract.Local.Compare(confirmation.Contract.Local) != 0) {
		ctx.Logger().Debug("[Transfer Gateway] failed to verify foreign contract creator",
			"expected-contract", mapping.ForeignContractCreator.Local,
			"expected-creator", mapping.ForeignContractCreator.Local,
			"actual-contract", confirmation.Contract.Local,
			"actual-creator", confirmation.Creator.Local,
		)
		return nil
	}

	foreignContractAddr := loom.UnmarshalAddressPB(mapping.ForeignContract)
	localContractAddr := loom.UnmarshalAddressPB(mapping.LocalContract)
	err := ctx.Set(contractAddrMappingKey(foreignContractAddr), &ContractAddressMapping{
		From: mapping.ForeignContract,
		To:   mapping.LocalContract,
	})
	if err != nil {
		return err
	}
	err = ctx.Set(contractAddrMappingKey(localContractAddr), &ContractAddressMapping{
		From: mapping.LocalContract,
		To:   mapping.ForeignContract,
	})
	if err != nil {
		return err
	}

	payload, err := proto.Marshal(&ContractMappingConfirmed{
		ForeignContract: mapping.ForeignContract,
		LocalContract:   mapping.LocalContract,
	})
	if err != nil {
		return err
	}
	ctx.EmitTopics(payload, contractMappingConfirmedEventTopic)
	return nil
}

// Returns the address of the DAppChain contract that corresponds to the given Ethereum address
func resolveToLocalContractAddr(ctx contract.StaticContext, foreignContractAddr loom.Address) (loom.Address, error) {
	var mapping ContractAddressMapping
	if err := ctx.Get(contractAddrMappingKey(foreignContractAddr), &mapping); err != nil {
		return loom.Address{}, err
	}
	return loom.UnmarshalAddressPB(mapping.To), nil
}

// Returns the address of the Ethereum contract that corresponds to the given DAppChain address
func resolveToForeignContractAddr(ctx contract.StaticContext, localContractAddr loom.Address) (loom.Address, error) {
	var mapping ContractAddressMapping
	if err := ctx.Get(contractAddrMappingKey(localContractAddr), &mapping); err != nil {
		return loom.Address{}, err
	}
	return loom.UnmarshalAddressPB(mapping.To), nil
}

// Checks if a pending or confirmed contract mapping referencing either of the given contracts exists
func contractMappingExists(ctx contract.StaticContext, foreignContractAddr, localContractAddr loom.Address) bool {
	var mapping ContractAddressMapping
	if err := ctx.Get(contractAddrMappingKey(foreignContractAddr), &mapping); err == nil {
		return true
	}
	if err := ctx.Get(contractAddrMappingKey(localContractAddr), &mapping); err == nil {
		return true
	}

	for _, entry := range ctx.Range(pendingContractMappingKeyPrefix) {
		var mapping PendingContractMapping
		if err := proto.Unmarshal(entry.Value, &mapping); err != nil {
			continue
		}
		if loom.UnmarshalAddressPB(mapping.ForeignContract).Compare(foreignContractAddr) == 0 {
			return true
		}
		if loom.UnmarshalAddressPB(mapping.LocalContract).Compare(localContractAddr) == 0 {
			return true
		}
	}

	return false
}
