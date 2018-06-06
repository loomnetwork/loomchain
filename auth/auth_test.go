package auth

import (
	"testing"
	"context"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"

	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/go-loom"
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

func TestThrottleTxMiddleware(t *testing.T) {

	origBytes := []byte("origin")
	_, privKey, err := ed25519.GenerateKey(nil)
	require.Nil(t, err)

	signer := auth.NewEd25519Signer([]byte(privKey))
	signedTx := auth.SignTx(signer, origBytes)
	signedTxBytes, err := proto.Marshal(signedTx)
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{})

	var tx SignedTx
	err = proto.Unmarshal(signedTxBytes, &tx)
	require.Nil(t, err)

	require.Equal(t, len(tx.PublicKey), ed25519.PublicKeySize)

	require.Equal(t,  len(tx.Signature), ed25519.SignatureSize)

	require.True(t,  ed25519.Verify(tx.PublicKey, tx.Inner, tx.Signature))

	origin := loom.Address{
		ChainID: state.Block().ChainID,
		Local:   loom.LocalAddressFromPublicKey(tx.PublicKey),
	}

	ctx := context.WithValue(state.Context(), contextKeyOrigin, origin)

	ThrottleTxMiddleware.ProcessTx(state.WithContext(ctx), tx.Inner,
		func(state loomchain.State, txBytes []byte) (loomchain.TxHandlerResult, error) {
			origin := Origin(state.Context())
			require.False(t,  origin.IsEmpty())
			require.Equal(t, getSessionAccessCount(state, origin), int16(1))
			return loomchain.TxHandlerResult{}, nil
		},
	)

}