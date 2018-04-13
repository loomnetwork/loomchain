package auth

import (
	"testing"

	proto "github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/loom"
	lp "github.com/loomnetwork/loom-plugin"
	"github.com/loomnetwork/loom/store"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"
)

func TestSignatureTxMiddleware(t *testing.T) {
	origBytes := []byte("hello")
	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	signer := lp.NewEd25519Signer([]byte(privKey))
	signedTx := SignTx(signer, origBytes)
	signedTxBytes, err := proto.Marshal(signedTx)
	state := loom.NewStoreState(nil, store.NewMemStore(), abci.Header{})
	SignatureTxMiddleware.ProcessTx(state, signedTxBytes,
		func(state loom.State, txBytes []byte) (loom.TxHandlerResult, error) {
			require.Equal(t, txBytes, origBytes)
			return loom.TxHandlerResult{}, nil
		},
	)
}
