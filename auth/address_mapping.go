package auth

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"
)

//type chainType int32
const (
	Loom   = ""
	Eos    = "Eos"
	Eth    = "Eth"
	Tron   = "Tron"
	Cosmos = "Cosmos"
)

var (
	/*
		chainType_name = map[int32]string{
			0: Loom,
			1: Eos,  // ed25519
			2: Eth,  // seckp2561k, KECK256k1
			3: Tron,
			4: Cosmos,
		}
		chainType_value = map[string]int32{
			Loom:   0,
			Eos:    1,
			Eth:    2,
			Tron:   3,
			Cosmos: 4,
		}*/
	externalNetworks = map[string]ExternalNetworks{
		"": {
			Prefix:  "default",
			Type:    Loom,
			Network: "1",
			Enabled: true,
		},
		"Eth": {
			Prefix:  "eth",
			Type:    Eth,
			Network: "1",
			Enabled: true,
		},
		"Eos": {
			Prefix:  "eos",
			Type:    Eos,
			Network: "1",
			Enabled: false,
		},
		"Tron": {
			Prefix:  "tron",
			Type:    Tron,
			Network: "1",
			Enabled: false,
		},
		"Cosmos": {
			Prefix:  "cosmos",
			Type:    Cosmos,
			Network: "cosmos.xyz",
			Enabled: false,
		},
		"Binance": {
			Prefix:  "bnb",
			Type:    Cosmos,
			Network: "testnet.binance.com",
			Enabled: false,
		},
	}

	originVerification = map[string]originVerificationFunc{
		Loom:   verifyEd25519,
		Eth:    verifyKECK256k1,
		Eos:    verifyEd25519,
		Tron:   getTronOrigin,
		Cosmos: getCosmosOrigin,
	}
)

type originVerificationFunc func(tx SignedTx) ([]byte, error)

//func (x chainType) String() string {
//	return proto.EnumName(chainType_name, int32(x))
//}

type ExternalNetworks struct {
	Prefix  string
	Type    string
	Network string
	Enabled bool // Activate karma module
}

func GetSignatureTxMiddleware(
	createAddressMappingCtx func(state loomchain.State) (contractpb.Context, error),
) loomchain.TxMiddlewareFunc {
	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (res loomchain.TxHandlerResult, err error) {
		var r loomchain.TxHandlerResult

		var tx SignedTx
		err = proto.Unmarshal(txBytes, &tx)
		if err != nil {
			return r, err
		}

		chain, found := externalNetworks[tx.ChainName]
		if !found {
			return r, errors.Wrapf(err, "chain type %v not supported", tx.ChainName)
		}

		var localAddr []byte
		verifier, found := originVerification[chain.Type]

		if !found {
			return r, fmt.Errorf("invalid chain id %v", chain.Type)
		}

		localAddr, err = verifier(tx)
		if err != nil {
			return r, errors.Wrapf(err, "tx origin cannot be verifyed as type %v", chain.Type)
		}

		addressMappingCtx, err := createAddressMappingCtx(state)
		if err != nil {
			return res, errors.Wrap(err, "failed to create address-mapping contract context")
		}

		origin, err := GetMappedOrigin(localAddr, chain.Prefix, state.Block().ChainID, addressMappingCtx)
		if err != nil {
			return r, err
		}

		ctx := context.WithValue(state.Context(), ContextKeyOrigin, origin)
		return next(state.WithContext(ctx), tx.Inner, isCheckTx)
	})
}

func GetMappedOrigin(localAlias []byte, txChainPrefix, appChainId string, ctx contractpb.Context) (loom.Address, error) {
	if externalNetworks[Loom].Prefix == txChainPrefix {
		return loom.Address{
			ChainID: appChainId,
			Local:   localAlias,
		}, nil
	}

	am := address_mapper.AddressMapper{}
	resp, err := am.GetMapping(ctx, &address_mapper.GetMappingRequest{
		From: loom.Address{
			ChainID: txChainPrefix, // todo what should the chain id be here?
			Local:   localAlias,
		}.MarshalPB(),
	})
	if err != nil {
		return loom.Address{}, errors.Wrapf(err, "find mapped address for %v", string(localAlias))
	}

	origin := loom.UnmarshalAddressPB(resp.To)
	if origin.ChainID != appChainId {
		return loom.Address{}, fmt.Errorf("mapped address %v from wrong chain %v", origin.String(), appChainId)
	}

	return origin, nil
}

func getEosOrigin(tx SignedTx) ([]byte, error) {
	return verifyEd25519(tx)
}

func getEthOrigin(tx SignedTx) ([]byte, error) {
	return verifyKECK256k1(tx)
}

func getTronOrigin(tx SignedTx) ([]byte, error) {
	return loom.LocalAddressFromPublicKeyV2(tx.PublicKey), fmt.Errorf("not implemented")
}

func getCosmosOrigin(tx SignedTx) ([]byte, error) {
	return loom.LocalAddressFromPublicKeyV2(tx.PublicKey), fmt.Errorf("not implemented")
}

func verifyEd25519(tx SignedTx) ([]byte, error) {
	if len(tx.PublicKey) != ed25519.PublicKeySize {
		return nil, errors.New("invalid public key length")
	}

	if len(tx.Signature) != ed25519.SignatureSize {
		return nil, errors.New("invalid signature ed25519 signature size length")
	}

	if !ed25519.Verify(tx.PublicKey, tx.Inner, tx.Signature) {
		return nil, errors.New("invalid signature ed25519 verify")
	}

	return loom.LocalAddressFromPublicKey(tx.PublicKey), nil
}

func verifyKECK256k1(tx SignedTx) ([]byte, error) {
	if len(tx.Signature) != 65 {
		return loom.LocalAddress{}, fmt.Errorf("signature must be 65 bytes, got %d bytes", len(tx.Signature))
	}
	stdSig := make([]byte, 65)
	copy(stdSig[:], tx.Signature[:])
	stdSig[len(tx.Signature)-1] -= 27

	var signer common.Address
	pubKey, err := crypto.Ecrecover(tx.PublicKey, stdSig)
	if err != nil {
		return pubKey, err
	}

	copy(signer[:], crypto.Keccak256(pubKey[1:])[12:])
	return pubKey, nil
}
