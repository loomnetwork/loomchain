package auth

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"
)

const (
	EthChainId = "eth" // hard coded in address-mapper as only chainId supported by address-mapper
	DefaultLoomChainId = "default"
	Loom = "loom"
	Eth  = "eth"
)

var (
	originVerification = map[string]originVerificationFunc{
		Loom: verifyEd25519,
		Eth:  verifySolidity66Byte,
	}
)

type originVerificationFunc func(tx SignedTx) ([]byte, error)

type ExternalNetworks struct {
	Prefix  string
	Type    string
	Network string
	Enabled bool
}

func DefaultExternalNetworks() map[string]ExternalNetworks {
	return map[string]ExternalNetworks{
		Loom: {
			Prefix:  DefaultLoomChainId,
			Type:    Loom,
			Network: "1",
			Enabled: true,
		},
		Eth: {
			Prefix:  EthChainId,
			Type:    Eth,
			Network: "1",
			Enabled: true,
		},
	}
}

func GetSignatureTxMiddleware(
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
			origin, err = chainIdVerication(tx)
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

func chainIdVerication(signedTx SignedTx) (loom.Address, error) {
	var nonceTx auth.NonceTx
	if err := proto.Unmarshal(signedTx.Inner, &nonceTx); err != nil {
		return loom.Address{}, errors.Wrap(err, "unwrap nonce Tx")
	}

	var tx loomchain.Transaction
	if err := proto.Unmarshal(nonceTx.Inner, &tx); err != nil {
		return loom.Address{}, errors.New("unmarshal tx")
	}

	var msg vm.MessageTx
	if err := proto.Unmarshal(tx.Data, &msg); err != nil {
		return loom.Address{}, errors.Wrapf(err, "unmarshal message tx %v", tx.Data)
	}

	if msg.From == nil {
		return loom.Address{}, errors.Errorf("nil from address")
	}

	fromAddr := loom.UnmarshalAddressPB(msg.From)
	var err error
	switch fromAddr.ChainID  {
	case DefaultLoomChainId:
		fromAddr.Local, err = verifyEd25519(signedTx)
	case EthChainId:
		fromAddr.Local, err = verifySolidity66Byte(signedTx)
	default:
		return loom.Address{}, errors.Wrapf(err, "unspported chain id %v", fromAddr.ChainID)
	}

	return fromAddr, nil

}

func addressMappingVerification(
	ctx contractpb.Context,
	tx SignedTx,
	externalNetworks map[string]ExternalNetworks,
	appChainId string,
) (loom.Address, error) {
	chain, found := externalNetworks[tx.ChainName]
	if !found {
		return loom.Address{}, errors.Errorf( "chain type %v not supported", tx.ChainName)
	}

	if !chain.Enabled {
		return loom.Address{}, errors.Errorf( "chain type %v not enabled", tx.ChainName)
	}

	var localAddr []byte
	verifier, found := originVerification[chain.Type]

	if !found {
		return loom.Address{}, fmt.Errorf("invalid chain id %v", chain.Type)
	}

	localAddr, err := verifier(tx)
	if err != nil {
		return loom.Address{}, errors.Wrapf(err, "tx origin cannot be verifyed as type %v", chain.Type)
	}

	origin, err := GetMappedOrigin(ctx, localAddr, chain.Prefix, appChainId, externalNetworks)
	if err != nil {
		return loom.Address{}, err
	}
	return origin, nil
}

func GetMappedOrigin(
	ctx contractpb.Context,
	localAlias []byte,
	txChainPrefix,
	appChainId string,
	externalNetworks map[string]ExternalNetworks,
) (loom.Address, error) {
	if _, ok := externalNetworks[Loom]; ok && externalNetworks[Loom].Prefix == txChainPrefix {
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
