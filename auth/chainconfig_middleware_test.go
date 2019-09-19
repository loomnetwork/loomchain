// +build evm

package auth

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/auth"
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/features"
	"github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/store"
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
	s := state.NewStoreState(nil, store.NewMemStore(), abci.Header{ChainID: "default"}, nil, nil)
	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	addresMapperAddr := fakeCtx.CreateContract(address_mapper.Contract)
	amCtx := contractpb.WrapPluginContext(fakeCtx.WithAddress(addresMapperAddr))
	authConfig := Config{
		Chains: map[string]ChainConfig{},
	}

	chainConfigMiddleware := NewChainConfigMiddleware(&authConfig, func(_ state.State) (contractpb.StaticContext, error) { return amCtx, nil })
	_, err = chainConfigMiddleware.ProcessTx(s, signedTxBytes,
		func(_ state.State, txBytes []byte, isCheckTx bool) (loomchain.TxHandlerResult, error) {
			require.Equal(t, txBytes, origBytes)
			return loomchain.TxHandlerResult{}, nil
		}, false,
	)
	require.NoError(t, err)
}

func TestChainConfigMiddlewareMultipleChain(t *testing.T) {
	s := state.NewStoreState(nil, store.NewMemStore(), abci.Header{ChainID: defaultLoomChainId}, nil, nil)
	s.SetFeature(features.AuthSigTxFeaturePrefix+"default", true)
	s.SetFeature(features.AuthSigTxFeaturePrefix+"eth", true)
	s.SetFeature(features.AuthSigTxFeaturePrefix+"tron", true)
	s.SetFeature(features.AuthSigTxFeaturePrefix+"binance", true)
	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	addresMapperAddr := fakeCtx.CreateContract(address_mapper.Contract)
	amCtx := contractpb.WrapPluginContext(fakeCtx.WithAddress(addresMapperAddr))

	ctx := context.WithValue(s.Context(), ContextKeyOrigin, origin)

	chains := map[string]ChainConfig{
		"default": {
			TxType:      LoomSignedTxType,
			AccountType: NativeAccountType,
		},
		"eth": {
			TxType:      EthereumSignedTxType,
			AccountType: MappedAccountType,
		},
		"tron": {
			TxType:      TronSignedTxType,
			AccountType: MappedAccountType,
		},
		"binance": {
			TxType:      BinanceSignedTxType,
			AccountType: MappedAccountType,
		},
	}
	authConfig := Config{
		Chains: chains,
	}

	tmx := NewChainConfigMiddleware(
		&authConfig,
		func(_ state.State) (contractpb.StaticContext, error) { return amCtx, nil },
	)

	txSigned := mockEd25519SignedTx(t, priKey1)
	_, err := throttleMiddlewareHandler(tmx, s, txSigned, ctx)
	require.NoError(t, err)
}
