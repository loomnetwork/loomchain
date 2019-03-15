package auth

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
)

type VerificationAlgorithm string
const (
	Loom    VerificationAlgorithm   = "loom"
	Eth     VerificationAlgorithm   = "eth"
	Tron    VerificationAlgorithm   = "tron"

	LoomName        = "loom"
	EthName         = "eth"
	TronName        = "tron"
)

var (
	originVerification = map[VerificationAlgorithm]originVerificationFunc{
		Loom: verifyEd25519,
		Eth:  verifySolidity66Byte,
		Tron: verifyTron,
	}
)

type originVerificationFunc func(tx SignedTx) ([]byte, error)

type ExternalNetworks struct {
	Prefix  string
	Type    VerificationAlgorithm
	Network string
	Enabled bool
}

func DefaultExternalNetworks(defaultChainId, ethChainId, tronChainId string) map[string]ExternalNetworks {
	externalNetworks := map[string]ExternalNetworks{
		LoomName: {
			Prefix:  defaultChainId,
			Type:    Loom,
			Network: "1",
			Enabled: true,
		},
	}
	if len(ethChainId) > 0 {
		externalNetworks[EthName] = ExternalNetworks{
			Prefix:  ethChainId,
			Type:    Eth,
			Network: "1",
			Enabled: true,
		}
	}
	if len(tronChainId) > 0 {
		externalNetworks[TronName] = ExternalNetworks{
			Prefix:  tronChainId,
			Type:    Tron,
			Network: "1",
			Enabled: true,
		}
	}
	return externalNetworks
}

func GetSignatureTxMiddleware(
	ethChainId, tronChainId string,
	externalNetworks map[string]ExternalNetworks,
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

		var origin loom.Address
		if len(tx.ChainName) == 0 {
			origin, err = chainIdVerification(tx, state.Block().ChainID, ethChainId, tronChainId)
		} else {
			addressMappingCtx, lerr := createAddressMappingCtx(state)
			if lerr != nil {
				return r, errors.Wrap(lerr, "failed to create address-mapping contract context")
			}
			origin, err = addressMappingVerification(addressMappingCtx, tx, externalNetworks, state.Block().ChainID)
		}
		if err != nil {
			return r, errors.Wrap(err, "verifying transaction signature")
		}

		ctx := context.WithValue(state.Context(), ContextKeyOrigin, origin)
		return next(state.WithContext(ctx), tx.Inner, isCheckTx)
	})
}

func GetActiveAddress(
	state loomchain.State,
	chainId string,
	local []byte,
	createAddressMappingCtx func(state loomchain.State) (contractpb.Context, error),
	externalNetworks map[string]ExternalNetworks,
) (loom.Address, error) {
	ctx, err := createAddressMappingCtx(state)
	if err != nil {
		return loom.Address{}, err
	}
	return GetMappedOrigin(ctx, local, chainId, state.Block().ChainID, externalNetworks)
}

func chainIdVerification(signedTx SignedTx, defaultChainId, ethChainId, tronChainId string,) (loom.Address, error) {
	caller,_, _, err := auth.GetFromToNonce(signedTx.Inner)
	if err != nil {
		return loom.Address{}, err
	}
	origin := loom.Address{ChainID: caller.ChainID}

	switch caller.ChainID {
	case "": return loom.Address{}, errors.Errorf("empty chain id")
	case defaultChainId: origin.Local, err = verifyEd25519(signedTx)
	case ethChainId: origin.Local, err = verifySolidity66Byte(signedTx)
	case tronChainId: origin.Local, err = verifyTron(signedTx)
	default:
		return loom.Address{}, errors.Wrapf(err, "unspported chain id %v", caller.ChainID)
	}

	if origin.Compare(caller) != 0 {
		return loom.Address{}, fmt.Errorf("Origin doesn't match caller: %v != %v", origin, caller)
	}

	return origin, nil
}

func addressMappingVerification(
	ctx contractpb.StaticContext,
	tx SignedTx,
	externalNetworks map[string]ExternalNetworks,
	appChainId string,
) (loom.Address, error) {
	chain, found := externalNetworks[tx.ChainName]
	if !found {
		return loom.Address{}, errors.Errorf("chain type %v not supported", tx.ChainName)
	}

	if !chain.Enabled {
		return loom.Address{}, errors.Errorf("chain type %v not enabled", tx.ChainName)
	}

	var localAddr []byte
	verifier, found := originVerification[chain.Type]

	if !found {
		return loom.Address{}, fmt.Errorf("invalid chain id %v", chain.Type)
	}

	localAddr, err := verifier(tx)
	if err != nil {
		return loom.Address{}, errors.Wrapf(err, "tx origin cannot be verified as type %v", chain.Type)
	}

	caller, _, _, err := auth.GetFromToNonce(tx.Inner)
	if err != nil {
		return loom.Address{}, err
	}
	if bytes.Compare(localAddr, caller.Local) != 0 {
		return loom.Address{}, fmt.Errorf("caller doesn't match verfied account %v != %v", hex.EncodeToString(localAddr), hex.EncodeToString(caller.Local))
	}

	origin, err := GetMappedOrigin(ctx, localAddr, chain.Prefix, appChainId, externalNetworks)
	if err != nil {
		return loom.Address{}, err
	}

	return origin, nil
}

func GetMappedOrigin(
	ctx contractpb.StaticContext,
	localAlias []byte,
	txChainPrefix,
	appChainId string,
	externalNetworks map[string]ExternalNetworks,
) (loom.Address, error) {
	if _, ok := externalNetworks[LoomName]; ok && externalNetworks[LoomName].Prefix == txChainPrefix {
		return loom.Address{
			ChainID: appChainId,
			Local:   localAlias,
		}, nil
	}

	am := address_mapper.AddressMapper{}
	resp, err := am.GetMapping(ctx, &address_mapper.GetMappingRequest{
		From: loom.Address{
			ChainID: txChainPrefix,
			Local:   localAlias,
		}.MarshalPB(),
	})
	if err != nil {
		return loom.Address{}, errors.Wrapf(err, "find mapped address for %v", hex.EncodeToString(localAlias))
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
