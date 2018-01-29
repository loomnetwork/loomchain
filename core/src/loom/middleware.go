package loom

import (
	"context"
	"errors"

	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/ed25519"
)

type TxMiddleware interface {
	Handle(ctx context.Context, txBytes []byte, next TxHandlerFunc) error
}

type TxMiddlewareFunc func(ctx context.Context, txBytes []byte, next TxHandlerFunc) error

func (f TxMiddlewareFunc) Handle(ctx context.Context, txBytes []byte, next TxHandlerFunc) error {
	return f(ctx, txBytes, next)
}

func MiddlewareTxHandler(
	middlewares []TxMiddleware,
	handler TxHandler,
) TxHandler {
	next := TxHandlerFunc(handler.Handle)

	for i := len(middlewares) - 1; i >= 0; i-- {
		m := middlewares[i]
		// Need local var otherwise infinite loop occurs
		nextLocal := next
		next = func(ctx context.Context, txBytes []byte) error {
			return m.Handle(ctx, txBytes, nextLocal)
		}
	}

	return next
}

var NoopTxHandler = TxHandlerFunc(func(ctx context.Context, txBytes []byte) error {
	return nil
})

var SignatureTxMiddleware = TxMiddlewareFunc(func(ctx context.Context, txBytes []byte, next TxHandlerFunc) error {
	var tx SignedTx

	err := proto.Unmarshal(txBytes, &tx)
	if err != nil {
		return err
	}

	for _, signer := range tx.Signers {
		var pubKey [ed25519.PublicKeySize]byte
		var sig [ed25519.SignatureSize]byte

		if len(signer.PublicKey) != len(pubKey) {
			return errors.New("invalid public key length")
		}

		if len(signer.Signature) != len(sig) {
			return errors.New("invalid signature length")
		}

		copy(pubKey[:], signer.PublicKey)
		copy(sig[:], signer.Signature)

		if !ed25519.Verify(&pubKey, tx.Inner, &sig) {
			return errors.New("invalid signature")
		}

		// TODO: set some context
	}

	return next(ctx, tx.Inner)
})
