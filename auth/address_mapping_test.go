// +build evm

package auth

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	amtypes "github.com/loomnetwork/go-loom/builtin/types/address_mapper"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	callId    = uint32(2)
	testChain = "testChain"
)

var (
	priKey1 = "PKAYW0doHy4RUQz9Hg8cpYYT4jpRH2AQAUSm6m0O2IvggzWbwX2CViNtD9f55kGssAOZG2MnsDU88QFYpTtwyg=="
	pubKey1 = "0x62666100f8988238d81831dc543D098572F283A1"

	addr1    = loom.MustParseAddress(testChain + ":" + pubKey1)
	origin   = loom.MustParseAddress(testChain + ":0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	contract = loom.MustParseAddress(testChain + ":0x9a1aC42a17AAD6Dbc6d21c162989d0f701074044")
)

func TestAddressMappingMiddleWare(t *testing.T) {
	log.Setup("debug", "file://-")
	log.Root.With("module", "throttle-middleware")

	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{ChainID: testChain}, nil)
	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	addresMapperAddr := fakeCtx.CreateContract(address_mapper.Contract)
	amCtx := contractpb.WrapPluginContext(fakeCtx.WithAddress(addresMapperAddr))

	ctx := context.WithValue(state.Context(), ContextKeyOrigin, origin)

	tmx := GetSignatureTxMiddleware(func(state loomchain.State) (contractpb.Context, error) { return amCtx, nil })

	// Normal loom transaction without address mapping
	txSigned := mockEd25519SignedTx(t, priKey1, Loom)
	_, err := throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)

	// Define supported chains by in middleware
	externalNetworks = map[string]ExternalNetworks{
		"": {
			Prefix:  testChain,
			Type:    Loom,
			Network: "1",
			Enabled: true,
		},
		EthChainId: {
			Prefix:  EthChainId,
			Type:    Eth,
			Network: "1",
			Enabled: true,
		},
	}

	// Init the contract
	am := address_mapper.AddressMapper{}
	require.NoError(t, am.Init(amCtx, &address_mapper.InitRequest{}))

	// generate eth key
	ethKey, err := crypto.GenerateKey()
	ethLocalAdr, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(ethKey.PublicKey).Hex())
	ethPublicAddr := loom.Address{ChainID: EthChainId, Local: ethLocalAdr}

	// tx using address mapping from eth account
	txSigned = mockEthSignedTx(t, ethKey)
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.Error(t, err)

	// set up address mapping between eth and loom accounts
	sig, err := address_mapper.SignIdentityMapping(addr1, ethPublicAddr, ethKey)
	mapping := amtypes.AddressMapperAddIdentityMappingRequest{
		From:      addr1.MarshalPB(),
		To:        ethPublicAddr.MarshalPB(),
		Signature: sig,
	}
	mapping = mapping
	require.NoError(t, am.AddIdentityMapping(amCtx, &mapping))

	// tx using address mapping from eth account
	txSigned = mockEthSignedTx(t, ethKey)
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)

}

func throttleMiddlewareHandler(ttm loomchain.TxMiddlewareFunc, state loomchain.State, signedTx []byte, ctx context.Context) (loomchain.TxHandlerResult, error) {
	return ttm.ProcessTx(state.WithContext(ctx), signedTx,
		func(state loomchain.State, txBytes []byte, isCheckTx bool) (res loomchain.TxHandlerResult, err error) {
			return loomchain.TxHandlerResult{}, err
		}, false,
	)
}

func mockEd25519SignedTx(t *testing.T, key, chainName string) []byte {
	ed25519PrivKey, err := base64.StdEncoding.DecodeString(key)
	require.NoError(t, err)
	signer := auth.NewSigner(auth.SignerTypeEd25519, ed25519PrivKey)
	signedTx := auth.SignTx(signer, mockNonceTx(t))
	signedTx.ChainName = chainName
	marshalledSignedTx, err := proto.Marshal(signedTx)
	require.NoError(t, err)
	return marshalledSignedTx
}

func mockEthSignedTx(t *testing.T, key *ecdsa.PrivateKey) []byte {
	nonceTx := mockNonceTx(t)

	ethLocalAdr, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(key.PublicKey).Hex())
	ethPublicAddr := loom.Address{ChainID: EthChainId, Local: ethLocalAdr}

	hash := crypto.Keccak256(ethPublicAddr.Bytes())
	signature, err := evmcompat.SoliditySign(hash, key)
	require.NoError(t, err)

	marshalledSignedTx, err := proto.Marshal(&auth.SignedTx{
		Inner:     nonceTx,
		Signature: signature,
		PublicKey: hash,
		ChainName: EthChainId,
	})
	require.NoError(t, err)
	return marshalledSignedTx
}

func mockNonceTx(t *testing.T) []byte {
	origBytes := []byte("origin")
	callTx, err := proto.Marshal(&vm.CallTx{
		VmType: vm.VMType_EVM,
		Input:  origBytes,
	})
	messageTx, err := proto.Marshal(&vm.MessageTx{
		Data: callTx,
		To:   contract.MarshalPB(),
	})
	require.NoError(t, err)
	tx, err := proto.Marshal(&loomchain.Transaction{
		Id:   callId,
		Data: messageTx,
	})
	require.Nil(t, err)
	nonceTx, err := proto.Marshal(&auth.NonceTx{
		Inner:    tx,
		Sequence: 1,
	})
	require.Nil(t, err)
	return nonceTx
}

/**/
