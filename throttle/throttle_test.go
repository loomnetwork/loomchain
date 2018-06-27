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
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	"github.com/loomnetwork/go-loom"
	"fmt"
	"runtime/debug"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/loomnetwork/loomchain/plugin"
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	ktypes "github.com/loomnetwork/loomchain/builtin/plugins/karma/types"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	maxKarma int64	= 10000
	oraclePublicAddress = addr1.Local.String()

	sources 			= map[string]int64{
								"sms": 10,
								"oauth": 10,
								"token": 5,
							}
)

func throttleMiddlewareHandler(ttm loomchain.TxMiddlewareFunc, state loomchain.State, tx auth.SignedTx, ctx context.Context) (loomchain.TxHandlerResult, error) {
	defer func() {
		if rval := recover(); rval != nil {
			logger := log.Root
			logger.Error("Panic in TX Handler", "rvalue", rval)
			println(debug.Stack())
		}
	}()
	return ttm.ProcessTx(state.WithContext(ctx), tx.Inner,
		func(state loomchain.State, txBytes []byte) (res loomchain.TxHandlerResult, err error) {
			return loomchain.TxHandlerResult{}, err
		},
	)
}

func TestThrottleTxMiddleware(t *testing.T) {
	log.Setup("debug", "file://-")
	log.Root.With("module", "throttle-middleware")
	var maxAccessCount = int64(10);
	var sessionDuration = int64(600);
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

	ctx := context.WithValue(state.Context(), loomAuth.ContextKeyOrigin, origin)


	registryObject := &registry.StateRegistry{
		State: state,
	}


	contractContext := contractpb.WrapPluginContext(
		goloomplugin.CreateFakeContext(addr1, addr1),
	)

	err = registryObject.Register("karma", contractContext.ContractAddress(), origin)
	require.Nil(t, err)

	contractAddress, err := registryObject.Resolve("karma")
	require.Nil(t, err)

	karmaOwner := &ktypes.KarmaUser{
		oraclePublicAddress,
	}

	config := karma.Config{
		MaxKarma: 						maxKarma,
		Oracle:				 			karmaOwner,
		Sources: 						sources,
		LastUpdateTime: 				contractContext.Now().Unix(),
	}


	configb, err := proto.Marshal(&config)
	require.Nil(t, err)

	contractState := loomchain.StateWithPrefix(plugin.DataPrefix(contractAddress), state)
	contractState.Set(karma.GetConfigKey(), configb)

	tmx := GetThrottleTxMiddleWare(maxAccessCount,sessionDuration)
	i := int64(1)

	totalAccessCount := maxAccessCount*2

	fmt.Println(ctx, tmx, i, totalAccessCount)

	for i <= totalAccessCount {
		_, err := throttleMiddlewareHandler(tmx, state , tx , ctx )
		if i <= maxAccessCount {
			require.Nil(t, err)
		}else{
			require.Error(t, err, fmt.Sprintf("Out of access count for current session: %d out of %d, Try after sometime!", i, maxAccessCount))
		}
		i += 1
	}


}