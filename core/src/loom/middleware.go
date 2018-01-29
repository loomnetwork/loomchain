package loom

import (
	"context"
	"errors"

	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/ed25519"
)

type TxMiddleware interface {
	Handle(ctx context.Context, txBytes []byte) ([]byte, error)
}

type TxMiddlewareFunc func(ctx context.Context, txBytes []byte) ([]byte, error)

func (f TxMiddlewareFunc) Handle(ctx context.Context, txBytes []byte) ([]byte, error) {
	return f(ctx, txBytes)
}

func HandleSignatureTx(ctx context.Context, txBytes []byte) ([]byte, error) {
	var tx SignedTx

	err := proto.Unmarshal(txBytes, &tx)
	if err != nil {
		return nil, err
	}

	for _, signer := range tx.Signers {
		var pubKey [ed25519.PublicKeySize]byte
		var sig [ed25519.SignatureSize]byte

		if len(signer.PublicKey) != len(pubKey) {
			return nil, errors.New("invalid public key length")
		}

		if len(signer.Signature) != len(sig) {
			return nil, errors.New("invalid signature length")
		}

		copy(pubKey[:], signer.PublicKey)
		copy(sig[:], signer.Signature)

		if !ed25519.Verify(&pubKey, tx.Inner, &sig) {
			return nil, errors.New("invalid signature")
		}

		// TODO: set some context
	}

	return tx.Inner, nil
}

var SignatureTxMiddleware = TxMiddlewareFunc(HandleSignatureTx)
