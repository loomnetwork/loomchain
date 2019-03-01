package auth

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"
)

const (
	EthChainId = "eth" // hard coded in address-mapper as only chainId supported by address-mapper

	Loom = ""
	Eth  = "eth"
)

var (
	externalNetworks = map[string]ExternalNetworks{
		"": {
			Prefix:  "default",
			Type:    Loom,
			Network: "1",
			Enabled: true,
		},
		EthChainId: {
			Prefix:  EthChainId,
			Type:    Eth,
			Network: "1",
			Enabled: true,
		},
	}

	originVerification = map[string]originVerificationFunc{
		Loom: verifyEd25519,
		Eth:  verifySolidity,
	}
)

type originVerificationFunc func(tx SignedTx) ([]byte, error)

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

		if !chain.Enabled {
			return r, errors.Wrapf(err, "chain type %v not enabled", tx.ChainName)
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

		origin, err := GetMappedOrigin(addressMappingCtx, localAddr, chain.Prefix, state.Block().ChainID)
		if err != nil {
			return r, err
		}

		ctx := context.WithValue(state.Context(), ContextKeyOrigin, origin)
		return next(state.WithContext(ctx), tx.Inner, isCheckTx)
	})
}

func GetMappedOrigin(ctx contractpb.Context, localAlias []byte, txChainPrefix, appChainId string) (loom.Address, error) {
	if _, ok := externalNetworks[Loom]; ok && externalNetworks[Loom].Prefix == txChainPrefix {
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

func verifySolidity(tx SignedTx) ([]byte, error) {
	ethAddr, err := evmcompat.SolidityRecover(tx.PublicKey, tx.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "verify solidity key")
	}
	return ethAddr.Bytes(), nil
}
