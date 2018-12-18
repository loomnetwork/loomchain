package auth

import (
	"context"
	"testing"

	proto "github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
	lauth "github.com/loomnetwork/loomchain/privval/auth"
	"github.com/loomnetwork/loomchain/store"
)

func TestSignatureTxMiddleware(t *testing.T) {
	origBytes := []byte("hello")
	signer := lauth.NewSigner(nil)
	signedTx := lauth.SignTx(signer, origBytes)
	signedTxBytes, _ := proto.Marshal(signedTx)
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{}, nil)
	SignatureTxMiddleware.ProcessTx(state, signedTxBytes,
		func(state loomchain.State, txBytes []byte) (loomchain.TxHandlerResult, error) {
			require.Equal(t, txBytes, origBytes)
			return loomchain.TxHandlerResult{}, nil
		},
	)
}
func TestSignatureTxMiddlewareMultipleTxSameBlock(t *testing.T) {
	pubkey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}

	nonceTxBytes, err := proto.Marshal(&NonceTx{
		Inner:    []byte{},
		Sequence: 1,
	})
	require.NoError(t, err)

	nonceTxBytes2, err := proto.Marshal(&NonceTx{
		Inner:    []byte{},
		Sequence: 2,
	})
	require.NoError(t, err)

	origin := loom.Address{
		ChainID: "default",
		Local:   loom.LocalAddressFromPublicKey(pubkey),
	}

	ctx := context.WithValue(nil, ContextKeyOrigin, origin)

	state := loomchain.NewStoreState(ctx, store.NewMemStore(), abci.Header{Height: 27}, nil)

	_, err = NonceTxMiddleware(state, nonceTxBytes,
		func(state loomchain.State, txBytes []byte) (loomchain.TxHandlerResult, error) {
			return loomchain.TxHandlerResult{}, nil
		},
	)
	require.Nil(t, err)

	//State is reset on every run
	ctx2 := context.WithValue(nil, ContextKeyOrigin, origin)
	state2 := loomchain.NewStoreState(ctx2, store.NewMemStore(), abci.Header{Height: 27}, nil)

	//If we get the same sequence number in same block we should get an error
	_, err = NonceTxMiddleware(state2, nonceTxBytes,
		func(state2 loomchain.State, txBytes []byte) (loomchain.TxHandlerResult, error) {
			return loomchain.TxHandlerResult{}, nil
		},
	)
	require.Errorf(t, err, "sequence number does not match")

	//State is reset on every run
	ctx3 := context.WithValue(nil, ContextKeyOrigin, origin)
	state3 := loomchain.NewStoreState(ctx3, store.NewMemStore(), abci.Header{Height: 27}, nil)

	//If we get to tx with incrementing sequence numbers we should be fine in the same block
	_, err = NonceTxMiddleware(state3, nonceTxBytes2,
		func(state3 loomchain.State, txBytes []byte) (loomchain.TxHandlerResult, error) {
			return loomchain.TxHandlerResult{}, nil
		},
	)
	require.Nil(t, err)

	///--------------increase block height should kill cache
	//State is reset on every run
	ctx4 := context.WithValue(nil, ContextKeyOrigin, origin)
	state4 := loomchain.NewStoreState(ctx4, store.NewMemStore(), abci.Header{Height: 28}, nil)

	//If we get to tx with incrementing sequence numbers we should be fine in the same block
	_, err = NonceTxMiddleware(state4, nonceTxBytes,
		func(state4 loomchain.State, txBytes []byte) (loomchain.TxHandlerResult, error) {
			return loomchain.TxHandlerResult{}, nil
		},
	)
	require.Nil(t, err)

}
