package throttle

import (
	"context"
	"fmt"
	`github.com/loomnetwork/loomchain/plugin`
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
	addr1          = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")

	maxKarma int64 = 10000
	oracle         = addr1.MarshalPB()
	
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
	
	var createRegistry   factory.RegistryFactoryFunc
	createRegistry, err = factory.NewRegistryFactory(factory.LatestRegistryVersion)
	require.Nil(t, err)
	registryObject := createRegistry(state)
	
	contractContext := contractpb.WrapPluginContext(
		goloomplugin.CreateFakeContext(addr1, addr1),
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
