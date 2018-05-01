package auth

import (
	"testing"

	proto "github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"

	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
)

func TestSignatureTxMiddleware(t *testing.T) {
	origBytes := []byte("hello")
	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	signer := auth.NewEd25519Signer([]byte(privKey))
	signedTx := auth.SignTx(signer, origBytes)
	signedTxBytes, err := proto.Marshal(signedTx)
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{})
	SignatureTxMiddleware.ProcessTx(state, signedTxBytes,
		func(state loomchain.State, txBytes []byte) (loomchain.TxHandlerResult, error) {
			require.Equal(t, txBytes, origBytes)
			return loomchain.TxHandlerResult{}, nil
		},
	)
}
