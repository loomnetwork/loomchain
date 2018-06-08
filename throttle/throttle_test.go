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
	"github.com/loomnetwork/go-loom"
	"fmt"
)


func throttleMiddlewareHandler(t *testing.T, ttm loomchain.TxMiddlewareFunc, th *Throttle, i int16, state loomchain.State, tx auth.SignedTx, ctx context.Context) (loomchain.TxHandlerResult, error) {
	return ttm.ProcessTx(state.WithContext(ctx), tx.Inner,
		func(state loomchain.State, txBytes []byte) (res loomchain.TxHandlerResult, err error) {
			return loomchain.TxHandlerResult{}, err
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

	th := NewThrottle(10,100)

	ctx := context.WithValue(state.Context(), loomAuth.ContextKeyOrigin, origin)
	tmx := GetThrottleTxMiddleWare(th)
	i := int16(1)

	totalAccessCount := th.maxAccessCount*2

	for i <= totalAccessCount {

		_, err := throttleMiddlewareHandler(t, tmx, th, i , state , tx , ctx )

		if i <= th.maxAccessCount {
			require.Nil(t, err)
			require.Equal(t, th.getAccessCount(), i)
		}else{
			require.Error(t, err, fmt.Sprintf("Out of access count for current session: %d out of %d, Try after sometime!", i, th.maxAccessCount))
		}
		i += 1
	}


}