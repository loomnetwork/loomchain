package throttle

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/loomchain"
	loomAuth "github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"
)

var (
	oracleAddr = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
)

func throttleMiddlewareHandler(ttm loomchain.TxMiddlewareFunc, state loomchain.State, tx auth.SignedTx, ctx context.Context) (loomchain.TxHandlerResult, error) {
	return ttm.ProcessTx(
		state.WithContext(ctx),
		tx.Inner,
		func(state loomchain.State, txBytes []byte, isCheckTx bool) (res loomchain.TxHandlerResult, err error) {

			var nonceTx loomAuth.NonceTx
			if err := proto.Unmarshal(txBytes, &nonceTx); err != nil {
				return res, errors.Wrap(err, "throttle: unwrap nonce Tx")
			}

			var tx loomchain.Transaction
			if err := proto.Unmarshal(nonceTx.Inner, &tx); err != nil {
				return res, errors.New("throttle: unmarshal tx")
			}
			var msg vm.MessageTx
			if err := proto.Unmarshal(tx.Data, &msg); err != nil {
				return res, errors.Wrapf(err, "unmarshal message tx %v", tx.Data)
			}
			var info string
			var data []byte
			if  tx.Id == callId {
				var callTx vm.CallTx
				if err := proto.Unmarshal(msg.Data, &callTx);  err != nil {
					return res, errors.Wrapf(err, "unmarshal call tx %v", msg.Data)
				}
				if callTx.VmType == vm.VMType_EVM {
					info = utils.CallEVM
				} else {
					info = utils.CallPlugin
				}
			} else if tx.Id == deployId {
				var deployTx vm.DeployTx
				if err := proto.Unmarshal(msg.Data, &deployTx);  err != nil {
					return res, errors.Wrapf(err, "unmarshal call tx %v", msg.Data)
				}
				if deployTx.VmType == vm.VMType_EVM {
					info = utils.DeployEvm
				} else {
					info = utils.DeployPlugin
				}
				data, err = proto.Marshal(&vm.DeployResponse{
					// Always use same contract address,
					// Might want to change that later.
					Contract: contract.MarshalPB(),
				})
			}

			return loomchain.TxHandlerResult{ Data: data, Info: info}, err
		},
		false,
	)
}

func TestThrottleTxMiddlewareDeployEnable(t *testing.T) {
	log.Setup("debug", "file://-")
	log.Root.With("module", "throttle-middleware")
	origBytes := []byte("origin")
	_, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	depoyTx, err := proto.Marshal(&loomchain.Transaction{
		Id:   1,
		Data: origBytes,
	})
	require.NoError(t, err)

	signer := auth.NewEd25519Signer([]byte(privKey))
	signedTxDeploy := auth.SignTx(signer, depoyTx)
	signedTxBytesDeploy, err := proto.Marshal(signedTxDeploy)
	require.NoError(t, err)
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{}, nil)
	var txDeploy auth.SignedTx
	err = proto.Unmarshal(signedTxBytesDeploy, &txDeploy)
	require.NoError(t, err)

	require.Equal(t, len(txDeploy.PublicKey), ed25519.PublicKeySize)
	require.Equal(t, len(txDeploy.Signature), ed25519.SignatureSize)
	require.True(t, ed25519.Verify(txDeploy.PublicKey, txDeploy.Inner, txDeploy.Signature))

	origin := loom.Address{
		ChainID: state.Block().ChainID,
		Local:   loom.LocalAddressFromPublicKey(txDeploy.PublicKey),
	}

	ctx := context.WithValue(state.Context(), loomAuth.ContextKeyOrigin, origin)

	// origin is the Tx sender. To make the sender the oracle we it as the oracle in GetThrottleTxMiddleWare. Otherwise use a different address (oracleAddr) in GetThrottleTxMiddleWare
	tmx1 := GetThrottleTxMiddleWare(func(int64) bool { return false }, func(int64) bool { return true }, oracleAddr)
	_, err = throttleMiddlewareHandler(tmx1, state, txDeploy, ctx)
	require.Error(t, err, "test: deploy should be enabled")
	require.Equal(t, err.Error(), "throttle: deploy transactions not enabled")
	tmx2 := GetThrottleTxMiddleWare(func(int64) bool { return false }, func(int64) bool { return true }, origin)
	_, err = throttleMiddlewareHandler(tmx2, state, txDeploy, ctx)
	require.NoError(t, err, "test: oracle should be able to deploy even with deploy diabled")
	tmx3 := GetThrottleTxMiddleWare(func(int64) bool { return true }, func(int64) bool { return true }, oracleAddr)
	_, err = throttleMiddlewareHandler(tmx3, state, txDeploy, ctx)
	require.NoError(t, err, "test: origin should be able to deploy")
	tmx4 := GetThrottleTxMiddleWare(func(int64) bool { return true }, func(int64) bool { return true }, origin)
	_, err = throttleMiddlewareHandler(tmx4, state, txDeploy, ctx)
	require.NoError(t, err, "test: oracles should be able to deploy")
}

func TestThrottleTxMiddlewareCallEnable(t *testing.T) {
	log.Setup("debug", "file://-")
	log.Root.With("module", "throttle-middleware")
	origBytes := []byte("origin")
	_, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	callTx, err := proto.Marshal(&loomchain.Transaction{
		Id:   2,
		Data: origBytes,
	})
	require.NoError(t, err, "marshal loomchain.Transaction")

	signer := auth.NewEd25519Signer([]byte(privKey))
	signedTxCall := auth.SignTx(signer, callTx)
	signedTxBytesCall, err := proto.Marshal(signedTxCall)
	require.NoError(t, err)
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{}, nil)
	var txCall auth.SignedTx
	err = proto.Unmarshal(signedTxBytesCall, &txCall)
	require.NoError(t, err)

	require.Equal(t, len(txCall.PublicKey), ed25519.PublicKeySize)
	require.Equal(t, len(txCall.Signature), ed25519.SignatureSize)
	require.True(t, ed25519.Verify(txCall.PublicKey, txCall.Inner, txCall.Signature))

	origin := loom.Address{
		ChainID: state.Block().ChainID,
		Local:   loom.LocalAddressFromPublicKey(txCall.PublicKey),
	}
	ctx := context.WithValue(state.Context(), loomAuth.ContextKeyOrigin, origin)

	// origin is the Tx sender. To make the sender the oracle we it as the oracle in GetThrottleTxMiddleWare. Otherwise use a different address (oracleAddr) in GetThrottleTxMiddleWare
	tmx1 := GetThrottleTxMiddleWare(func(int64) bool { return false }, func(int64) bool { return false }, oracleAddr)
	_, err = throttleMiddlewareHandler(tmx1, state, txCall, ctx)
	require.Error(t, err, "test: call should be enabled")
	require.Equal(t, err.Error(), "throttle: call transactions not enabled")
	tmx2 := GetThrottleTxMiddleWare(func(int64) bool { return false }, func(int64) bool { return false }, origin)
	_, err = throttleMiddlewareHandler(tmx2, state, txCall, ctx)
	require.NoError(t, err, "test: oracle should be able to call even with call diabled")
	tmx3 := GetThrottleTxMiddleWare(func(int64) bool { return false }, func(int64) bool { return true }, oracleAddr)
	_, err = throttleMiddlewareHandler(tmx3, state, txCall, ctx)
	require.NoError(t, err, "test: origin should be able to call")
	tmx4 := GetThrottleTxMiddleWare(func(int64) bool { return false }, func(int64) bool { return true }, origin)
	_, err = throttleMiddlewareHandler(tmx4, state, txCall, ctx)
	require.NoError(t, err, "test: oracles should be able to call")
}
