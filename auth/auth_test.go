package auth

import (
	"context"
	"testing"

	proto "github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"

	loom "github.com/loomnetwork/go-loom"
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
	require.NoError(t, err)
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{}, nil, nil)
	SignatureTxMiddleware.ProcessTx(state, signedTxBytes,
		func(state loomchain.State, txBytes []byte, isCheckTx bool) (loomchain.TxHandlerResult, error) {
			require.Equal(t, txBytes, origBytes)
			return loomchain.TxHandlerResult{}, nil
		}, false,
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

	ctx := context.WithValue(context.Background(), ContextKeyOrigin, origin)
	ctx = context.WithValue(ctx, ContextKeyCheckTx, true)

	state := loomchain.NewStoreState(ctx, store.NewMemStore(), abci.Header{Height: 27}, nil, nil)

	_, err = NonceTxMiddleware(state, nonceTxBytes,
		func(state loomchain.State, txBytes []byte, isCheckTx bool) (loomchain.TxHandlerResult, error) {
			return loomchain.TxHandlerResult{}, nil
		}, false,
	)
	require.Nil(t, err)
	NonceTxPostNonceMiddleware(state, nonceTxBytes, loomchain.TxHandlerResult{}, nil)

	//State is reset on every run
	ctx2 := context.WithValue(context.Background(), ContextKeyOrigin, origin)
	state2 := loomchain.NewStoreState(ctx2, store.NewMemStore(), abci.Header{Height: 27}, nil, nil)
	_ = context.WithValue(ctx2, ContextKeyCheckTx, true)

	//If we get the same sequence number in same block we should get an error
	_, err = NonceTxMiddleware(state2, nonceTxBytes,
		func(state2 loomchain.State, txBytes []byte, isCheckTx bool) (loomchain.TxHandlerResult, error) {
			return loomchain.TxHandlerResult{}, nil
		}, true,
	)
	require.Errorf(t, err, "sequence number does not match")
	//	NonceTxPostNonceMiddleware shouldnt get called on an error

	//State is reset on every run
	ctx3 := context.WithValue(context.Background(), ContextKeyOrigin, origin)
	state3 := loomchain.NewStoreState(ctx3, store.NewMemStore(), abci.Header{Height: 27}, nil, nil)
	_ = context.WithValue(ctx3, ContextKeyCheckTx, true)

	//If we get to tx with incrementing sequence numbers we should be fine in the same block
	_, err = NonceTxMiddleware(state3, nonceTxBytes2,
		func(state3 loomchain.State, txBytes []byte, isCheckTx bool) (loomchain.TxHandlerResult, error) {
			return loomchain.TxHandlerResult{}, nil
		}, true,
	)
	require.Nil(t, err)
	NonceTxPostNonceMiddleware(state, nonceTxBytes, loomchain.TxHandlerResult{}, nil)

	//Try a deliverTx at same height it should be fine
	ctx3Dx := context.WithValue(context.Background(), ContextKeyOrigin, origin)
	state3Dx := loomchain.NewStoreState(ctx3Dx, store.NewMemStore(), abci.Header{Height: 27}, nil, nil)
	_ = context.WithValue(ctx3Dx, ContextKeyCheckTx, true)

	_, err = NonceTxMiddleware(state3Dx, nonceTxBytes,
		func(state3 loomchain.State, txBytes []byte, isCheckTx bool) (loomchain.TxHandlerResult, error) {
			return loomchain.TxHandlerResult{}, nil
		}, false,
	)
	require.Nil(t, err)
	NonceTxPostNonceMiddleware(state, nonceTxBytes, loomchain.TxHandlerResult{}, nil)

	///--------------increase block height should kill cache
	//State is reset on every run
	ctx4 := context.WithValue(nil, ContextKeyOrigin, origin)
	state4 := loomchain.NewStoreState(ctx4, store.NewMemStore(), abci.Header{Height: 28}, nil, nil)

	//If we get to tx with incrementing sequence numbers we should be fine in the same block
	_, err = NonceTxMiddleware(state4, nonceTxBytes,
		func(state4 loomchain.State, txBytes []byte, isCheckTx bool) (loomchain.TxHandlerResult, error) {
			return loomchain.TxHandlerResult{}, nil
		}, true,
	)
	require.Nil(t, err)
	NonceTxPostNonceMiddleware(state, nonceTxBytes, loomchain.TxHandlerResult{}, nil)

}
