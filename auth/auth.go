package auth

import (
	"context"
	"errors"

	"github.com/gogo/protobuf/proto"
	"golang.org/x/crypto/ed25519"

	"github.com/loomnetwork/loom"
	lp "github.com/loomnetwork/loom-plugin"
	"github.com/loomnetwork/loom-plugin/util"
)

type contextKey string

func (c contextKey) String() string {
	return "auth " + string(c)
}

var (
	contextKeyOrigin = contextKey("origin")
)

func Origin(ctx context.Context) lp.Address {
	return ctx.Value(contextKeyOrigin).(lp.Address)
}

var SignatureTxMiddleware = loom.TxMiddlewareFunc(func(
	state loom.State,
	txBytes []byte,
	next loom.TxHandlerFunc,
) (loom.TxHandlerResult, error) {
	var r loom.TxHandlerResult

	var tx SignedTx
	err := proto.Unmarshal(txBytes, &tx)
	if err != nil {
		return r, err
	}

	if len(tx.PublicKey) != ed25519.PublicKeySize {
		return r, errors.New("invalid public key length")
	}

	if len(tx.Signature) != ed25519.SignatureSize {
		return r, errors.New("invalid signature length")
	}

	if !ed25519.Verify(tx.PublicKey, tx.Inner, tx.Signature) {
		return r, errors.New("invalid signature")
	}

	origin := lp.Address{
		ChainID: state.Block().ChainID,
		Local:   lp.LocalAddressFromPublicKey(tx.PublicKey),
	}

	ctx := context.WithValue(state.Context(), contextKeyOrigin, origin)
	return next(state.WithContext(ctx), tx.Inner)
})

func nonceKey(addr lp.Address) []byte {
	return util.PrefixKey([]byte("nonce"), addr.Bytes())
}

func Nonce(state loom.ReadOnlyState, addr lp.Address) uint64 {
	return loom.NewSequence(nonceKey(addr)).Value(state)
}

var NonceTxMiddleware = loom.TxMiddlewareFunc(func(
	state loom.State,
	txBytes []byte,
	next loom.TxHandlerFunc,
) (loom.TxHandlerResult, error) {
	var r loom.TxHandlerResult
	origin := Origin(state.Context())
	if origin.IsEmpty() {
		return r, errors.New("transaction has no origin")
	}
	seq := loom.NewSequence(nonceKey(origin)).Next(state)

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

// SignTx generates a signed tx containing the given bytes.
func SignTx(signer lp.Signer, txBytes []byte) *SignedTx {
	return &SignedTx{
		Inner:     txBytes,
		Signature: signer.Sign(txBytes),
		PublicKey: signer.PublicKey(),
	}
}
