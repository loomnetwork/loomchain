package throttle

import (
	"context"
	"testing"
	"github.com/gogo/protobuf/proto"
	abci "github.com/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"
	loomAuth "github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	"fmt"
	"github.com/loomnetwork/go-loom"
)


func throttleMiddlewareHandler(t *testing.T, cfg Config, i int16, state loomchain.State, tx auth.SignedTx, ctx context.Context) {
	defer func() {
		if rval := recover(); rval != nil {
			require.Equal(t, rval, fmt.Sprintf("Ran out of access count for current session: %d out of %d, Try after sometime!", i, 100))
			t.Log( rval)
		}
	}()

	ThrottleTxMiddleware.ProcessTx(state.WithContext(ctx), tx.Inner,
		func(state loomchain.State, txBytes []byte) (res loomchain.TxHandlerResult, err error) {

			origin := loomAuth.Origin(state.Context())
			require.False(t,  origin.IsEmpty())
			if i <= cfg.ThrottleMaxAccessCount {
				require.Nil(t, err)
				require.Equal(t, getSessionAccessCount(state, origin), i)
			}

			return loomchain.TxHandlerResult{}, nil
		},
	)
}

func TestThrottleTxMiddleware(t *testing.T) {
	cfg := DefaultLimits()

	origBytes := []byte("origin")
	_, privKey, err := ed25519.GenerateKey(nil)
	require.Nil(t, err)

	signer := auth.NewEd25519Signer([]byte(privKey))
	signedTx := auth.SignTx(signer, origBytes)
	signedTxBytes, err := proto.Marshal(signedTx)
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{})

	var tx auth.SignedTx
	err = proto.Unmarshal(signedTxBytes, &tx)
	require.Nil(t, err)

	require.Equal(t, len(tx.PublicKey), ed25519.PublicKeySize)

	require.Equal(t,  len(tx.Signature), ed25519.SignatureSize)

	require.True(t,  ed25519.Verify(tx.PublicKey, tx.Inner, tx.Signature))

	origin := loom.Address{
		ChainID: state.Block().ChainID,
		Local:   loom.LocalAddressFromPublicKey(tx.PublicKey),
	}

	ctx := context.WithValue(state.Context(), loomAuth.ContextKeyOrigin, origin)

	i := int16(1)
	for i <= cfg.ThrottleMaxAccessCount*2 {
		throttleMiddlewareHandler(t,  cfg, i , state , tx , ctx )
		i += 1
	}


}