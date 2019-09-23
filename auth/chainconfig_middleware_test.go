// +build evm

package auth

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/auth"
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"

	"github.com/loomnetwork/loomchain/auth/keys"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/features"
	appstate "github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/txhandler"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"
)

func TestChainConfigMiddlewareSingleChain(t *testing.T) {
	origBytes := []byte("hello")
	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	signer := auth.NewEd25519Signer([]byte(privKey))
	signedTx := auth.SignTx(signer, origBytes)
	signedTxBytes, err := proto.Marshal(signedTx)
	require.NoError(t, err)
	state := appstate.NewStoreState(nil, store.NewMemStore(), abci.Header{ChainID: "default"}, nil, nil)
	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	addresMapperAddr := fakeCtx.CreateContract(address_mapper.Contract)
	amCtx := contractpb.WrapPluginContext(fakeCtx.WithAddress(addresMapperAddr))
	authConfig := keys.Config{
		Chains: map[string]keys.ChainConfig{},
	}

	chainConfigMiddleware := NewChainConfigMiddleware(&authConfig, func(state appstate.State) (contractpb.StaticContext, error) { return amCtx, nil })
	_, err = chainConfigMiddleware.ProcessTx(state, signedTxBytes,
		func(state appstate.State, txBytes []byte, isCheckTx bool) (txhandler.TxHandlerResult, error) {
			require.Equal(t, txBytes, origBytes)
			return txhandler.TxHandlerResult{}, nil
		}, false,
	)
	require.NoError(t, err)
}

func TestChainConfigMiddlewareMultipleChain(t *testing.T) {
	state := appstate.NewStoreState(nil, store.NewMemStore(), abci.Header{ChainID: defaultLoomChainId}, nil, nil)
	state.SetFeature(features.AuthSigTxFeaturePrefix+"default", true)
	state.SetFeature(features.AuthSigTxFeaturePrefix+"eth", true)
	state.SetFeature(features.AuthSigTxFeaturePrefix+"tron", true)
	state.SetFeature(features.AuthSigTxFeaturePrefix+"binance", true)
	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	addresMapperAddr := fakeCtx.CreateContract(address_mapper.Contract)
	amCtx := contractpb.WrapPluginContext(fakeCtx.WithAddress(addresMapperAddr))

	ctx := context.WithValue(state.Context(), keys.ContextKeyOrigin, origin)

	chains := map[string]keys.ChainConfig{
		"default": {
			TxType:      keys.LoomSignedTxType,
			AccountType: keys.NativeAccountType,
		},
		"eth": {
			TxType:      keys.EthereumSignedTxType,
			AccountType: keys.MappedAccountType,
		},
		"tron": {
			TxType:      keys.TronSignedTxType,
			AccountType: keys.MappedAccountType,
		},
		"binance": {
			TxType:      keys.BinanceSignedTxType,
			AccountType: keys.MappedAccountType,
		},
	}
	authConfig := keys.Config{
		Chains: chains,
	}

	tmx := NewChainConfigMiddleware(
		&authConfig,
		func(_ appstate.State) (contractpb.StaticContext, error) { return amCtx, nil },
	)

	txSigned := mockEd25519SignedTx(t, priKey1)
	_, err := throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)
}
