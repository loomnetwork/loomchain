package auth

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"

	"github.com/loomnetwork/loomchain/auth/keys"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/features"
	appstate "github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/txhandler"
)

var originRecoveryFuncs = map[keys.SignedTxType]originRecoveryFunc{
	keys.LoomSignedTxType:     verifyEd25519,
	keys.EthereumSignedTxType: verifySolidity66Byte,
	keys.TronSignedTxType:     verifyTron,
	keys.BinanceSignedTxType:  verifyBinance,
}

type originRecoveryFunc func(tx SignedTx, allowedSigTypes []evmcompat.SignatureType) ([]byte, error)

// NewMultiChainSignatureTxMiddleware returns tx signing middleware that supports a set of chain
// specific signing algos.
func NewMultiChainSignatureTxMiddleware(
	chains map[string]keys.ChainConfig,
	createAddressMapperCtx func(state appstate.State) (contractpb.StaticContext, error),
) txhandler.TxMiddlewareFunc {
	return txhandler.TxMiddlewareFunc(func(
		state appstate.State,
		txBytes []byte,
		next txhandler.TxHandlerFunc,
		isCheckTx bool,
	) (txhandler.TxHandlerResult, error) {
		var r txhandler.TxHandlerResult

		var signedTx SignedTx
		if err := proto.Unmarshal(txBytes, &signedTx); err != nil {
			return r, err
		}

		var nonceTx NonceTx
		if err := proto.Unmarshal(signedTx.Inner, &nonceTx); err != nil {
			return r, errors.Wrap(err, "failed to unmarshal NonceTx")
		}

		var tx types.Transaction
		if err := proto.Unmarshal(nonceTx.Inner, &tx); err != nil {
			return r, errors.Wrap(err, "failed to unmarshal Transaction")
		}

		var msg vm.MessageTx
		if err := proto.Unmarshal(tx.Data, &msg); err != nil {
			return r, errors.Wrap(err, "failed to unmarshal MessageTx")
		}

		if msg.From == nil {
			return r, errors.New("malformed MessageTx, sender not specified")
		}

		msgSender := loom.UnmarshalAddressPB(msg.From)

		chain, found := chains[msgSender.ChainID]
		if !found {
			return r, fmt.Errorf("unknown chain ID %s", msgSender.ChainID)
		}

		recoverOrigin, found := originRecoveryFuncs[chain.TxType]
		if !found {
			return r, fmt.Errorf("recovery function for Tx type %v not found", chain.TxType)
		}

		recoveredAddr, err := recoverOrigin(signedTx, getAllowedSignatureTypes(state, msgSender.ChainID))
		if err != nil {
			return r, errors.Wrapf(err, "failed to recover origin (tx type %v, chain ID %s)",
				chain.TxType, msgSender.ChainID,
			)
		}

		if !bytes.Equal(recoveredAddr, msgSender.Local) {
			return r, fmt.Errorf("message sender %s doesn't match origin %s",
				hex.EncodeToString(msgSender.Local), hex.EncodeToString(recoveredAddr),
			)
		}

		switch chain.AccountType {
		case keys.NativeAccountType: // pass through origin & message sender as is
			ctx := context.WithValue(state.Context(), keys.ContextKeyOrigin, msgSender)
			return next(state.WithContext(ctx), signedTx.Inner, isCheckTx)

		case keys.MappedAccountType: // map origin & message sender to an address on this chain
			origin, err := getMappedAccountAddress(state, msgSender, createAddressMapperCtx)
			if err != nil {
				return r, err
			}

			msg.From = origin.MarshalPB()
			msgTxBytes, err := proto.Marshal(&msg)
			if err != nil {
				return r, errors.Wrap(err, "failed to marshal MessageTx")
			}

			tx.Data = msgTxBytes
			txBytes, err := proto.Marshal(&tx)
			if err != nil {
				return r, errors.Wrap(err, "failed to marshal Transaction")
			}

			nonceTx.Inner = txBytes
			nonceTxBytes, err := proto.Marshal(&nonceTx)
			if err != nil {
				return r, errors.Wrap(err, "failed to marshal NonceTx")
			}

			ctx := context.WithValue(state.Context(), keys.ContextKeyOrigin, origin)
			return next(state.WithContext(ctx), nonceTxBytes, isCheckTx)

		default:
			return r, fmt.Errorf("Invalid account type %v for chain ID %s", chain.AccountType, msgSender.ChainID)
		}
	})
}

func getMappedAccountAddress(
	state appstate.State,
	addr loom.Address,
	createAddressMapperCtx func(state appstate.State) (contractpb.StaticContext, error),
) (loom.Address, error) {
	ctx, err := createAddressMapperCtx(state)
	if err != nil {
		return loom.Address{}, errors.Wrap(err, "failed to create Address Mapper context")
	}

	am := &address_mapper.AddressMapper{}

	resp, err := am.GetMapping(ctx, &address_mapper.GetMappingRequest{
		From: addr.MarshalPB(),
	})
	if err != nil {
		return loom.Address{}, errors.Wrapf(err, "failed to map account %s", addr.String())
	}

	mappedAddr := loom.UnmarshalAddressPB(resp.To)
	if mappedAddr.ChainID != state.Block().ChainID {
		return loom.Address{}, fmt.Errorf("mapped account %s has wrong chain ID", addr.String())
	}

	return mappedAddr, nil
}

func verifyEd25519(tx SignedTx, _ []evmcompat.SignatureType) ([]byte, error) {
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

func getAllowedSignatureTypes(state appstate.State, chainID string) []evmcompat.SignatureType {
	if !state.FeatureEnabled(features.MultiChainSigTxMiddlewareVersion1_1, false) {
		return []evmcompat.SignatureType{
			evmcompat.SignatureType_EIP712,
			evmcompat.SignatureType_GETH,
			evmcompat.SignatureType_TREZOR,
			evmcompat.SignatureType_TRON,
		}
	}

	// TODO: chain <-> sig type associations should be in the loom.yml, not hard-coded here
	switch chainID {
	case "tron":
		if state.FeatureEnabled(features.AuthSigTxFeaturePrefix+"tron", false) {
			return []evmcompat.SignatureType{evmcompat.SignatureType_TRON}
		}
	case "binance":
		if state.FeatureEnabled(features.AuthSigTxFeaturePrefix+"binance", false) {
			return []evmcompat.SignatureType{evmcompat.SignatureType_BINANCE}
		}
	default:
		return []evmcompat.SignatureType{
			evmcompat.SignatureType_EIP712,
			evmcompat.SignatureType_GETH,
			evmcompat.SignatureType_TREZOR,
		}
	}

	return nil
}
