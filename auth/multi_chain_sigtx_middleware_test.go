// +build evm

package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	amtypes "github.com/loomnetwork/go-loom/builtin/types/address_mapper"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/vm"
	sha3 "github.com/miguelmota/go-solidity-sha3"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/loomnetwork/loomchain/features"
	"github.com/loomnetwork/loomchain/store"
)

const (
	sequence           = uint64(4)
	defaultLoomChainId = "default"
)

var (
	ethPrivateKey = "fa6b7c0f1845e1260e8f1eee2ac11ae21238a06fb2634c40625b32f9022a0ab1"
	priKey1       = "PKAYW0doHy4RUQz9Hg8cpYYT4jpRH2AQAUSm6m0O2IvggzWbwX2CViNtD9f55kGssAOZG2MnsDU88QFYpTtwyg=="
	pubKey1       = "0x62666100f8988238d81831dc543D098572F283A1"

	//tronPrivateKey = "65e48fcb00c88c0d4b9c3ee0bff77102dc72eda882f0891acb830e46f0f75e8b"

	addr1    = loom.MustParseAddress(defaultLoomChainId + ":" + pubKey1)
	origin   = loom.MustParseAddress(defaultLoomChainId + ":0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	contract = loom.MustParseAddress(defaultLoomChainId + ":0x9a1aC42a17AAD6Dbc6d21c162989d0f701074044")
)

func TestSigning(t *testing.T) {
	pk, err := crypto.GenerateKey()
	require.NoError(t, err)
	err = crypto.SaveECDSA("newpk", pk)
	require.NoError(t, err)

	privateKey, err := crypto.HexToECDSA(ethPrivateKey)
	require.NoError(t, err)
	publicKey := crypto.FromECDSAPub(&privateKey.PublicKey)
	require.NoError(t, err)
	to := contract
	nonce := uint64(7)

	// Encode
	nonceTx := []byte("nonceTx")
	ethLocalAdr, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(privateKey.PublicKey).Hex())
	require.NoError(t, err)
	hash := sha3.SoliditySHA3(
		sha3.Address(common.BytesToAddress(ethLocalAdr)),
		sha3.Address(common.BytesToAddress(to.Local)),
		sha3.Uint64(nonce),
		nonceTx,
	)

	signature, err := evmcompat.GenerateTypedSig(hash, privateKey, evmcompat.SignatureType_EIP712)
	require.NoError(t, err)

	tx := &auth.SignedTx{
		Inner:     nonceTx,
		Signature: signature,
		PublicKey: publicKey,
	}

	// Decode
	allowedSigTypes := []evmcompat.SignatureType{evmcompat.SignatureType_EIP712}
	recoverdAddr, err := evmcompat.RecoverAddressFromTypedSig(hash, tx.Signature, allowedSigTypes)
	require.NoError(t, err)
	require.True(t, bytes.Equal(recoverdAddr.Bytes(), ethLocalAdr))

	signatureNoRecoverID := signature[1 : len(tx.Signature)-1] // remove recovery ID
	require.True(t, crypto.VerifySignature(tx.PublicKey, hash, signatureNoRecoverID))

}

func TestTronSigning(t *testing.T) {
	privateKey, err := crypto.HexToECDSA(ethPrivateKey)
	require.NoError(t, err)
	publicKey := crypto.FromECDSAPub(&privateKey.PublicKey)
	require.NoError(t, err)
	to := contract
	nonce := uint64(7)

	// Encode
	nonceTx := []byte("nonceTx")
	foreignLocalAddr, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(privateKey.PublicKey).Hex())
	require.NoError(t, err)

	hash := sha3.SoliditySHA3(
		sha3.Address(common.BytesToAddress(foreignLocalAddr)),
		sha3.Address(common.BytesToAddress(to.Local)),
		sha3.Uint64(nonce),
		nonceTx,
	)
	prefixedHash := evmcompat.PrefixHeader(hash, evmcompat.SignatureType_TRON)

	signature, err := evmcompat.GenerateTypedSig(prefixedHash, privateKey, evmcompat.SignatureType_TRON)
	require.NoError(t, err)

	tx := &auth.SignedTx{
		Inner:     nonceTx,
		Signature: signature,
		PublicKey: publicKey,
	}

	// Decode
	allowedSigTypes := []evmcompat.SignatureType{evmcompat.SignatureType_TRON}
	recoverdAddr, err := evmcompat.RecoverAddressFromTypedSig(hash, tx.Signature, allowedSigTypes)
	require.NoError(t, err)
	require.True(t, bytes.Equal(recoverdAddr.Bytes(), foreignLocalAddr))

	signatureNoRecoverID := signature[1 : len(tx.Signature)-1] // remove recovery ID
	require.True(t, crypto.VerifySignature(tx.PublicKey, prefixedHash, signatureNoRecoverID))
}

func TestBinanceSigning(t *testing.T) {
	privateKey, err := crypto.HexToECDSA(ethPrivateKey)
	require.NoError(t, err)
	signer := auth.NewBinanceSigner(crypto.FromECDSA(privateKey))
	publicKey := signer.PublicKey()
	require.NoError(t, err)
	to := contract
	nonce := uint64(7)

	// Encode
	nonceTx := []byte("nonceTx")
	foreignLocalAddr, err := loom.LocalAddressFromHexString(evmcompat.BitcoinAddress(publicKey).Hex())
	require.NoError(t, err)
	hash := sha3.SoliditySHA3(
		sha3.Address(common.BytesToAddress(foreignLocalAddr)),
		sha3.Address(common.BytesToAddress(to.Local)),
		sha3.Uint64(nonce),
		nonceTx,
	)

	signature, err := evmcompat.GenerateTypedSig(hash, privateKey, evmcompat.SignatureType_BINANCE)
	require.NoError(t, err)

	tx := &auth.SignedTx{
		Inner:     nonceTx,
		Signature: signature,
		PublicKey: publicKey,
	}

	// Decode
	allowedSigTypes := []evmcompat.SignatureType{evmcompat.SignatureType_BINANCE}
	recoverdAddr, err := evmcompat.RecoverAddressFromTypedSig(hash, tx.Signature, allowedSigTypes)
	require.NoError(t, err)
	require.True(t, bytes.Equal(recoverdAddr.Bytes(), foreignLocalAddr))

	signatureNoRecoverID := signature[1 : len(tx.Signature)-1] // remove recovery ID
	require.True(t, crypto.VerifySignature(tx.PublicKey, hash, signatureNoRecoverID))
}

func TestEthAddressMappingVerification(t *testing.T) {
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{ChainID: defaultLoomChainId}, nil, nil)
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
	}
	tmx := NewMultiChainSignatureTxMiddleware(
		chains,
		func(state loomchain.State) (contractpb.StaticContext, error) { return amCtx, nil },
	)

	// Normal local transaction without address mapping
	txSigned := mockEd25519SignedTx(t, priKey1)
	_, err := throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)

	// Define supported chains in middleware

	// Init the contract
	am := address_mapper.AddressMapper{}
	require.NoError(t, am.Init(amCtx, &address_mapper.InitRequest{}))

	// Generate ETH key
	ethKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	ethLocalAdr, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(ethKey.PublicKey).Hex())
	require.NoError(t, err)
	ethPublicAddr := loom.Address{ChainID: "eth", Local: ethLocalAdr}

	// Transaction using address mapping from an ETH account. Gives error.
	txSigned = mockSignedTx(t, "eth", &auth.EthSigner66Byte{PrivateKey: ethKey})
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.Error(t, err)

	// Set up address mapping between ETH and local accounts
	sig, err := address_mapper.SignIdentityMapping(addr1, ethPublicAddr, ethKey, evmcompat.SignatureType_EIP712)
	require.NoError(t, err)
	mapping := amtypes.AddressMapperAddIdentityMappingRequest{
		From:      addr1.MarshalPB(),
		To:        ethPublicAddr.MarshalPB(),
		Signature: sig,
	}
	require.NoError(t, am.AddIdentityMapping(amCtx, &mapping))

	// Transaction using address mapping from an ETH account. No error this time as the mapped local account is found.
	txSigned = mockSignedTx(t, "eth", &auth.EthSigner66Byte{ethKey})
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)
}

func TestBinanceAddressMappingVerification(t *testing.T) {
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{ChainID: defaultLoomChainId}, nil, nil)
	state.SetFeature(features.AddressMapperVersion1_1, true)
	state.SetFeature(features.MultiChainSigTxMiddlewareVersion1_1, true)
	state.SetFeature(features.AuthSigTxFeaturePrefix+"binance", true)
	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	// FIXME: Having to set feature flags twice is pretty stupid... need to fix this state/ctx mess.
	fakeCtx.SetFeature(features.AddressMapperVersion1_1, true)
	state.SetFeature(features.MultiChainSigTxMiddlewareVersion1_1, true)
	fakeCtx.SetFeature(features.AuthSigTxFeaturePrefix+"binance", true)
	addresMapperAddr := fakeCtx.CreateContract(address_mapper.Contract)
	amCtx := contractpb.WrapPluginContext(fakeCtx.WithAddress(addresMapperAddr))

	ctx := context.WithValue(state.Context(), ContextKeyOrigin, origin)

	chains := map[string]ChainConfig{
		"default": {
			TxType:      LoomSignedTxType,
			AccountType: NativeAccountType,
		},
		"binance": {
			TxType:      BinanceSignedTxType,
			AccountType: MappedAccountType,
		},
	}
	tmx := NewMultiChainSignatureTxMiddleware(
		chains,
		func(state loomchain.State) (contractpb.StaticContext, error) { return amCtx, nil },
	)

	// Normal local transaction without address mapping
	txSigned := mockEd25519SignedTx(t, priKey1)
	_, err := throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)

	// Init the contract
	am := address_mapper.AddressMapper{}
	require.NoError(t, am.Init(amCtx, &address_mapper.InitRequest{}))

	// Generate a Binance key
	privKey, err := crypto.HexToECDSA(ethPrivateKey)
	require.NoError(t, err)
	signer := auth.NewBinanceSigner(crypto.FromECDSA(privKey))
	foreignLocalAddr, err := loom.LocalAddressFromHexString(evmcompat.BitcoinAddress(signer.PublicKey()).Hex())
	require.NoError(t, err)
	foreignPublicAddr := loom.Address{ChainID: "binance", Local: foreignLocalAddr}

	// Transaction using address mapping from a Binance account. Gives error.
	txSigned = mockBinanceSignedTx(t, "binance", signer)
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.Error(t, err)

	// Set up address mapping between Binance and local accounts
	sig, err := address_mapper.SignIdentityMapping(addr1, foreignPublicAddr, privKey, evmcompat.SignatureType_BINANCE)
	require.NoError(t, err)
	mapping := amtypes.AddressMapperAddIdentityMappingRequest{
		From:      addr1.MarshalPB(),
		To:        foreignPublicAddr.MarshalPB(),
		Signature: sig,
	}
	require.NoError(t, am.AddIdentityMapping(amCtx, &mapping))

	// Transaction using address mapping from a Binance account. No error this time as the mapped local account is found.
	txSigned = mockBinanceSignedTx(t, "binance", signer)
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)
}

func TestChainIdVerification(t *testing.T) {
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{ChainID: defaultLoomChainId}, nil, nil)
	state.SetFeature(features.AddressMapperVersion1_1, true)
	state.SetFeature(features.MultiChainSigTxMiddlewareVersion1_1, true)
	state.SetFeature(features.AuthSigTxFeaturePrefix+"tron", true)
	state.SetFeature(features.AuthSigTxFeaturePrefix+"binance", true)
	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	state.SetFeature(features.AddressMapperVersion1_1, true)
	state.SetFeature(features.MultiChainSigTxMiddlewareVersion1_1, true)
	state.SetFeature(features.AuthSigTxFeaturePrefix+"tron", true)
	state.SetFeature(features.AuthSigTxFeaturePrefix+"binance", true)
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
			AccountType: NativeAccountType,
		},
		"tron": {
			TxType:      TronSignedTxType,
			AccountType: NativeAccountType,
		},
		"binance": {
			TxType:      BinanceSignedTxType,
			AccountType: NativeAccountType,
		},
	}
	tmx := NewMultiChainSignatureTxMiddleware(
		chains,
		func(state loomchain.State) (contractpb.StaticContext, error) { return amCtx, nil },
	)

	// Normal local transaction without address mapping
	txSigned := mockEd25519SignedTx(t, priKey1)
	_, err := throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)

	// Define supported chains in middleware

	// Init the contract
	am := address_mapper.AddressMapper{}
	require.NoError(t, am.Init(amCtx, &address_mapper.InitRequest{}))

	// Generate ETH key
	ethKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	// Transaction signed with an Ethereum key, address mapping disabled, the caller address passed through
	// to contracts will be eth:xxxxxx, i.e. the ETH account is passed through without being mapped
	// to a local account.
	// Don't try this in production.
	txSigned = mockSignedTx(t, "eth", &auth.EthSigner66Byte{PrivateKey: ethKey})
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)

	// Transaction signed with a Tron key, address mapping disabled, the caller address passed through
	// to contracts will be tron:xxxxxx, i.e. the Tron account is passed through without being mapped
	// to a local account.
	// Don't try this in production.
	txSigned = mockSignedTx(t, "tron", &auth.TronSigner{PrivateKey: ethKey})
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)

	// Binance
	txSigned = mockBinanceSignedTx(t, "binance", auth.NewBinanceSigner(crypto.FromECDSA(ethKey)))
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
	pubKey, err := crypto.UnmarshalPubkey(signer.PublicKey())
	require.NoError(t, err)
	ethLocalAdr, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(*pubKey).Hex())
	require.NoError(t, err)
	nonceTx := mockNonceTx(t, loom.Address{ChainID: chainID, Local: ethLocalAdr}, sequence)

	signedTx := auth.SignTx(signer, nonceTx)
	marshalledSignedTx, err := proto.Marshal(signedTx)
	require.NoError(t, err)
	return marshalledSignedTx
}

func mockBinanceSignedTx(t *testing.T, chainID string, signer auth.Signer) []byte {
	foreignLocalAddr, err := loom.LocalAddressFromHexString(evmcompat.BitcoinAddress(signer.PublicKey()).Hex())
	require.NoError(t, err)
	nonceTx := mockNonceTx(t, loom.Address{ChainID: chainID, Local: foreignLocalAddr}, sequence)

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
	require.NoError(t, err)
	messageTx, err := proto.Marshal(&vm.MessageTx{
		Data: callTx,
		To:   contract.MarshalPB(),
		From: from.MarshalPB(),
	})
	require.NoError(t, err)
	tx, err := proto.Marshal(&loomchain.Transaction{
		Id:   uint32(types.TxID_CALL),
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
