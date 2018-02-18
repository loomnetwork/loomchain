package loom

import (
	"errors"

	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/ed25519"
)

type TxMiddleware interface {
	Handle(state State, txBytes []byte, next TxHandlerFunc) (TxHandlerResult, error)
}

type TxMiddlewareFunc func(state State, txBytes []byte, next TxHandlerFunc) (TxHandlerResult, error)

func (f TxMiddlewareFunc) Handle(state State, txBytes []byte, next TxHandlerFunc) (TxHandlerResult, error) {
	return f(state, txBytes, next)
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
		next = func(state State, txBytes []byte) (TxHandlerResult, error) {
			return m.Handle(state, txBytes, nextLocal)
		}
	}

	return next
}

var NoopTxHandler = TxHandlerFunc(func(state State, txBytes []byte) (TxHandlerResult, error) {
	return TxHandlerResult{}, nil
})

var SignatureTxMiddleware = TxMiddlewareFunc(func(state State, txBytes []byte, next TxHandlerFunc) (TxHandlerResult, error) {
	r := TxHandlerResult{}

	var tx SignedTx
	err := proto.Unmarshal(txBytes, &tx)
	if err != nil {
		return r, err
	}

	for _, signer := range tx.Signers {
		var pubKey [ed25519.PublicKeySize]byte
		var sig [ed25519.SignatureSize]byte

		if len(signer.PublicKey) != len(pubKey) {
			return r, errors.New("invalid public key length")
		}

		if len(signer.Signature) != len(sig) {
			return r, errors.New("invalid signature length")
		}

		copy(pubKey[:], signer.PublicKey)
		copy(sig[:], signer.Signature)

		if !ed25519.Verify(&pubKey, tx.Inner, &sig) {
			return r, errors.New("invalid signature")
		}

		// TODO: set some context
	}

	return next(state, tx.Inner)
})
