// +build evm

package auth

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	amtypes "github.com/loomnetwork/go-loom/builtin/types/address_mapper"
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/store"
	"github.com/pkg/errors"
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

func TestAddressMappingVerification(t *testing.T) {
	//t.Skip()
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{ChainID: testChain}, nil)
	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	addresMapperAddr := fakeCtx.CreateContract(address_mapper.Contract)
	amCtx := contractpb.WrapPluginContext(fakeCtx.WithAddress(addresMapperAddr))

	ctx := context.WithValue(state.Context(), ContextKeyOrigin, origin)

	externalNetworks := map[string]ExternalNetworks{
		Loom: {
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
	tmx := GetSignatureTxMiddleware(externalNetworks, func(state loomchain.State) (contractpb.Context, error) { return amCtx, nil })

	// Normal loom transaction without address mapping
	txSigned := mockEd25519SignedTx(t, priKey1, Loom)
	_, err := throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)

	// Define supported chains by in middleware

	// Init the contract
	am := address_mapper.AddressMapper{}
	require.NoError(t, am.Init(amCtx, &address_mapper.InitRequest{}))

	// generate eth key
	ethKey, err := crypto.GenerateKey()

	ethLocalAdr, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(ethKey.PublicKey).Hex())
	ethPublicAddr := loom.Address{ChainID: EthChainId, Local: ethLocalAdr}

	// tx using address mapping from eth account. Gives error.
	txSigned = mockEthSignedTx(t, ethKey, EthChainId)
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

	// tx using address mapping from eth account. No error this time as mapped loom account is found.
	txSigned = mockEthSignedTx(t, ethKey, EthChainId)
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)
}

func TestChainIdVerification(t *testing.T) {
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{ChainID: testChain}, nil)
	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	addresMapperAddr := fakeCtx.CreateContract(address_mapper.Contract)
	amCtx := contractpb.WrapPluginContext(fakeCtx.WithAddress(addresMapperAddr))

	ctx := context.WithValue(state.Context(), ContextKeyOrigin, origin)

	externalNetworks := map[string]ExternalNetworks{
		Loom: {
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
	tmx := GetSignatureTxMiddleware(externalNetworks, func(state loomchain.State) (contractpb.Context, error) { return amCtx, nil })

	// Normal loom transaction without address mapping
	txSigned := mockEd25519SignedTx(t, priKey1, "")
	_, err := throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)

	// Define supported chains by in middleware

	// Init the contract
	am := address_mapper.AddressMapper{}
	require.NoError(t, am.Init(amCtx, &address_mapper.InitRequest{}))

	// generate eth key
	ethKey, err := crypto.GenerateKey()

	// tx using address mapping from eth account. No error as using chainId mapping.
	txSigned = mockEthSignedTx(t, ethKey, "")
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)
}

func throttleMiddlewareHandler(ttm loomchain.TxMiddlewareFunc, state loomchain.State, signedTx []byte, ctx context.Context) (loomchain.TxHandlerResult, error) {
	return ttm.ProcessTx(state.WithContext(ctx), signedTx,
		func(state loomchain.State, txBytes []byte, isCheckTx bool) (res loomchain.TxHandlerResult, err error) {

			var nonceTx NonceTx
			if err := proto.Unmarshal(txBytes, &nonceTx); err != nil {
				return res, errors.Wrap(err, "throttle: unwrap nonce Tx")
			}

			var tx loomchain.Transaction
			if err := proto.Unmarshal(nonceTx.Inner, &tx); err != nil {
				return res, errors.New("throttle: unmarshal tx")
			}
			var msg vm.MessageTx
			if err := proto.Unmarshal(tx.Data, &msg); err != nil {
				return res, errors.Wrapf(err, "unmarshal message tx %v", tx.Data)
			}

			origin := Origin(state.Context())
			caller := loom.UnmarshalAddressPB(msg.From)
			if caller.Compare(origin) != 0 {
				return res, fmt.Errorf("Origin doesn't match caller: - %v != %v", origin, caller)
			}

			return loomchain.TxHandlerResult{}, err
		}, false,
	)
}

func mockEd25519SignedTx(t *testing.T, key, chainName string) []byte {
	ed25519PrivKey, err := base64.StdEncoding.DecodeString(key)
	require.NoError(t, err)
	signer := auth.NewSigner(auth.SignerTypeEd25519, ed25519PrivKey)

	nonceTx := mockNonceTx(t, loom.Address{ChainID: DefaultLoomChainId, Local: signer.PublicKey()})
	signedTx := auth.SignTx(signer, nonceTx)
	signedTx.ChainName = chainName
	marshalledSignedTx, err := proto.Marshal(signedTx)
	require.NoError(t, err)
	return marshalledSignedTx
}

func mockEthSignedTx(t *testing.T, key *ecdsa.PrivateKey, chainName string) []byte {
	ethLocalAdr, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(key.PublicKey).Hex())
	ethPublicAddr := loom.Address{ChainID: EthChainId, Local: ethLocalAdr}
	nonceTx := mockNonceTx(t, ethPublicAddr)

	singer := &auth.EthSigner66Byte{key}
	signedTx := auth.SignTx(singer, nonceTx)
	signedTx.ChainName = chainName
	marshalledSignedTx, err := proto.Marshal(signedTx)
	require.NoError(t, err)
	return marshalledSignedTx

}

func mockNonceTx(t *testing.T, from loom.Address) []byte {
	origBytes := []byte("origin")
	callTx, err := proto.Marshal(&vm.CallTx{
		VmType: vm.VMType_EVM,
		Input:  origBytes,
	})
	messageTx, err := proto.Marshal(&vm.MessageTx{
		Data: callTx,
		To:   contract.MarshalPB(),
		From: from.MarshalPB(),
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
