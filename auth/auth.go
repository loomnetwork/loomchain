package auth

import (
	"context"
	"errors"

	"github.com/gogo/protobuf/proto"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ripemd160"

	"github.com/loomnetwork/loom"
	"github.com/loomnetwork/loom/util"
)

type contextKey string

func (c contextKey) String() string {
	return "auth " + string(c)
}

var (
	contextKeySender = contextKey("sender")
)

func makeLocalAddress(pubKey []byte) loom.LocalAddress {
	hasher := ripemd160.New()
	hasher.Write(pubKey[:]) // does not error

	var addr loom.LocalAddress
	copy(addr, hasher.Sum(nil))
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

	sender := &loom.Address{
		ChainID: state.Block().ChainID,
		Local:   makeLocalAddress(tx.PublicKey),
	}

	ctx := context.WithValue(state.Context(), contextKeySender, sender)
	return next(state.WithContext(ctx), tx.Inner)
})

func nonceKey(addr *loom.Address) []byte {
	return util.PrefixKey([]byte("nonce"), addr.Bytes())
}

var NonceTxMiddleware = loom.TxMiddlewareFunc(func(
	state loom.State,
	txBytes []byte,
	next loom.TxHandlerFunc,
) (loom.TxHandlerResult, error) {
	var r loom.TxHandlerResult
	sender := Sender(state.Context())
	if sender == nil {
		return r, errors.New("transaction has no sender")
	}
	seq := loom.NewSequence(nonceKey(sender)).Next(state)

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
func SignTx(txBytes []byte, key ed25519.PrivateKey) *SignedTx {
	sig := ed25519.Sign(key, txBytes)
	pubKey := key.Public().(ed25519.PublicKey)
	return &SignedTx{
		Inner:     txBytes,
		Signature: sig,
		PublicKey: []byte(pubKey),
	}
}
