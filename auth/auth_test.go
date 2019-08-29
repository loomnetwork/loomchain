package auth

import (
	"context"
	"errors"
	"testing"

	proto "github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/config"
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

	cfg := config.DefaultConfig()
	cfg.NonceHandler.IncNonceOnFailedTx = true

	ctx := context.WithValue(context.Background(), ContextKeyOrigin, origin)
	ctx = context.WithValue(ctx, ContextKeyCheckTx, true)
	kvStore := store.NewMemStore()
	state := loomchain.NewStoreState(ctx, kvStore, abci.Header{Height: 27}, nil, nil).WithOnChainConfig(cfg)

	_, err = NonceTxHandler.Nonce(state, kvStore, nonceTxBytes,
		func(state loomchain.State, txBytes []byte, isCheckTx bool) (loomchain.TxHandlerResult, error) {
			return loomchain.TxHandlerResult{}, nil
		}, false,
	)
	require.Nil(t, err)
	NonceTxPostNonceMiddleware(state, nonceTxBytes, loomchain.TxHandlerResult{}, nil, false)

	//State is reset on every run
	ctx2 := context.WithValue(context.Background(), ContextKeyOrigin, origin)
	kvStore2 := store.NewMemStore()
	state2 := loomchain.NewStoreState(ctx2, kvStore2, abci.Header{Height: 27}, nil, nil).WithOnChainConfig(cfg)
	ctx2 = context.WithValue(ctx2, ContextKeyCheckTx, true)

	//If we get the same sequence number in same block we should get an error
	_, err = NonceTxHandler.Nonce(state2, kvStore2, nonceTxBytes,
		func(state2 loomchain.State, txBytes []byte, isCheckTx bool) (loomchain.TxHandlerResult, error) {
			return loomchain.TxHandlerResult{}, nil
		}, true,
	)
	require.Errorf(t, err, "sequence number does not match")
	//	NonceTxPostNonceMiddleware shouldnt get called on an error

	//State is reset on every run
	ctx3 := context.WithValue(context.Background(), ContextKeyOrigin, origin)
	kvStore3 := store.NewMemStore()
	state3 := loomchain.NewStoreState(ctx3, kvStore3, abci.Header{Height: 27}, nil, nil).WithOnChainConfig(cfg)
	ctx3 = context.WithValue(ctx3, ContextKeyCheckTx, true)

	//If we get to tx with incrementing sequence numbers we should be fine in the same block
	_, err = NonceTxHandler.Nonce(state3, kvStore3, nonceTxBytes2,
		func(state3 loomchain.State, txBytes []byte, isCheckTx bool) (loomchain.TxHandlerResult, error) {
			return loomchain.TxHandlerResult{}, nil
		}, true,
	)
	require.Nil(t, err)
	NonceTxPostNonceMiddleware(state, nonceTxBytes2, loomchain.TxHandlerResult{}, nil, false)

	//Try a deliverTx at same height it should be fine
	ctx3Dx := context.WithValue(context.Background(), ContextKeyOrigin, origin)
	kvStore3Dx := store.NewMemStore()
	state3Dx := loomchain.NewStoreState(ctx3Dx, kvStore3Dx, abci.Header{Height: 27}, nil, nil).WithOnChainConfig(cfg)
	ctx3Dx = context.WithValue(ctx3Dx, ContextKeyCheckTx, true)

	_, err = NonceTxHandler.Nonce(state3Dx, kvStore3Dx, nonceTxBytes,
		func(state3 loomchain.State, txBytes []byte, isCheckTx bool) (loomchain.TxHandlerResult, error) {
			return loomchain.TxHandlerResult{}, nil
		}, false,
	)
	require.Nil(t, err)
	NonceTxPostNonceMiddleware(state, nonceTxBytes, loomchain.TxHandlerResult{}, nil, false)

	///--------------increase block height should kill cache
	//State is reset on every run
	ctx4 := context.WithValue(nil, ContextKeyOrigin, origin)
	kvStore4 := store.NewMemStore()
	state4 := loomchain.NewStoreState(ctx4, kvStore4, abci.Header{Height: 28}, nil, nil).WithOnChainConfig(cfg)
	//If we get to tx with incrementing sequence numbers we should be fine in the same block
	_, err = NonceTxHandler.Nonce(state4, kvStore4, nonceTxBytes,
		func(state4 loomchain.State, txBytes []byte, isCheckTx bool) (loomchain.TxHandlerResult, error) {
			return loomchain.TxHandlerResult{}, nil
		}, true,
	)
	require.Nil(t, err)
	NonceTxPostNonceMiddleware(state, nonceTxBytes, loomchain.TxHandlerResult{}, nil, false)
}

func TestRevertedTxNonceMiddleware(t *testing.T) {
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

	cfg := config.DefaultConfig()
	cfg.NonceHandler.IncNonceOnFailedTx = true

	origin := loom.Address{
		ChainID: "default",
		Local:   loom.LocalAddressFromPublicKey(pubkey),
	}

	ctx := context.WithValue(context.Background(), ContextKeyOrigin, origin)
	ctx = context.WithValue(ctx, ContextKeyCheckTx, true)
	kvStore := store.NewMemStore()
	storeTx := store.WrapAtomic(kvStore).BeginTx()
	state := loomchain.NewStoreState(ctx, storeTx, abci.Header{Height: 27}, nil, nil).WithOnChainConfig(cfg)

	// Nonce is 0
	currentNonce := Nonce(state, origin)
	require.Equal(t, uint64(0), currentNonce)

	// Send a successful tx
	_, err = NonceTxHandler.Nonce(state, kvStore, nonceTxBytes,
		func(state loomchain.State, txBytes []byte, isCheckTx bool) (loomchain.TxHandlerResult, error) {
			return loomchain.TxHandlerResult{}, nil
		}, false,
	)
	require.Nil(t, err)
	NonceTxPostNonceMiddleware(state, nonceTxBytes, loomchain.TxHandlerResult{}, nil, false)
	storeTx.Commit()
	storeTx.Rollback()

	// Send a failed tx, nonce should increase even though the transaction is reverted
	_, err = NonceTxHandler.Nonce(state, kvStore, nonceTxBytes2,
		func(state loomchain.State, txBytes []byte, isCheckTx bool) (loomchain.TxHandlerResult, error) {
			return loomchain.TxHandlerResult{}, errors.New("EVM transaction reverted")
		}, false,
	)
	require.Error(t, err)
	storeTx.Rollback()

	currentNonce = Nonce(state, origin)
	require.Equal(t, uint64(2), currentNonce)

	// disable IncrementNonceOnFailedTx
	cfg.NonceHandler.IncNonceOnFailedTx = false

	nonceTxBytes3, err := proto.Marshal(&NonceTx{
		Inner:    []byte{},
		Sequence: 3,
	})
	require.NoError(t, err)

	// Send another failed tx, nonce should not increment because the transaction reverted
	_, err = NonceTxHandler.Nonce(state, kvStore, nonceTxBytes3,
		func(state loomchain.State, txBytes []byte, isCheckTx bool) (loomchain.TxHandlerResult, error) {
			return loomchain.TxHandlerResult{}, errors.New("EVM transaction reverted")
		}, false,
	)
	require.Error(t, err)
	storeTx.Rollback()

	// expect nonce to be the same
	currentNonce = Nonce(state, origin)
	require.Equal(t, uint64(2), currentNonce)
}
