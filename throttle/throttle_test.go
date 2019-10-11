// +build evm

package throttle

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	loomAuth "github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"
)

const (
	maxDeployCount  = int64(15)
	maxCallCount    = int64(10)
	sessionDuration = int64(600)
)

var (
	addr1  = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	origin = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")

	sources = []*ktypes.KarmaSourceReward{
		{Name: "sms", Reward: 1, Target: ktypes.KarmaSourceTarget_CALL},
		{Name: "oauth", Reward: 2, Target: ktypes.KarmaSourceTarget_CALL},
		{Name: "token", Reward: 3, Target: ktypes.KarmaSourceTarget_CALL},
		{Name: karma.CoinDeployToken, Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
	}

	sourceStates = []*ktypes.KarmaSource{
		{Name: "sms", Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(2)}},
		{Name: "oauth", Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(1)}},
		{Name: "token", Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(1)}},
		{Name: karma.CoinDeployToken, Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(maxDeployCount)}},
	}

	userState = ktypes.KarmaState{ //types.BigUInt
		SourceStates:     sourceStates,
		DeployKarmaTotal: &types.BigUInt{Value: *loom.NewBigUIntFromInt(1 * maxDeployCount)},
		CallKarmaTotal:   &types.BigUInt{Value: *loom.NewBigUIntFromInt(1*2 + 2*1 + 3*1)},
	}
)

func TestDeployThrottleTxMiddleware(t *testing.T) {
	log.Setup("debug", "file://-")
	log.Root.With("module", "throttle-middleware")

	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{}, nil, nil)

	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	karmaAddr := fakeCtx.CreateContract(karma.Contract)
	contractContext := contractpb.WrapPluginContext(fakeCtx.WithAddress(karmaAddr))

	// Init the karma contract
	karmaContract := &karma.Karma{}
	require.NoError(t, karmaContract.Init(contractContext, &ktypes.KarmaInitRequest{
		Sources: sources,
	}))

	// This can also be done on init, but more concise this way
	require.NoError(t, karma.AddKarma(contractContext, origin, sourceStates))

	ctx := context.WithValue(state.Context(), loomAuth.ContextKeyOrigin, origin)

	tmx := GetKarmaMiddleWare(
		true,
		maxCallCount,
		sessionDuration,
		func(state loomchain.State) (contractpb.Context, error) {
			return contractContext, nil
		},
	)

	deployKarma := userState.DeployKarmaTotal

	for i := int64(1); i <= deployKarma.Value.Int64()+1; i++ {
		txSigned := mockSignedTx(t, uint64(i), types.TxID_DEPLOY, vm.VMType_PLUGIN, contract)
		_, err := throttleMiddlewareHandler(tmx, state, txSigned, ctx)

		if i <= deployKarma.Value.Int64() {
			require.NoError(t, err)
		}
	}

	for i := int64(1); i <= deployKarma.Value.Int64()+1; i++ {
		txSigned := mockSignedTx(t, uint64(i), types.TxID_ETHEREUM, vm.VMType_EVM, loom.Address{})
		_, err := throttleMiddlewareHandler(tmx, state, txSigned, ctx)

		if i <= deployKarma.Value.Int64() {
			require.NoError(t, err)
		}
	}
}

func TestCallThrottleTxMiddleware(t *testing.T) {
	log.Setup("debug", "file://-")
	log.Root.With("module", "throttle-middleware")

	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{}, nil, nil)

	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	karmaAddr := fakeCtx.CreateContract(karma.Contract)
	contractContext := contractpb.WrapPluginContext(fakeCtx.WithAddress(karmaAddr))

	// Init the karma contract
	karmaContract := &karma.Karma{}
	require.NoError(t, karmaContract.Init(contractContext, &ktypes.KarmaInitRequest{
		Sources: sources,
	}))

	// This can also be done on init, but more concise this way
	require.NoError(t, karma.AddKarma(contractContext, origin, sourceStates))

	ctx := context.WithValue(state.Context(), loomAuth.ContextKeyOrigin, origin)

	tmx := GetKarmaMiddleWare(
		true,
		maxCallCount,
		sessionDuration,
		func(state loomchain.State) (contractpb.Context, error) {
			return contractContext, nil
		},
	)

	callKarma := userState.CallKarmaTotal

	for i := int64(1); i <= maxCallCount*2+callKarma.Value.Int64(); i++ {
		txSigned := mockSignedTx(t, uint64(i), types.TxID_CALL, vm.VMType_PLUGIN, contract)
		_, err := throttleMiddlewareHandler(tmx, state, txSigned, ctx)

		if i <= maxCallCount+callKarma.Value.Int64() {
			require.NoError(t, err)
		} else {
			require.Error(t, err, fmt.Sprintf("Out of calls for current session: %d out of %d, Try after sometime!", i, maxCallCount))
		}
	}
}

func mockSignedTx(t *testing.T, sequence uint64, id types.TxID, vmType vm.VMType, to loom.Address) auth.SignedTx {
	origBytes := []byte("origin")
	// TODO: wtf is this generating a new key every time, what's the point of the sequence number then?
	_, privKey, err := ed25519.GenerateKey(nil)
	require.Nil(t, err)

	var messageTx []byte

	switch id {
	case types.TxID_CALL:
		callTx, err := proto.Marshal(&vm.CallTx{
			VmType: vmType,
			Input:  origBytes,
		})
		require.NoError(t, err)

		messageTx, err = proto.Marshal(&vm.MessageTx{
			Data: callTx,
			To:   to.MarshalPB(),
		})
		require.NoError(t, err)

	case types.TxID_DEPLOY:
		deployTX, err := proto.Marshal(&vm.DeployTx{
			VmType: vmType,
			Code:   origBytes,
		})
		require.NoError(t, err)

		messageTx, err = proto.Marshal(&vm.MessageTx{
			Data: deployTX,
			To:   to.MarshalPB(),
		})
		require.NoError(t, err)

	case types.TxID_MIGRATION:
		migrationTx, err := proto.Marshal(&vm.MigrationTx{
			ID: 1,
		})
		require.NoError(t, err)

		messageTx, err = proto.Marshal(&vm.MessageTx{
			Data: migrationTx,
			To:   to.MarshalPB(),
		})
		require.NoError(t, err)

	case types.TxID_ETHEREUM:
		ethBytes, err := ethTxBytes(sequence, to, origBytes)
		require.NoError(t, err)
		messageTx, err = proto.Marshal(&vm.MessageTx{
			Data: ethBytes,
			To:   to.MarshalPB(),
		})
		require.NoError(t, err)

	default:
		require.FailNow(t, "invalid tx ID")
	}

	tx, err := proto.Marshal(&loomchain.Transaction{
		Id:   id,
		Data: messageTx,
	})
	require.NoError(t, err)
	nonceTx, err := proto.Marshal(&auth.NonceTx{
		Inner:    tx,
		Sequence: sequence,
	})
	require.Nil(t, err)

	signer := auth.NewEd25519Signer([]byte(privKey))
	signedTx := auth.SignTx(signer, nonceTx)
	signedTxBytes, err := proto.Marshal(signedTx)
	require.Nil(t, err)
	var txSigned auth.SignedTx
	err = proto.Unmarshal(signedTxBytes, &txSigned)
	require.Nil(t, err)

	require.Equal(t, len(txSigned.PublicKey), ed25519.PublicKeySize)
	require.Equal(t, len(txSigned.Signature), ed25519.SignatureSize)
	require.True(t, ed25519.Verify(txSigned.PublicKey, txSigned.Inner, txSigned.Signature))
	return txSigned
}

func ethTxBytes(sequence uint64, to loom.Address, data []byte) ([]byte, error) {
	bigZero := big.NewInt(0)
	var tx *etypes.Transaction
	if to.IsEmpty() {
		tx = etypes.NewContractCreation(
			sequence,
			big.NewInt(24),
			0,
			bigZero,
			data,
		)
	} else {
		tx = etypes.NewTransaction(
			sequence,
			common.BytesToAddress(to.Local),
			big.NewInt(11),
			0,
			bigZero,
			data,
		)
	}
	ethChainID, err := evmcompat.ToEthereumChainID("default")
	if err != nil {
		return nil, err
	}
	signer := etypes.NewEIP155Signer(ethChainID)
	ethKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	tx, err = etypes.SignTx(tx, signer, ethKey)
	if err != nil {
		return nil, err
	}
	return rlp.EncodeToBytes(&tx)
}
