package throttle

import (
	"context"
	"fmt"
	"github.com/loomnetwork/loomchain/plugin"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	loomAuth "github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"
)

var (
	oracleAddr = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")

	maxKarma int64 = 10000
	oracle         = oracleAddr.MarshalPB()

	sources = []*karma.SourceReward{
		&karma.SourceReward{"sms", 10},
		&karma.SourceReward{"oauth", 10},
		&karma.SourceReward{"token", 5},
	}
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
	var maxAccessCount = int64(10)
	var sessionDuration = int64(600)
	origBytes := []byte("origin")
	_, privKey, err := ed25519.GenerateKey(nil)
	require.Nil(t, err)

	depoyTx, err := proto.Marshal(&loomchain.Transaction{
		Id:   1,
		Data: origBytes,
	})
	nonceDeploy, err := proto.Marshal(&auth.NonceTx{
		Inner:    depoyTx,
		Sequence: 4,
	})
	require.Nil(t, err)

	signer := auth.NewEd25519Signer([]byte(privKey))
	signedTxDeploy := auth.SignTx(signer, nonceDeploy)
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

	var createRegistry factory.RegistryFactoryFunc
	createRegistry, err = factory.NewRegistryFactory(factory.LatestRegistryVersion)
	require.Nil(t, err)
	registryObject := createRegistry(state)

	contractContext := contractpb.WrapPluginContext(
		goloomplugin.CreateFakeContext(oracleAddr, oracleAddr),
	)

	err = registryObject.Register("karma", contractContext.ContractAddress(), origin)
	require.Nil(t, err)

	contractAddress, err := registryObject.Resolve("karma")
	require.Nil(t, err)

	config := karma.Config{
		Sources:        sources,
		LastUpdateTime: contractContext.Now().Unix(),
	}

	configb, err := proto.Marshal(&config)
	require.Nil(t, err)

	contractState := loomchain.StateWithPrefix(plugin.DataPrefix(contractAddress), state)
	contractState.Set(karma.GetConfigKey(), configb)

	// origin is the Tx sender. To make the sender the oracle we it as the oracle in GetThrottleTxMiddleWare. Otherwise use a different address (oracleAddr) in GetThrottleTxMiddleWare
	tmx1 := GetThrottleTxMiddleWare(maxAccessCount, sessionDuration, true, maxKarma, false, true, oracleAddr, factory.LatestRegistryVersion)
	_, err = throttleMiddlewareHandler(tmx1, state, txDeploy, ctx)
	require.Error(t, err, "test: deploy should be enabled")
	require.Equal(t, err.Error(), "throttle: deploy tx not enabled")
	tmx2 := GetThrottleTxMiddleWare(maxAccessCount, sessionDuration, true, maxKarma, false, true, origin, factory.LatestRegistryVersion)
	_, err = throttleMiddlewareHandler(tmx2, state, txDeploy, ctx)
	require.NoError(t, err, "test: oracle should be able to deploy even with deploy diabled")
	tmx3 := GetThrottleTxMiddleWare(maxAccessCount, sessionDuration, true, maxKarma, true, true, oracleAddr, factory.LatestRegistryVersion)
	_, err = throttleMiddlewareHandler(tmx3, state, txDeploy, ctx)
	require.Error(t, err, "test: origin should have no karma")
	tmx4 := GetThrottleTxMiddleWare(maxAccessCount, sessionDuration, true, maxKarma, true, true, origin, factory.LatestRegistryVersion)
	_, err = throttleMiddlewareHandler(tmx4, state, txDeploy, ctx)
	require.NoError(t, err, "test: oracles should be able to deploy without karma")

	tmx5 := GetThrottleTxMiddleWare(maxAccessCount, sessionDuration, false, maxKarma, false, true, oracleAddr, factory.LatestRegistryVersion)
	_, err = throttleMiddlewareHandler(tmx5, state, txDeploy, ctx)
	require.Error(t, err, "test: deploy should be enabled")
	require.Equal(t, err.Error(), "throttle: deploy tx not enabled")
	tmx6 := GetThrottleTxMiddleWare(maxAccessCount, sessionDuration, false, maxKarma, false, true, origin, factory.LatestRegistryVersion)
	_, err = throttleMiddlewareHandler(tmx6, state, txDeploy, ctx)
	require.NoError(t, err, "test: oracle should be able to deploy even with deloy disabled")
	tmx7 := GetThrottleTxMiddleWare(maxAccessCount, sessionDuration, false, maxKarma, true, true, oracleAddr, factory.LatestRegistryVersion)
	_, err = throttleMiddlewareHandler(tmx7, state, txDeploy, ctx)
	require.NoError(t, err, "test: karma should not matter")
	tmx8 := GetThrottleTxMiddleWare(maxAccessCount, sessionDuration, false, maxKarma, true, true, origin, factory.LatestRegistryVersion)
	_, err = throttleMiddlewareHandler(tmx8, state, txDeploy, ctx)
	require.NoError(t, err, "test: oracles should be able to deploy without karma")

}

func TestThrottleTxMiddleware(t *testing.T) {
	t.Skip("todo")
	log.Setup("debug", "file://-")
	log.Root.With("module", "throttle-middleware")
	var maxAccessCount = int64(10)
	var sessionDuration = int64(600)
	origBytes := []byte("origin")
	_, privKey, err := ed25519.GenerateKey(nil)
	require.Nil(t, err)

	signer := auth.NewEd25519Signer([]byte(privKey))
	signedTx := auth.SignTx(signer, origBytes)
	signedTxBytes, err := proto.Marshal(signedTx)
	//state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{})
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

	var createRegistry factory.RegistryFactoryFunc
	createRegistry, err = factory.NewRegistryFactory(factory.LatestRegistryVersion)
	require.Nil(t, err)
	registryObject := createRegistry(state)

	contractContext := contractpb.WrapPluginContext(
		goloomplugin.CreateFakeContext(oracleAddr, oracleAddr),
	)

	err = registryObject.Register("karma", contractContext.ContractAddress(), origin)
	require.Nil(t, err)

	contractAddress, err := registryObject.Resolve("karma")
	require.Nil(t, err)

	config := karma.Config{
		Sources:        sources,
		LastUpdateTime: contractContext.Now().Unix(),
	}

	configb, err := proto.Marshal(&config)
	require.Nil(t, err)

	contractState := loomchain.StateWithPrefix(plugin.DataPrefix(contractAddress), state)
	contractState.Set(karma.GetConfigKey(), configb)

	tmx := GetThrottleTxMiddleWare(maxAccessCount, sessionDuration, true, maxKarma, true, true, origin, factory.LatestRegistryVersion)
	i := int64(1)

	totalAccessCount := maxAccessCount * 2

	fmt.Println(ctx, tmx, i, totalAccessCount)

	for i <= totalAccessCount {
		_, err := throttleMiddlewareHandler(tmx, state, tx, ctx)
		if i <= maxAccessCount {
			require.Error(t, err, "test: origin has no karma1")
		} else {
			require.Error(t, err, fmt.Sprintf("Out of access count for current session: %d out of %d, Try after sometime!", i, maxAccessCount))
		}
		i += 1
	}

}
