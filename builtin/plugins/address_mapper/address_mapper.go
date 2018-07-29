// +build evm

package address_mapper

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	loom "github.com/loomnetwork/go-loom"
	amtypes "github.com/loomnetwork/go-loom/builtin/types/address_mapper"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/util"
	ssha "github.com/miguelmota/go-solidity-sha3"
	"github.com/pkg/errors"
)

type (
	AddressMapping = amtypes.AddressMapperMapping

	InitRequest               = amtypes.AddressMapperInitRequest
	AddIdentityMappingRequest = amtypes.AddressMapperAddIdentityMappingRequest
	AddContractMappingRequest = amtypes.AddressMapperAddContractMappingRequest
	RemoveMappingRequest      = amtypes.AddressMapperRemoveMappingRequest
	GetMappingRequest         = amtypes.AddressMapperGetMappingRequest
	GetMappingResponse        = amtypes.AddressMapperGetMappingResponse
)

const (
	SignatureType_EIP712 uint8 = 0
	SignatureType_GETH   uint8 = 1
	SignatureType_TREZOR uint8 = 2
)

var (
	// ErrrNotAuthorized indicates that a contract method failed because the caller didn't have
	// the permission to execute that method.
	ErrNotAuthorized = errors.New("[Address Mapper] not authorized")
	// ErrInvalidRequest is a generic error that's returned when something is wrong with the
	// request message, e.g. missing or invalid fields.
	ErrInvalidRequest = errors.New("[Address Mapper] invalid request")
)

func addressKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte("addr"), addr.Bytes())
}

type AddressMapper struct {
}

func (am *AddressMapper) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "addressmapper",
		Version: "0.1.0",
	}, nil
}

func (am *AddressMapper) Init(ctx contract.Context, req *InitRequest) error {
	return nil
}

// AddIdentityMapping adds a mapping between a DAppChain account and a Mainnet account.
// The caller must provide proof of ownership of the Mainnet account.
func (am *AddressMapper) AddIdentityMapping(ctx contract.Context, req *AddIdentityMappingRequest) error {
	if req.From == nil || req.To == nil || req.Signature == nil {
		return ErrInvalidRequest
	}
	from := loom.UnmarshalAddressPB(req.From)
	to := loom.UnmarshalAddressPB(req.To)
	if from.ChainID == "" || to.ChainID == "" {
		return ErrInvalidRequest
	}
	if from.Compare(to) == 0 {
		return ErrInvalidRequest
	}

	callerAddr := ctx.Message().Sender
	if callerAddr.Compare(from) == 0 {
		if err := verifySig(from, to, to.ChainID, req.Signature); err != nil {
			return errors.Wrap(err, ErrNotAuthorized.Error())
		}
	} else if callerAddr.Compare(to) == 0 {
		if err := verifySig(from, to, from.ChainID, req.Signature); err != nil {
			return errors.Wrap(err, ErrNotAuthorized.Error())
		}
	} else {
		return ErrInvalidRequest
	}

	err := ctx.Set(addressKey(from), &AddressMapping{
		From: req.From,
		To:   req.To,
	})
	if err != nil {
		return err
	}
	err = ctx.Set(addressKey(to), &AddressMapping{
		From: req.To,
		To:   req.From,
	})
	if err != nil {
		return err
	}
	return nil
}

// AddContractMapping adds a mapping between a DAppChain contract and a Mainnet contract.
// TODO: either we only let validators add contract mappings by consensus, or we require a signed
//       hash from the owner and verification by the oracles (and eventually a consesus of oracles).
func (am *AddressMapper) AddContractMapping(ctx contract.Context, req *AddContractMappingRequest) error {
	if req.From == nil || req.To == nil {
		return ErrInvalidRequest
	}
	from := loom.UnmarshalAddressPB(req.From)
	to := loom.UnmarshalAddressPB(req.To)
	if from.ChainID == "" || to.ChainID == "" {
		return ErrInvalidRequest
	}
	if from.Compare(to) == 0 {
		return ErrInvalidRequest
	}

	err := ctx.Set(addressKey(from), &AddressMapping{
		From: req.From,
		To:   req.To,
	})
	if err != nil {
		return err
	}
	err = ctx.Set(addressKey(to), &AddressMapping{
		From: req.To,
		To:   req.From,
	})
	if err != nil {
		return err
	}
	return nil
}

func (am *AddressMapper) RemoveMapping(ctx contract.StaticContext, req *RemoveMappingRequest) error {
	// TODO
	return nil
}

func (am *AddressMapper) GetMapping(ctx contract.StaticContext, req *GetMappingRequest) (*GetMappingResponse, error) {
	if req.From == nil {
		return nil, ErrInvalidRequest
	}
	var mapping AddressMapping
	addr := loom.UnmarshalAddressPB(req.From)
	if err := ctx.Get(addressKey(addr), &mapping); err != nil {
		return nil, errors.Wrapf(err, "[Address Mapper] failed to map address %v", addr)
	}
	return &GetMappingResponse{
		From: mapping.From,
		To:   mapping.To,
	}, nil
}

func verifySig(from, to loom.Address, chainID string, sig []byte) error {
	if len(sig) != 66 {
		return fmt.Errorf("signature must be 66 bytes, not %d bytes", len(sig))
	}
	if chainID != "eth" {
		return fmt.Errorf("verification of addresses on chain '%s' not supported", chainID)
	}
	if (chainID != from.ChainID) && (chainID != to.ChainID) {
		return fmt.Errorf("chain ID %s doesn't match either address", chainID)
	}

	hash := ssha.SoliditySHA3(
		ssha.Address(common.BytesToAddress(from.Local)),
		ssha.Address(common.BytesToAddress(to.Local)),
	)

	switch sig[0] {
	case SignatureType_GETH:
		hash = ssha.SoliditySHA3(
			ssha.String("\x19Ethereum Signed Message:\n32"),
			ssha.Bytes32(hash),
		)
	case SignatureType_TREZOR:
		hash = ssha.SoliditySHA3(
			ssha.String("\x19Ethereum Signed Message:\n\x20"),
			ssha.Bytes32(hash),
		)
	}

	signerAddr, err := evmcompat.SolidityRecover(hash, sig[1:])
	if err != nil {
		return err
	}

	if (chainID == from.ChainID) && (bytes.Compare(signerAddr.Bytes(), from.Local) != 0) {
		return fmt.Errorf("signer address doesn't match, %s != %s", signerAddr.Hex(), from.Local.String())
	} else if (chainID == to.ChainID) && (bytes.Compare(signerAddr.Bytes(), to.Local) != 0) {
		return fmt.Errorf("signer address doesn't match, %s = %s", signerAddr.Hex(), to.Local.String())
	}
	return nil
}

func SignIdentityMapping(from, to loom.Address, key *ecdsa.PrivateKey) ([]byte, error) {
	hash := ssha.SoliditySHA3(
		ssha.Address(common.BytesToAddress(from.Local)),
		ssha.Address(common.BytesToAddress(to.Local)),
	)
	sig, err := evmcompat.SoliditySign(hash, key)
	if err != nil {
		return nil, err
	}
	// Prefix the sig with a single byte indicating the sig type, in this case EIP712
	return append(make([]byte, 1, 66), sig...), nil
}

var Contract plugin.Contract = contract.MakePluginContract(&AddressMapper{})
