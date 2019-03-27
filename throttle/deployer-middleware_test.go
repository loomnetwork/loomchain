package throttle

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	dwtypes "github.com/loomnetwork/go-loom/builtin/types/deployer_whitelist"
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	loomAuth "github.com/loomnetwork/loomchain/auth"
	dw "github.com/loomnetwork/loomchain/builtin/plugins/deployer_whitelist"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"
)

var (
	owner = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	guest = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

type contextFactory func(state loomchain.State) (contractpb.Context, error)

func TestDeployerWhitelistMiddleware(t *testing.T) {
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{}, nil)
	state.SetFeature(dwFeature, true)

	txSignedPlugin := mockSignedTxWithoutNounce(t, uint64(1), deployId, vm.VMType_PLUGIN, contract)
	txSignedEVM := mockSignedTxWithoutNounce(t, uint64(2), deployId, vm.VMType_EVM, contract)
	//init contract
	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	dwAddr := fakeCtx.CreateContract(dw.Contract)
	contractContext := contractpb.WrapPluginContext(fakeCtx.WithAddress(dwAddr))

	dwContract := &dw.DeployerWhitelist{}
	require.NoError(t, dwContract.Init(contractContext, &dwtypes.InitRequest{
		Owner: owner.MarshalPB(),
	}))

	guestCtx := context.WithValue(state.Context(), loomAuth.ContextKeyOrigin, guest)
	ownerCtx := context.WithValue(state.Context(), loomAuth.ContextKeyOrigin, owner)

	dwMiddleware, err := NewDeployerWhitelistMiddleware(
		func(state loomchain.State) (contractpb.Context, error) {
			return contractContext, nil
		},
	)
	require.NoError(t, err)

	// unauthorized deployer
	_, err = throttleMiddlewareHandler(dwMiddleware, state, txSignedPlugin, guestCtx)
	require.Equal(t, ErrNotAuthorized, err)
	// unauthorized deployer
	_, err = throttleMiddlewareHandler(dwMiddleware, state, txSignedEVM, guestCtx)
	require.Equal(t, ErrNotAuthorized, err)

	// authorized deployer
	_, err = throttleMiddlewareHandler(dwMiddleware, state, txSignedPlugin, ownerCtx)
	require.NoError(t, err)
	_, err = throttleMiddlewareHandler(dwMiddleware, state, txSignedEVM, ownerCtx)
	require.NoError(t, err)
}

func mockSignedTxWithoutNounce(t *testing.T, sequence uint64, id uint32, vmType vm.VMType, to loom.Address) auth.SignedTx {
	origBytes := []byte("origin")
	// TODO: wtf is this generating a new key every time, what's the point of the sequence number then?
	_, privKey, err := ed25519.GenerateKey(nil)
	require.Nil(t, err)

	var messageTx []byte
	require.True(t, id == callId || id == deployId)
	if id == callId {
		callTx, err := proto.Marshal(&vm.CallTx{
			VmType: vmType,
			Input:  origBytes,
		})
		require.NoError(t, err)

		messageTx, err = proto.Marshal(&vm.MessageTx{
			Data: callTx,
			To:   to.MarshalPB(),
		})
	} else {
		deployTX, err := proto.Marshal(&vm.DeployTx{
			VmType: vmType,
			Code:   origBytes,
		})
		require.NoError(t, err)

		messageTx, err = proto.Marshal(&vm.MessageTx{
			Data: deployTX,
			To:   to.MarshalPB(),
		})
	}

	tx, err := proto.Marshal(&loomchain.Transaction{
		Id:   id,
		Data: messageTx,
	})

	signer := auth.NewEd25519Signer([]byte(privKey))
	signedTx := auth.SignTx(signer, tx)
	signedTxBytes, err := proto.Marshal(signedTx)
	var txSigned auth.SignedTx
	err = proto.Unmarshal(signedTxBytes, &txSigned)
	require.Nil(t, err)

	require.Equal(t, len(txSigned.PublicKey), ed25519.PublicKeySize)
	require.Equal(t, len(txSigned.Signature), ed25519.SignatureSize)
	require.True(t, ed25519.Verify(txSigned.PublicKey, txSigned.Inner, txSigned.Signature))
	return txSigned
}
