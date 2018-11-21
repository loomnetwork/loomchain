package auth

import (
	"github.com/loomnetwork/go-loom/auth"
)

type Signer interface {
	auth.Signer
}

type SignedTx = auth.SignedTx
type NonceTx = auth.NonceTx

func SignTx(signer Signer, txBytes []byte) *auth.SignedTx {
	return auth.SignTx(signer, txBytes)
}
