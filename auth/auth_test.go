package auth

import (
	"testing"

	proto "github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"golang.org/x/crypto/ed25519"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/privval"
	"github.com/loomnetwork/loomchain/privval/auth"
	"github.com/loomnetwork/loomchain/store"
)

func TestSignatureTxMiddleware(t *testing.T) {
	var signer auth.Signer
	origBytes := []byte("hello")
	if privval.EnableSecp256k1 {
		privKey := secp256k1.GenPrivKey()
		signer = privval.NewSecp256k1Signer(privKey.Bytes())
	} else {
		_, privKey, err := ed25519.GenerateKey(nil)
		if err != nil {
			panic(err)
		}
		signer = auth.NewEd25519Signer([]byte(privKey))
	}
	signedTx := auth.SignTx(signer, origBytes)
	signedTxBytes, err := proto.Marshal(signedTx)
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{}, nil)
	SignatureTxMiddleware.ProcessTx(state, signedTxBytes,
		func(state loomchain.State, txBytes []byte) (loomchain.TxHandlerResult, error) {
			require.Equal(t, txBytes, origBytes)
			return loomchain.TxHandlerResult{}, nil
		},
	)
}
