// +build evm

package address_mapper

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	amtypes "github.com/loomnetwork/go-loom/builtin/types/address_mapper"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/features"
	ssha "github.com/miguelmota/go-solidity-sha3"
	"github.com/pkg/errors"
)

type (
	AddressMapping     = amtypes.AddressMapperMapping
	ListAddressMapping = amtypes.ListAddressMapperMapping

	InitRequest                  = amtypes.AddressMapperInitRequest
	AddIdentityMappingRequest    = amtypes.AddressMapperAddIdentityMappingRequest
	RemoveMappingRequest         = amtypes.AddressMapperRemoveMappingRequest
	GetMappingRequest            = amtypes.AddressMapperGetMappingRequest
	GetMappingResponse           = amtypes.AddressMapperGetMappingResponse
	GetMultiChainMappingRequest  = amtypes.AddressMapperGetMultichainMappingRequest
	GetMultiChainMappingResponse = amtypes.AddressMapperGetMultichainMappingResponse

	HasMappingRequest  = amtypes.AddressMapperHasMappingRequest
	HasMappingResponse = amtypes.AddressMapperHasMappingResponse

	ListMappingRequest  = amtypes.AddressMapperListMappingRequest
	ListMappingResponse = amtypes.AddressMapperListMappingResponse
)

var (
	// ErrrNotAuthorized indicates that a contract method failed because the caller didn't have
	// the permission to execute that method.
	ErrNotAuthorized = errors.New("[Address Mapper] not authorized")
	// ErrInvalidRequest is a generic error that's returned when something is wrong with the
	// request message, e.g. missing or invalid fields.
	ErrInvalidRequest = errors.New("[Address Mapper] invalid request")
	// ErrAlreadyRegistered indicates that from and/or to are already registered in
	// address mapper contract.
	ErrAlreadyRegistered = errors.New("[Address Mapper] identity mapping already exists")

	AddressPrefix = "addr"
)

func addressKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte(AddressPrefix), addr.Bytes())
}

func multiChainAddressKey(addr loom.Address, targetChainID string, sequence uint64) []byte {
	return util.PrefixKey([]byte(AddressPrefix), addr.Bytes(), []byte(targetChainID), UintToBytesBigEndian(sequence))
}

func getMultiChainAddressKey(addr loom.Address, targetChainID string) []byte {
	return util.PrefixKey([]byte(AddressPrefix), addr.Bytes(), []byte(targetChainID), []byte{})
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

	var isMultiChainEnable bool
	if ctx.FeatureEnabled(features.AddressMapperVersion1_2, false) {
		isMultiChainEnable = true
	}

	from := loom.UnmarshalAddressPB(req.From)
	to := loom.UnmarshalAddressPB(req.To)
	if from.ChainID == "" || to.ChainID == "" {
		return ErrInvalidRequest
	}

	if from.Compare(to) == 0 {
		return ErrInvalidRequest
	}

	allowedSigTypes := []evmcompat.SignatureType{
		evmcompat.SignatureType_EIP712,
		evmcompat.SignatureType_GETH,
		evmcompat.SignatureType_TREZOR,
		evmcompat.SignatureType_TRON,
	}

	if ctx.FeatureEnabled(features.AddressMapperVersion1_1, false) {
		allowedSigTypes = append(allowedSigTypes, evmcompat.SignatureType_BINANCE)
	}

	callerAddr := ctx.Message().Sender
	if callerAddr.Compare(from) == 0 {
		if err := verifySig(from, to, to.ChainID, req.Signature, allowedSigTypes); err != nil {
			return errors.Wrap(err, ErrNotAuthorized.Error())
		}
	} else if callerAddr.Compare(to) == 0 {
		if err := verifySig(from, to, from.ChainID, req.Signature, allowedSigTypes); err != nil {
			return errors.Wrap(err, ErrNotAuthorized.Error())
		}
	} else {
		return ErrInvalidRequest
	}

	var existingMapping AddressMapping
	if !isMultiChainEnable {
		if err := ctx.Get(addressKey(from), &existingMapping); err != contract.ErrNotFound {
			if err == nil {
				return ErrAlreadyRegistered
			}
			return err
		}

		if err := ctx.Get(addressKey(to), &existingMapping); err != contract.ErrNotFound {
			if err == nil {
				return ErrAlreadyRegistered
			}
			return err
		}
	}

	if !isMultiChainEnable {
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
	} else {

		err := ctx.Set(multiChainAddressKey(from, req.To.ChainId, 0), &AddressMapping{
			From: req.From,
			To:   req.To,
		})
		if err != nil {
			return err
		}

		err = ctx.Set(multiChainAddressKey(to, req.From.ChainId, 0), &AddressMapping{
			From: req.To,
			To:   req.From,
		})
		if err != nil {
			return err
		}
		// fmt.Printf("multichain key    %+v\n", multiChainAddressKey(from, req.To.ChainId, 0))
		// fmt.Printf("GET multichain key%+v\n", getMultiChainAddressKey(from, req.To.ChainId))
		// fmt.Printf("multichain key    %+v\n", multiChainAddressKey(to, req.From.ChainId, 0))
		// fmt.Printf("GET multichain key%+v\n", getMultiChainAddressKey(to, req.From.ChainId))

		// fmt.Println("IS SET 1", ctx.Has(multiChainAddressKey(from, req.To.ChainId, 0)))
		// fmt.Println("IS SET 2", ctx.Has(multiChainAddressKey(to, req.From.ChainId, 0)))
		// fmt.Println("IS SET 3", ctx.Has(getMultiChainAddressKey(from, req.To.ChainId)))
		// fmt.Println("IS SET 4", ctx.Has(getMultiChainAddressKey(to, req.From.ChainId)))
	}
	return nil
}

func (am *AddressMapper) RemoveMapping(ctx contract.StaticContext, req *RemoveMappingRequest) error {
	// TODO
	return nil
}

func (am *AddressMapper) ListMapping(ctx contract.StaticContext, req *ListMappingRequest) (*ListMappingResponse, error) {
	mappingRange := ctx.Range([]byte(AddressPrefix))
	listMappingResponse := ListMappingResponse{
		Mappings: []*AddressMapping{},
	}
	addressList := make(map[string]bool, len(mappingRange)/2)
	for _, m := range mappingRange {
		var mapping AddressMapping
		if err := proto.Unmarshal(m.Value, &mapping); err != nil {
			return &ListMappingResponse{}, errors.Wrap(err, "unmarshal mapping")
		}
		if addressList[mapping.From.String()] || addressList[mapping.To.String()] {
			continue
		}
		listMappingResponse.Mappings = append(listMappingResponse.Mappings, &AddressMapping{
			From: mapping.From,
			To:   mapping.To,
		})
		addressList[mapping.From.String()] = true
		addressList[mapping.To.String()] = true
	}
	return &listMappingResponse, nil
}

func (am *AddressMapper) HasMapping(ctx contract.StaticContext, req *HasMappingRequest) (*HasMappingResponse, error) {
	if req.From == nil {
		return nil, ErrInvalidRequest
	}

	var isMultiChainEnable bool
	if ctx.FeatureEnabled(features.AddressMapperVersion1_2, false) {
		isMultiChainEnable = true
	}
	hasResponse := HasMappingResponse{HasMapping: true}
	addr := loom.UnmarshalAddressPB(req.From)
	if !isMultiChainEnable {
		var mapping AddressMapping
		if err := ctx.Get(addressKey(addr), &mapping); err != nil {
			if err != contract.ErrNotFound {
				return nil, errors.Wrapf(err, "[Address Mapper] failed to load mapping for address: %v", addr)
			}
			hasResponse.HasMapping = false
		}
	} else {
		addr := loom.UnmarshalAddressPB(req.From)
		mappingRange := ctx.Range(addressKey(addr))
		if len(mappingRange) == 0 {
			hasResponse.HasMapping = false
		}
	}
	return &hasResponse, nil
}

func (am *AddressMapper) GetMapping(ctx contract.StaticContext, req *GetMappingRequest) (*GetMappingResponse, error) {
	if req.From == nil {
		return nil, ErrInvalidRequest
	}
	var isMultiChain bool
	if ctx.FeatureEnabled(features.AddressMapperVersion1_2, false) {
		isMultiChain = true
	}
	var mapping AddressMapping
	addr := loom.UnmarshalAddressPB(req.From)
	if !isMultiChain {
		if err := ctx.Get(addressKey(addr), &mapping); err != nil {
			return nil, errors.Wrapf(err, "[Address Mapper] failed to map address %v", addr)
		}
	} else {
		mappingRange := ctx.Range(addressKey(addr))
		for _, m := range mappingRange {
			if err := proto.Unmarshal(m.Value, &mapping); err != nil {
				return nil, errors.Wrap(err, "unmarshal mapping")
			}
		}
	}
	return &GetMappingResponse{
		From: mapping.From,
		To:   mapping.To,
	}, nil
}

func (am *AddressMapper) GetMultiChainMapping(ctx contract.StaticContext, req *GetMultiChainMappingRequest) (*GetMultiChainMappingResponse, error) {
	if req.From == nil {
		return nil, ErrInvalidRequest
	}

	if !ctx.FeatureEnabled(features.AddressMapperVersion1_2, false) {
		return nil, ErrNotAuthorized
	}
	addr := loom.UnmarshalAddressPB(req.From)
	var mappingRange = plugin.RangeData{}
	if req.ChainId == "" {
		mappingRange = ctx.Range(addressKey(addr))
	} else {
		mappingRange = ctx.Range(getMultiChainAddressKey(addr, req.ChainId))
	}

	getMultiChainResponse := GetMultiChainMappingResponse{
		Mappings: []*AddressMapping{},
	}
	for _, m := range mappingRange {
		var mapping AddressMapping
		if err := proto.Unmarshal(m.Value, &mapping); err != nil {
			return &GetMultiChainMappingResponse{}, errors.Wrap(err, "unmarshal mapping")
		}
		getMultiChainResponse.Mappings = append(getMultiChainResponse.Mappings, &AddressMapping{
			From: mapping.From,
			To:   mapping.To,
		})
	}
	return &getMultiChainResponse, nil
}

// func nonceKey(addr loom.Address) []byte {
// 	return util.PrefixKey([]byte("nonce"), addr.Bytes())
// }

// func Nonce(state loomchain.ReadOnlyState, addr loom.Address) uint64 {
// 	return
// 	(nonceKey(addr)).Value(state)
// }

func verifySig(from, to loom.Address, chainID string, sig []byte, allowedSigTypes []evmcompat.SignatureType) error {

	if (chainID != from.ChainID) && (chainID != to.ChainID) {
		return fmt.Errorf("chain ID %s doesn't match either address", chainID)
	}
	//nonce := 0 // from address nonce
	hash := ssha.SoliditySHA3(
		ssha.Address(common.BytesToAddress(from.Local)),
		ssha.Address(common.BytesToAddress(to.Local)),
		// ssha.Uint64(nonce),
		// ssha.String(to.ChainID),
	)

	sigType := evmcompat.SignatureType(sig[0])
	if sigType == evmcompat.SignatureType_BINANCE {
		hash = evmcompat.GenSHA256(
			ssha.Address(common.BytesToAddress(from.Local)),
			ssha.Address(common.BytesToAddress(to.Local)),
			// ssha.Uint64(nonce),
			// ssha.String(to.ChainID),
		)
	}

	signerAddr, err := evmcompat.RecoverAddressFromTypedSig(hash, sig, allowedSigTypes)
	if err != nil {
		return err
	}

	if (chainID == from.ChainID) && (bytes.Compare(signerAddr.Bytes(), from.Local) != 0) {
		return fmt.Errorf("signer address doesn't match, %s != %s", signerAddr.Hex(), from.Local.String())
	} else if (chainID == to.ChainID) && (bytes.Compare(signerAddr.Bytes(), to.Local) != 0) {
		return fmt.Errorf("signer address doesn't match, %s != %s", signerAddr.Hex(), to.Local.String())
	}
	return nil
}

func SignIdentityMapping(from, to loom.Address, key *ecdsa.PrivateKey, sigType evmcompat.SignatureType) ([]byte, error) {

	hash := ssha.SoliditySHA3(
		ssha.Address(common.BytesToAddress(from.Local)),
		ssha.Address(common.BytesToAddress(to.Local)),
		//ssha.Uint64(nonce),
		// ssha.String(to.ChainID),
	)

	if sigType == evmcompat.SignatureType_TRON {
		hash = evmcompat.PrefixHeader(hash, evmcompat.SignatureType_TRON)
	} else if sigType == evmcompat.SignatureType_BINANCE {
		hash = evmcompat.GenSHA256(
			ssha.Address(common.BytesToAddress(from.Local)),
			ssha.Address(common.BytesToAddress(to.Local)),
			//ssha.Uint64(nonce),
			// ssha.String(to.ChainID),
		)
	}

	return evmcompat.GenerateTypedSig(hash, key, sigType)
}

var Contract plugin.Contract = contract.MakePluginContract(&AddressMapper{})

func UintToBytesBigEndian(height uint64) []byte {
	heightB := make([]byte, 8)
	binary.BigEndian.PutUint64(heightB, height)
	return heightB
}
