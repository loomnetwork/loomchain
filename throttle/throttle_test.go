package throttle

import (
	"context"
	"fmt"
	"runtime/debug"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/loomchain"
	loomAuth "github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"
)

func throttleMiddlewareHandler(t *testing.T, ttm loomchain.TxMiddlewareFunc, state loomchain.State, tx auth.SignedTx, ctx context.Context) (loomchain.TxHandlerResult, error) {
	defer func() {
		if rval := recover(); rval != nil {
			logger := log.Default
			logger.Error("Panic in TX Handler", "rvalue", rval)
			println(debug.Stack())
		}
	}()
	return ttm.ProcessTx(state.WithContext(ctx), tx.Inner,
		func(state loomchain.State, txBytes []byte) (res loomchain.TxHandlerResult, err error) {
			return loomchain.TxHandlerResult{}, err
		},
	)
}

func TestThrottleTxMiddleware(t *testing.T) {
	log.Setup("debug", "file://-")
	log.Default.With("module", "throttle-middleware")
	var maxAccessCount = int64(10)
	var sessionDuration = int64(600)
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

	require.Equal(t, len(tx.Signature), ed25519.SignatureSize)

	require.True(t, ed25519.Verify(tx.PublicKey, tx.Inner, tx.Signature))

	origin := loom.Address{
		ChainID: state.Block().ChainID,
		Local:   loom.LocalAddressFromPublicKey(tx.PublicKey),
	}

	ctx := context.WithValue(state.Context(), loomAuth.ContextKeyOrigin, origin)
	tmx := GetThrottleTxMiddleWare(maxAccessCount, sessionDuration)
	i := int64(1)

	totalAccessCount := maxAccessCount * 2

	fmt.Println(ctx, tmx, i, totalAccessCount)

	for i <= totalAccessCount {
		_, err := throttleMiddlewareHandler(t, tmx, state, tx, ctx)
		if i <= maxAccessCount {
			require.Nil(t, err)
		} else {
			require.Error(t, err, fmt.Sprintf("Out of access count for current session: %d out of %d, Try after sometime!", i, maxAccessCount))
		}
		i += 1
	}

}
