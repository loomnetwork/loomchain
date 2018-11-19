package auth

import (
	"context"
	"errors"

	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/loomnetwork/loomchain/privval"

	"github.com/gogo/protobuf/proto"
	"golang.org/x/crypto/ed25519"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
)

type contextKey string

func (c contextKey) String() string {
	return "auth " + string(c)
}

var (
	ContextKeyOrigin = contextKey("origin")
)

func Origin(ctx context.Context) loom.Address {
	return ctx.Value(ContextKeyOrigin).(loom.Address)
}

var SignatureTxMiddleware = loomchain.TxMiddlewareFunc(func(
	state loomchain.State,
	txBytes []byte,
	next loomchain.TxHandlerFunc,
) (loomchain.TxHandlerResult, error) {
	var r loomchain.TxHandlerResult

	var tx SignedTx
	err := proto.Unmarshal(txBytes, &tx)
	if err != nil {
		return r, err
	}

	if privval.EnableSecp256k1 {
		if len(tx.PublicKey) != secp256k1.PubKeySecp256k1Size {
			return r, errors.New("invalid public key length")
		}

		secp256k1PubKey := secp256k1.PubKeySecp256k1{}
		copy(secp256k1PubKey[:], tx.PublicKey[:])
		secp256k1Signature := secp256k1.SignatureSecp256k1FromBytes(tx.Signature)
		if !secp256k1PubKey.VerifyBytes(tx.Inner, secp256k1Signature) {
			return r, errors.New("invalid signature")
		}
	} else {
		if len(tx.PublicKey) != ed25519.PublicKeySize {
			return r, errors.New("invalid public key length")
		}

		if len(tx.Signature) != ed25519.SignatureSize {
			return r, errors.New("invalid signature length")
		}

		if !ed25519.Verify(tx.PublicKey, tx.Inner, tx.Signature) {
			return r, errors.New("invalid signature")
		}
	}

	origin := loom.Address{
		ChainID: state.Block().ChainID,
		Local:   loom.LocalAddressFromPublicKey(tx.PublicKey),
	}

	ctx := context.WithValue(state.Context(), ContextKeyOrigin, origin)
	return next(state.WithContext(ctx), tx.Inner)
})

func nonceKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte("nonce"), addr.Bytes())
}

func Nonce(state loomchain.ReadOnlyState, addr loom.Address) uint64 {
	return loomchain.NewSequence(nonceKey(addr)).Value(state)
}

var NonceTxMiddleware = loomchain.TxMiddlewareFunc(func(
	state loomchain.State,
	txBytes []byte,
	next loomchain.TxHandlerFunc,
) (loomchain.TxHandlerResult, error) {
	var r loomchain.TxHandlerResult
	origin := Origin(state.Context())
	if origin.IsEmpty() {
		return r, errors.New("transaction has no origin")
	}
	seq := loomchain.NewSequence(nonceKey(origin)).Next(state)

	var tx NonceTx
	err := proto.Unmarshal(txBytes, &tx)
	if err != nil {
		return r, err
	}

	if tx.Sequence != seq {
		return r, errors.New("sequence number does not match")
	}

	return next(state, tx.Inner)
})
