package auth

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
)

type SignedTxType string

const (
	LoomSignedTxType     SignedTxType = "loom"
	EthereumSignedTxType SignedTxType = "eth"
	TronSignedTxType     SignedTxType = "tron"
	EosSignedTxType      SignedTxType = "eos"
)

// AccountType is used to specify which address should be used on-chain to identify a tx sender.
type AccountType int

const (
	// NativeAccountType indicates that the tx sender address should be passed through to contracts
	// as is, with the original chain ID intact.
	NativeAccountType AccountType = iota
	// MappedAccountType indicates that the tx sender address should be mapped to a DAppChain
	// address before being passed through to contracts.
	MappedAccountType
)

var originRecoveryFuncs = map[SignedTxType]originRecoveryFunc{
	LoomSignedTxType:     verifyEd25519,
	EthereumSignedTxType: verifySolidity66Byte,
	TronSignedTxType:     verifyTron,
	EosSignedTxType:      verifyEos,
}

type originRecoveryFunc func(tx SignedTx) ([]byte, error)

// NewMultiChainSignatureTxMiddleware returns tx signing middleware that supports a set of chain
// specific signing algos.
func NewMultiChainSignatureTxMiddleware(
	chains map[string]ChainConfig,
	createAddressMapperCtx func(state loomchain.State) (contractpb.Context, error),
) loomchain.TxMiddlewareFunc {
	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (loomchain.TxHandlerResult, error) {
		var r loomchain.TxHandlerResult

		var tx SignedTx
		if err := proto.Unmarshal(txBytes, &tx); err != nil {
			return r, err
		}

		msgSender, err := getMessageTxSender(tx.Inner)
		if err != nil {
			return r, errors.Wrap(err, "failed to extract message sender")
		}

		chain, found := chains[msgSender.ChainID]
		if !found {
			return r, fmt.Errorf("unknown chain ID %s", msgSender.ChainID)
		}

		recoverOrigin, found := originRecoveryFuncs[chain.TxType]
		if !found {
			return r, fmt.Errorf("recovery function for Tx type %v not found", chain.TxType)
		}

		recoveredAddr, err := recoverOrigin(tx)
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

		origin := msgSender

		if chain.AccountType == MappedAccountType {
			addressMapperCtx, lerr := createAddressMapperCtx(state)
			if lerr != nil {
				return r, errors.Wrap(lerr, "failed to create address-mapping contract context")
			}
			var err error
			origin, err = getMappedOrigin(addressMapperCtx, msgSender, state.Block().ChainID)
			if err != nil {
				return r, err
			}
		}

		ctx := context.WithValue(state.Context(), ContextKeyOrigin, origin)
		return next(state.WithContext(ctx), tx.Inner, isCheckTx)
	})
}

func GetActiveAddress(
	state loomchain.State,
	addr loom.Address,
	createAddressMapperCtx func(state loomchain.State) (contractpb.Context, error),
) (loom.Address, error) {
	ctx, err := createAddressMapperCtx(state)
	if err != nil {
		return loom.Address{}, err
	}
	return getMappedOrigin(ctx, addr, state.Block().ChainID)
}

func getMappedOrigin(
	ctx contractpb.StaticContext, origin loom.Address, toChainID string,
) (loom.Address, error) {
	am := &address_mapper.AddressMapper{}

	resp, err := am.GetMapping(ctx, &address_mapper.GetMappingRequest{
		From: origin.MarshalPB(),
	})
	if err != nil {
		return loom.Address{}, errors.Wrapf(err, "failed to map origin %s", origin.String())
	}

	mappedOrigin := loom.UnmarshalAddressPB(resp.To)
	if mappedOrigin.ChainID != toChainID {
		return loom.Address{}, fmt.Errorf("mapped origin %s has wrong chain ID", origin.String())
	}

	return mappedOrigin, nil
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

func getMessageTxSender(nonceTxBytes []byte) (loom.Address, error) {
	var nonceTx NonceTx
	if err := proto.Unmarshal(nonceTxBytes, &nonceTx); err != nil {
		return loom.Address{}, errors.Wrap(err, "failed to unmarshal NonceTx")
	}

	var tx types.Transaction
	if err := proto.Unmarshal(nonceTx.Inner, &tx); err != nil {
		return loom.Address{}, errors.Wrap(err, "failed to unmarshal Transaction")
	}

	var msg vm.MessageTx
	if err := proto.Unmarshal(tx.Data, &msg); err != nil {
		return loom.Address{}, errors.Wrap(err, "failed to unmarshal MessageTx")
	}

	if msg.From == nil {
		return loom.Address{}, errors.New("malformed MessageTx, sender not specified")
	}

	return loom.UnmarshalAddressPB(msg.From), nil
}

