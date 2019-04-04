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
	"github.com/loomnetwork/loomchain/store"
)

const (
	featurePrefix = "auth:sigtx:"
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

	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{ChainID: "default"}, nil, nil)
	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	addresMapperAddr := fakeCtx.CreateContract(address_mapper.Contract)
	amCtx := contractpb.WrapPluginContext(fakeCtx.WithAddress(addresMapperAddr))
	authConfig := Config{
		Chains: map[string]ChainConfig{},
	}

	chainConfigMiddleware := NewChainConfigMiddleware(&authConfig, func(state loomchain.State) (contractpb.Context, error) { return amCtx, nil })
	_, err = chainConfigMiddleware.ProcessTx(state, signedTxBytes,
		func(state loomchain.State, txBytes []byte, isCheckTx bool) (loomchain.TxHandlerResult, error) {
			require.Equal(t, txBytes, origBytes)
			return loomchain.TxHandlerResult{}, nil
		}, false,
	)
	require.NoError(t, err)
}

// TODO: This test doesn't really test multiple chains at the moment, chains have to be enabled via
//       feature flags.
func TestChainConfigMiddlewareMultipleChain(t *testing.T) {
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{ChainID: defaultLoomChainId}, nil, nil)
	state.SetFeature(featurePrefix+"default", true)
	state.SetFeature(featurePrefix+"tron", true)
	state.SetFeature(featurePrefix+"eth", true)
	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	addresMapperAddr := fakeCtx.CreateContract(address_mapper.Contract)
	amCtx := contractpb.WrapPluginContext(fakeCtx.WithAddress(addresMapperAddr))

	ctx := context.WithValue(state.Context(), ContextKeyOrigin, origin)

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
	}
	authConfig := Config{
		Chains: chains,
	}

	tmx := NewChainConfigMiddleware(
		&authConfig,
		func(state loomchain.State) (contractpb.Context, error) { return amCtx, nil },
	)

	txSigned := mockEd25519SignedTx(t, priKey1)
	_, err := throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)
}
