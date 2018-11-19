package throttle

import (
	"context"
	"testing"

	"github.com/loomnetwork/loomchain/privval"
	"github.com/tendermint/tendermint/crypto/secp256k1"

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
	var privKey []byte
	var signer auth.Signer
	var err error

	log.Setup("debug", "file://-")
	log.Root.With("module", "throttle-middleware")
	origBytes := []byte("origin")

	if privval.EnableSecp256k1 {
		privKey = secp256k1.GenPrivKey().Bytes()
		signer = privval.NewSecp256k1Signer(privKey)
	} else {
		_, privKey, err = ed25519.GenerateKey(nil)
		require.NoError(t, err)

		signer = auth.NewEd25519Signer([]byte(privKey))
	}

	depoyTx, err := proto.Marshal(&loomchain.Transaction{
		Id:   1,
		Data: origBytes,
	})
	require.NoError(t, err)

	signedTxDeploy := auth.SignTx(signer, depoyTx)
	signedTxBytesDeploy, err := proto.Marshal(signedTxDeploy)
	require.NoError(t, err)
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{})
	var txDeploy auth.SignedTx
	err = proto.Unmarshal(signedTxBytesDeploy, &txDeploy)
	require.NoError(t, err)

	if privval.EnableSecp256k1 {
		var pubKey secp256k1.PubKeySecp256k1
		var sign secp256k1.SignatureSecp256k1

		require.Equal(t, len(txDeploy.PublicKey), secp256k1.PubKeySecp256k1Size)
		copy(pubKey[:], txDeploy.PublicKey)
		copy(sign[:], txDeploy.Signature)
		require.True(t, pubKey.VerifyBytes(txDeploy.Inner, sign))
	} else {
		require.Equal(t, len(txDeploy.PublicKey), ed25519.PublicKeySize)
		require.Equal(t, len(txDeploy.Signature), ed25519.SignatureSize)
		require.True(t, ed25519.Verify(txDeploy.PublicKey, txDeploy.Inner, txDeploy.Signature))
	}

	origin := loom.Address{
		ChainID: state.Block().ChainID,
		Local:   loom.LocalAddressFromPublicKey(txDeploy.PublicKey),
	}

	ctx := context.WithValue(state.Context(), loomAuth.ContextKeyOrigin, origin)

	// origin is the Tx sender. To make the sender the oracle we it as the oracle in GetThrottleTxMiddleWare. Otherwise use a different address (oracleAddr) in GetThrottleTxMiddleWare
	tmx1 := GetThrottleTxMiddleWare(false, true, oracleAddr)
	_, err = throttleMiddlewareHandler(tmx1, state, txDeploy, ctx)
	require.Error(t, err, "test: deploy should be enabled")
	require.Equal(t, err.Error(), "throttle: deploy transactions not enabled")
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

func TestThrottleTxMiddlewareCallEnable(t *testing.T) {
	var privKey []byte
	var signer auth.Signer
	var err error

	log.Setup("debug", "file://-")
	log.Root.With("module", "throttle-middleware")
	origBytes := []byte("origin")

	if privval.EnableSecp256k1 {
		privKey = secp256k1.GenPrivKey().Bytes()
		signer = privval.NewSecp256k1Signer(privKey)
	} else {
		_, privKey, err := ed25519.GenerateKey(nil)
		require.NoError(t, err)

		signer = auth.NewEd25519Signer(privKey)
	}

	callTx, err := proto.Marshal(&loomchain.Transaction{
		Id:   2,
		Data: origBytes,
	})
	require.NoError(t, err, "marshal loomchain.Transaction")

	signedTxCall := auth.SignTx(signer, callTx)
	signedTxBytesCall, err := proto.Marshal(signedTxCall)
	require.NoError(t, err)
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{})
	var txCall auth.SignedTx
	err = proto.Unmarshal(signedTxBytesCall, &txCall)
	require.NoError(t, err)

	if privval.EnableSecp256k1 {
		var pubKey secp256k1.PubKeySecp256k1
		var sign secp256k1.SignatureSecp256k1

		require.Equal(t, len(txCall.PublicKey), secp256k1.PubKeySecp256k1Size)
		copy(pubKey[:], txCall.PublicKey)
		copy(sign[:], txCall.Signature)
		require.True(t, pubKey.VerifyBytes(txCall.Inner, sign))
	} else {
		require.Equal(t, len(txCall.PublicKey), ed25519.PublicKeySize)
		require.Equal(t, len(txCall.Signature), ed25519.SignatureSize)
		require.True(t, ed25519.Verify(txCall.PublicKey, txCall.Inner, txCall.Signature))
	}

	origin := loom.Address{
		ChainID: state.Block().ChainID,
		Local:   loom.LocalAddressFromPublicKey(txCall.PublicKey),
	}
	ctx := context.WithValue(state.Context(), loomAuth.ContextKeyOrigin, origin)

	// origin is the Tx sender. To make the sender the oracle we it as the oracle in GetThrottleTxMiddleWare. Otherwise use a different address (oracleAddr) in GetThrottleTxMiddleWare
	tmx1 := GetThrottleTxMiddleWare(false, false, oracleAddr)
	_, err = throttleMiddlewareHandler(tmx1, state, txCall, ctx)
	require.Error(t, err, "test: call should be enabled")
	require.Equal(t, err.Error(), "throttle: call transactions not enabled")
	tmx2 := GetThrottleTxMiddleWare(false, false, origin)
	_, err = throttleMiddlewareHandler(tmx2, state, txCall, ctx)
	require.NoError(t, err, "test: oracle should be able to call even with call diabled")
	tmx3 := GetThrottleTxMiddleWare(false, true, oracleAddr)
	_, err = throttleMiddlewareHandler(tmx3, state, txCall, ctx)
	require.NoError(t, err, "test: origin should be able to call")
	tmx4 := GetThrottleTxMiddleWare(false, true, origin)
	_, err = throttleMiddlewareHandler(tmx4, state, txCall, ctx)
	require.NoError(t, err, "test: oracles should be able to call")
}
