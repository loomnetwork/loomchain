package throttle

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	loomAuth "github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"
	"testing"
)

const (
	deployId = 1
	callId   = 2
)

var (
	addr1  = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	origin = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")

	sources = []*ktypes.KarmaSourceReward{
		{"sms", 1},
		{"oauth", 2},
		{"token", 3},
	}

	sourceStates = []*ktypes.KarmaSource{
		{"sms", 2},
		{"oauth", 1},
		{"token", 1},
	}
)

func TestDeployThrottleTxMiddleware(t *testing.T) {
	log.Setup("debug", "file://-")
	log.Root.With("module", "throttle-middleware")
	var maxAccessCount = int64(5)
	var sessionDuration = int64(600)
	var deployCount = int64(10)

	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{})

	var createRegistry factory.RegistryFactoryFunc
	createRegistry, err := factory.NewRegistryFactory(factory.LatestRegistryVersion)
	require.NoError(t, err)
	registryObject := createRegistry(state)

	contractContext := contractpb.WrapPluginContext(
		goloomplugin.CreateFakeContext(addr1, addr1),
	)
	karmaAddr := contractContext.ContractAddress()
	karmaState := loomchain.StateWithPrefix(plugin.DataPrefix(karmaAddr), state)
	require.NoError(t, registryObject.Register("karma", karmaAddr, addr1))

	karmaSources := ktypes.KarmaSources{
		Sources: sources,
	}
	sourcesB, err := proto.Marshal(&karmaSources)
	require.NoError(t, err)
	karmaState.Set(karma.SourcesKey, sourcesB)

	sourceStatesB, err := proto.Marshal(&ktypes.KarmaState{
		SourceStates: sourceStates,
	})
	require.NoError(t, err)
	stateKey := karma.GetUserStateKey(origin.MarshalPB())
	karmaState.Set(stateKey, sourceStatesB)

	ctx := context.WithValue(state.Context(), loomAuth.ContextKeyOrigin, origin)

	tmx := GetKarmaMiddleWare(
		true,
		maxAccessCount,
		sessionDuration,
		deployCount,
		factory.LatestRegistryVersion,
	)

	totalAccessCount := maxAccessCount * 2
	for i := int64(1); i <= totalAccessCount; i++ {

		txSigned := mockSignedTx(t, state, uint64(i), deployId)
		_, err := throttleMiddlewareHandler(tmx, state, txSigned, ctx)

		if i <= maxAccessCount {
			require.NoError(t, err)
		} else {
			require.Error(t, err, fmt.Sprintf("Out of access count for current session: %d out of %d, Try after sometime!", i, maxAccessCount))
		}
	}
}

func TestCallThrottleTxMiddleware(t *testing.T) {
	log.Setup("debug", "file://-")
	log.Root.With("module", "throttle-middleware")
	var maxAccessCount = int64(5)
	var sessionDuration = int64(600)
	var deployCount = int64(10)

	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{})

	var createRegistry factory.RegistryFactoryFunc
	createRegistry, err := factory.NewRegistryFactory(factory.LatestRegistryVersion)
	require.NoError(t, err)
	registryObject := createRegistry(state)

	contractContext := contractpb.WrapPluginContext(
		goloomplugin.CreateFakeContext(addr1, addr1),
	)
	karmaAddr := contractContext.ContractAddress()
	karmaState := loomchain.StateWithPrefix(plugin.DataPrefix(karmaAddr), state)
	require.NoError(t, registryObject.Register("karma", karmaAddr, addr1))

	karmaSources := ktypes.KarmaSources{
		Sources: sources,
	}
	sourcesB, err := proto.Marshal(&karmaSources)
	require.NoError(t, err)
	karmaState.Set(karma.SourcesKey, sourcesB)

	sourceStatesB, err := proto.Marshal(&ktypes.KarmaState{
		SourceStates: sourceStates,
	})
	require.NoError(t, err)
	stateKey := karma.GetUserStateKey(origin.MarshalPB())
	karmaState.Set(stateKey, sourceStatesB)

	ctx := context.WithValue(state.Context(), loomAuth.ContextKeyOrigin, origin)

	tmx := GetKarmaMiddleWare(
		true,
		maxAccessCount,
		sessionDuration,
		deployCount,
		factory.LatestRegistryVersion,
	)
	karmaCount := karma.CalculateTotalKarma(karmaSources, ktypes.KarmaState{
		SourceStates: sourceStates,
	})
	totalAccessCount := maxAccessCount*2 + karmaCount
	for i := int64(1); i <= totalAccessCount; i++ {
		txSigned := mockSignedTx(t, state, uint64(i), callId)
		_, err := throttleMiddlewareHandler(tmx, state, txSigned, ctx)

		if i <= maxAccessCount+karmaCount {
			require.NoError(t, err)
		} else {
			require.Error(t, err, fmt.Sprintf("Out of access count for current session: %d out of %d, Try after sometime!", i, maxAccessCount))
		}
	}
}

func mockSignedTx(t *testing.T, state loomchain.State, sequence uint64, id uint32) auth.SignedTx {
	origBytes := []byte("origin")
	_, privKey, err := ed25519.GenerateKey(nil)
	require.Nil(t, err)

	tx, err := proto.Marshal(&loomchain.Transaction{
		Id:   id,
		Data: origBytes,
	})
	nonceTx, err := proto.Marshal(&auth.NonceTx{
		Inner:    tx,
		Sequence: sequence,
	})
	require.Nil(t, err)

	signer := auth.NewEd25519Signer([]byte(privKey))
	signedTx := auth.SignTx(signer, nonceTx)
	signedTxBytes, err := proto.Marshal(signedTx)
	var txSigned auth.SignedTx
	err = proto.Unmarshal(signedTxBytes, &txSigned)
	require.Nil(t, err)

	require.Equal(t, len(txSigned.PublicKey), ed25519.PublicKeySize)
	require.Equal(t, len(txSigned.Signature), ed25519.SignatureSize)
	require.True(t, ed25519.Verify(txSigned.PublicKey, txSigned.Inner, txSigned.Signature))
	return txSigned
}
