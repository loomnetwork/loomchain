package throttle

import (
	"context"
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
	"testing"
)

var (
	oracleAddr = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
)

func throttleMiddlewareHandler(ttm loomchain.TxMiddlewareFunc, state loomchain.State, tx auth.SignedTx, ctx context.Context) (loomchain.TxHandlerResult, error) {
	return ttm.ProcessTx(state.WithContext(ctx), tx.Inner,
		func(state loomchain.State, txBytes []byte) (res loomchain.TxHandlerResult, err error) {
			return loomchain.TxHandlerResult{}, err
		},
	)
}

func TestThrottleTxMiddlewareDeployEnable(t *testing.T) {
	log.Setup("debug", "file://-")
	log.Root.With("module", "throttle-middleware")
	origBytes := []byte("origin")
	_, privKey, err := ed25519.GenerateKey(nil)
	require.Nil(t, err)

	depoyTx, err := proto.Marshal(&loomchain.Transaction{
		Id:   1,
		Data: origBytes,
	})
	require.Nil(t, err)

	signer := auth.NewEd25519Signer([]byte(privKey))
	signedTxDeploy := auth.SignTx(signer, depoyTx)
	signedTxBytesDeploy, err := proto.Marshal(signedTxDeploy)
	//state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{})
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{})
	var txDeploy auth.SignedTx
	err = proto.Unmarshal(signedTxBytesDeploy, &txDeploy)
	require.Nil(t, err)

	require.Equal(t, len(txDeploy.PublicKey), ed25519.PublicKeySize)
	require.Equal(t, len(txDeploy.Signature), ed25519.SignatureSize)
	require.True(t, ed25519.Verify(txDeploy.PublicKey, txDeploy.Inner, txDeploy.Signature))

	origin := loom.Address{
		ChainID: state.Block().ChainID,
		Local:   loom.LocalAddressFromPublicKey(txDeploy.PublicKey),
	}

	ctx := context.WithValue(state.Context(), loomAuth.ContextKeyOrigin, origin)

	// origin is the Tx sender. To make the sender the oracle we it as the oracle in GetThrottleTxMiddleWare. Otherwise use a different address (oracleAddr) in GetThrottleTxMiddleWare
	tmx1 := GetThrottleTxMiddleWare(false, true, oracleAddr)
	_, err = throttleMiddlewareHandler(tmx1, state, txDeploy, ctx)
	require.Error(t, err, "test: deploy should be enabled")
	require.Equal(t, err.Error(), "throttle: deploy tx not enabled")
	tmx2 := GetThrottleTxMiddleWare(false, true, origin)
	_, err = throttleMiddlewareHandler(tmx2, state, txDeploy, ctx)
	require.NoError(t, err, "test: oracle should be able to deploy even with deploy diabled")
	tmx3 := GetThrottleTxMiddleWare(true, true, oracleAddr)
	_, err = throttleMiddlewareHandler(tmx3, state, txDeploy, ctx)
	require.NoError(t, err, "test: origin should be able to deploy")
	tmx4 := GetThrottleTxMiddleWare(true, true, origin)
	_, err = throttleMiddlewareHandler(tmx4, state, txDeploy, ctx)
	require.NoError(t, err, "test: oracles should be able to deploy")
}
