package auth

import (
	"context"
	"errors"

	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/ed25519"
	"golang.org/x/crypto/ripemd160"

	"github.com/loomnetwork/loom"
)

type contextKey string

func (c contextKey) String() string {
	return "auth " + string(c)
}

var (
	contextKeySender = contextKey("sender")
)

func makeLocalAddress(pubKey [ed25519.PublicKeySize]byte) loom.LocalAddress {
	hasher := ripemd160.New()
	hasher.Write(pubKey[:]) // does not error

	var addr loom.LocalAddress
	copy(addr[:], hasher.Sum(nil))
	return addr
}

func Sender(ctx context.Context) *loom.Address {
	return ctx.Value(contextKeySender).(*loom.Address)
}

var SignatureTxMiddleware = loom.TxMiddlewareFunc(func(
	state loom.State,
	txBytes []byte,
	next loom.TxHandlerFunc,
) (loom.TxHandlerResult, error) {
	r := loom.TxHandlerResult{}

	var tx SignedTx
	err := proto.Unmarshal(txBytes, &tx)
	if err != nil {
		return r, err
	}

	var pubKey [ed25519.PublicKeySize]byte
	var sig [ed25519.SignatureSize]byte

	if len(tx.PublicKey) != len(pubKey) {
		return r, errors.New("invalid public key length")
	}

	if len(tx.Signature) != len(sig) {
		return r, errors.New("invalid signature length")
	}

	copy(pubKey[:], tx.PublicKey)
	copy(sig[:], tx.Signature)

	if !ed25519.Verify(&pubKey, tx.Inner, &sig) {
		return r, errors.New("invalid signature")
	}

	sender := &loom.Address{
		ChainID: state.Block().ChainID,
		Local:   makeLocalAddress(pubKey),
	}

	ctx := context.WithValue(state.Context(), contextKeySender, sender)
	return next(state.WithContext(ctx), tx.Inner)
})
