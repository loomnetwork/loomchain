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
	"github.com/pkg/errors"
	"fmt"
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

func rvalError(r interface{}) error {
	var err error
	switch x := r.(type) {
	case string:
		err = errors.New(x)
	case error:
		err = x
	default:
		err = errors.New("unknown panic")
	}
	return err
}

func throttleMiddlewareHandler(t *testing.T, i int16, state loomchain.State, tx SignedTx, ctx context.Context) {

	defer func() {
		rval := recover()
		if rval != nil {
			require.Equal(t, rval, fmt.Sprintf("Ran out of access count for current session: %d out of %d, Try after sometime!", i, 100))
			t.Log( rval)
		}
	}()

	ThrottleTxMiddleware.ProcessTx(state.WithContext(ctx), tx.Inner,
		func(state loomchain.State, txBytes []byte) (res loomchain.TxHandlerResult, err error) {

			origin := Origin(state.Context())
			require.False(t,  origin.IsEmpty())
			if i <= 100 {
				require.Nil(t, err)
				require.Equal(t, getSessionAccessCount(state, origin), i)
			}

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

	i := int16(1)
	for i <= 120 {
		throttleMiddlewareHandler(t , i , state , tx , ctx )
		i += 1
	}


}