package client

import (
	"github.com/loomnetwork/loom/auth"
)

type DAppChainClient interface {
	CommitTx(signer auth.Signer, txBytes []byte) error
}
