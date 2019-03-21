package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"golang.org/x/crypto/ed25519"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/store"
)

const (
	callId             = uint32(2)
	sequence           = uint64(4)
	defaultLoomChainId = "default"
)

var (
	ethPrivateKey = "fa6b7c0f1845e1260e8f1eee2ac11ae21238a06fb2634c40625b32f9022a0ab1"
	priKey1       = "PKAYW0doHy4RUQz9Hg8cpYYT4jpRH2AQAUSm6m0O2IvggzWbwX2CViNtD9f55kGssAOZG2MnsDU88QFYpTtwyg=="
	pubKey1       = "0x62666100f8988238d81831dc543D098572F283A1"

	addr1    = loom.MustParseAddress(defaultLoomChainId + ":" + pubKey1)
	origin   = loom.MustParseAddress(defaultLoomChainId + ":0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	contract = loom.MustParseAddress(defaultLoomChainId + ":0x9a1aC42a17AAD6Dbc6d21c162989d0f701074044")
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

	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{ChainID: "default"}, nil)
	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	addresMapperAddr := fakeCtx.CreateContract(address_mapper.Contract)
	amCtx := contractpb.WrapPluginContext(fakeCtx.WithAddress(addresMapperAddr))
	chains := map[string]ChainConfig{}

	chainConfigMiddleware := NewChainConfigMiddleware(chains, func(state loomchain.State) (contractpb.Context, error) { return amCtx, nil })
	_, err = chainConfigMiddleware.ProcessTx(state, signedTxBytes,
		func(state loomchain.State, txBytes []byte, isCheckTx bool) (loomchain.TxHandlerResult, error) {
			require.Equal(t, txBytes, origBytes)
			return loomchain.TxHandlerResult{}, nil
		}, false,
	)
	require.NoError(t, err)
}

func TestChainConfigMiddlewareMultipleChain(t *testing.T) {
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{ChainID: defaultLoomChainId}, nil)
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
	tmx := NewChainConfigMiddleware(
		chains,
		func(state loomchain.State) (contractpb.Context, error) { return amCtx, nil },
	)

	txSigned := mockEd25519SignedTx(t, priKey1)
	_, err := throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)
}

func mockEd25519SignedTx(t *testing.T, key string) []byte {
	ed25519PrivKey, err := base64.StdEncoding.DecodeString(key)
	require.NoError(t, err)
	signer := auth.NewSigner(auth.SignerTypeEd25519, ed25519PrivKey)

	addr := loom.Address{
		ChainID: defaultLoomChainId,
		Local:   loom.LocalAddressFromPublicKey(signer.PublicKey()),
	}
	nonceTx := mockNonceTx(t, addr, sequence)

	signedTx := auth.SignTx(signer, nonceTx)
	marshalledSignedTx, err := proto.Marshal(signedTx)
	require.NoError(t, err)
	return marshalledSignedTx
}

func mockSignedTx(t *testing.T, chainID string, signer auth.Signer) []byte {
	privateKey, err := crypto.UnmarshalPubkey(signer.PublicKey())
	require.NoError(t, err)
	ethLocalAdr, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(*privateKey).Hex())
	nonceTx := mockNonceTx(t, loom.Address{ChainID: chainID, Local: ethLocalAdr}, sequence)

	signedTx := auth.SignTx(signer, nonceTx)
	marshalledSignedTx, err := proto.Marshal(signedTx)
	require.NoError(t, err)
	return marshalledSignedTx
}

func mockNonceTx(t *testing.T, from loom.Address, sequence uint64) []byte {
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
		Sequence: sequence,
	})
	require.Nil(t, err)
	return nonceTx
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
			if origin.ChainID == caller.ChainID && caller.Compare(origin) != 0 {
				return res, fmt.Errorf("Origin doesn't match caller: - %v != %v", origin, caller)
			}
			return loomchain.TxHandlerResult{}, err
		}, false,
	)
}
