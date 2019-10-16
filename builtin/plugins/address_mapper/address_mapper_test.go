// +build evm

package address_mapper

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain/features"
	ssha "github.com/miguelmota/go-solidity-sha3"
	"github.com/stretchr/testify/suite"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	addr3 = loom.MustParseAddress("chain:0x3bA260874e6Ada53d4e0010fdE38cf2CD072A1be")
	addr4 = loom.MustParseAddress("chain:0x5BE813e5AA5ea26930edfc84ef12Cc164a652327")

	sigType = evmcompat.SignatureType_EIP712
)

type AddressMapperTestSuite struct {
	suite.Suite
	validEthKey     *ecdsa.PrivateKey
	validEthKey2    *ecdsa.PrivateKey
	invalidEthKey   *ecdsa.PrivateKey
	validEthAddr    loom.Address
	validEthAddr2   loom.Address
	invalidEthAddr  loom.Address
	validDAppAddr   loom.Address
	invalidDAppAddr loom.Address

	validTronKey   *ecdsa.PrivateKey
	validTronAddr  loom.Address
	validDAppAddr2 loom.Address

	validBinanceKey   *ecdsa.PrivateKey
	validBinanceAddr  loom.Address
	validBinanceKey2  *ecdsa.PrivateKey
	validBinanceAddr2 loom.Address
	validDAppAddr3    loom.Address
}

func (s *AddressMapperTestSuite) SetupTest() {
	r := s.Require()
	var err error
	s.validEthKey, err = crypto.GenerateKey()
	r.NoError(err)
	s.invalidEthKey, err = crypto.GenerateKey()
	r.NoError(err)
	ethLocalAddr, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(s.validEthKey.PublicKey).Hex())
	r.NoError(err)
	s.invalidEthAddr = loom.Address{ChainID: "", Local: ethLocalAddr}
	s.validEthAddr = loom.Address{ChainID: "eth", Local: ethLocalAddr}
	s.invalidDAppAddr = loom.Address{ChainID: "", Local: addr1.Local}
	s.validDAppAddr = loom.Address{ChainID: "chain", Local: addr1.Local}

	s.validTronKey, err = crypto.GenerateKey()
	r.NoError(err)
	tronLocalAddr, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(s.validTronKey.PublicKey).Hex())
	s.validTronAddr = loom.Address{ChainID: "tron", Local: tronLocalAddr}
	s.validDAppAddr2 = loom.Address{ChainID: "chain", Local: addr3.Local}

	s.validBinanceKey, err = crypto.GenerateKey()
	r.NoError(err)
	pubkey := secp256k1.CompressPubkey(s.validBinanceKey.X, s.validBinanceKey.Y)
	binanceLocalAddr, err := loom.LocalAddressFromHexString(evmcompat.BitcoinAddress(pubkey).Hex())
	s.validBinanceAddr = loom.Address{ChainID: "binance", Local: binanceLocalAddr}
	s.validDAppAddr3 = loom.Address{ChainID: "chain", Local: addr4.Local}

	s.validEthKey2, err = crypto.GenerateKey()
	r.NoError(err)
	ethLocalAddr2, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(s.validEthKey2.PublicKey).Hex())
	r.NoError(err)
	s.validEthAddr2 = loom.Address{ChainID: "eth", Local: ethLocalAddr2}

	s.validBinanceKey2, err = crypto.GenerateKey()
	r.NoError(err)
	pubkey = secp256k1.CompressPubkey(s.validBinanceKey2.X, s.validBinanceKey2.Y)
	binanceLocalAddr, err = loom.LocalAddressFromHexString(evmcompat.BitcoinAddress(pubkey).Hex())
	s.validBinanceAddr2 = loom.Address{ChainID: "binance", Local: binanceLocalAddr}

	fmt.Printf("EthAddr: %v, DAppAddr: %v\n", s.validEthAddr, s.validDAppAddr)
}

func TestAddressMapperTestSuite(t *testing.T) {
	suite.Run(t, new(AddressMapperTestSuite))
}

func (s *AddressMapperTestSuite) TestAddressMapperHasIdentityMapping() {
	r := s.Require()
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(s.validDAppAddr /*caller*/, loom.RootAddress("chain") /*contract*/),
	)

	amContract := &AddressMapper{}
	r.NoError(amContract.Init(ctx, &InitRequest{}))

	hasMappingResponse, err := amContract.HasMapping(ctx, &HasMappingRequest{
		From: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(false, hasMappingResponse.HasMapping)

	sig, err := SignIdentityMapping(s.validEthAddr, s.validDAppAddr, s.validEthKey, sigType)
	r.NoError(err)
	r.NoError(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validEthAddr.MarshalPB(),
		To:        s.validDAppAddr.MarshalPB(),
		Signature: sig,
	}))

	hasMappingResponse, err = amContract.HasMapping(ctx, &HasMappingRequest{
		From: s.validEthAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(true, hasMappingResponse.HasMapping)

	hasMappingResponse, err = amContract.HasMapping(ctx, &HasMappingRequest{
		From: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(true, hasMappingResponse.HasMapping)

	// r.NoError(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
	// 	From:      s.validEthAddr.MarshalPB(),
	// 	To:        s.validDAppAddr.MarshalPB(),
	// 	Signature: sig,
	// }))
}

func (s *AddressMapperTestSuite) TestAddressMapperAddSameIdentityMapping() {
	r := s.Require()
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(s.validDAppAddr /*caller*/, loom.RootAddress("chain") /*contract*/),
	)

	amContract := &AddressMapper{}
	r.NoError(amContract.Init(ctx, &InitRequest{}))

	sig, err := SignIdentityMapping(s.validEthAddr, s.validDAppAddr, s.validEthKey, sigType)
	r.NoError(err)
	r.NoError(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validEthAddr.MarshalPB(),
		To:        s.validDAppAddr.MarshalPB(),
		Signature: sig,
	}))

	r.EqualError(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validEthAddr.MarshalPB(),
		To:        s.validDAppAddr.MarshalPB(),
		Signature: sig,
	}), ErrAlreadyRegistered.Error(), "should error if either identity is already registered")

	invertedSig, err := SignIdentityMapping(s.validDAppAddr, s.validEthAddr, s.validEthKey, sigType)
	r.EqualError(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validDAppAddr.MarshalPB(),
		To:        s.validEthAddr.MarshalPB(),
		Signature: invertedSig,
	}), ErrAlreadyRegistered.Error(), "should error if either identity is already registered")
}

func (s *AddressMapperTestSuite) TestAddressMapperAddNewIdentityMapping() {
	r := s.Require()
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(s.validDAppAddr /*caller*/, loom.RootAddress("chain") /*contract*/),
	)

	amContract := &AddressMapper{}
	r.NoError(amContract.Init(ctx, &InitRequest{}))

	// Eth
	sig, err := SignIdentityMapping(s.validEthAddr, s.validDAppAddr, s.validEthKey, sigType)
	r.NoError(err)
	r.NoError(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validEthAddr.MarshalPB(),
		To:        s.validDAppAddr.MarshalPB(),
		Signature: sig,
	}))

	resp, err := amContract.GetMapping(ctx, &GetMappingRequest{
		From: s.validEthAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(s.validEthAddr.MarshalPB(), resp.From)
	s.Equal(s.validDAppAddr.MarshalPB(), resp.To)

	resp, err = amContract.GetMapping(ctx, &GetMappingRequest{
		From: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(s.validDAppAddr.MarshalPB(), resp.From)
	s.Equal(s.validEthAddr.MarshalPB(), resp.To)

	// Tron
	ctx = contract.WrapPluginContext(
		plugin.CreateFakeContext(s.validDAppAddr2 /*caller*/, loom.RootAddress("chain") /*contract*/),
	)
	sig, err = SignIdentityMapping(s.validTronAddr, s.validDAppAddr2, s.validTronKey, evmcompat.SignatureType_TRON)
	r.NoError(err)
	r.NoError(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validTronAddr.MarshalPB(),
		To:        s.validDAppAddr2.MarshalPB(),
		Signature: sig,
	}))

	resp, err = amContract.GetMapping(ctx, &GetMappingRequest{
		From: s.validTronAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(s.validTronAddr.MarshalPB(), resp.From)
	s.Equal(s.validDAppAddr2.MarshalPB(), resp.To)

	resp, err = amContract.GetMapping(ctx, &GetMappingRequest{
		From: s.validDAppAddr2.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(s.validDAppAddr2.MarshalPB(), resp.From)
	s.Equal(s.validTronAddr.MarshalPB(), resp.To)

	// Binance
	fakeCtx := plugin.CreateFakeContext(s.validDAppAddr3 /*caller*/, loom.RootAddress("chain") /*contract*/)
	fakeCtx.SetFeature(features.AddressMapperVersion1_1, true)
	ctx = contract.WrapPluginContext(fakeCtx)
	sig, err = SignIdentityMapping(s.validBinanceAddr, s.validDAppAddr3, s.validBinanceKey, evmcompat.SignatureType_BINANCE)
	r.NoError(err)
	r.NoError(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validBinanceAddr.MarshalPB(),
		To:        s.validDAppAddr3.MarshalPB(),
		Signature: sig,
	}))

	resp, err = amContract.GetMapping(ctx, &GetMappingRequest{
		From: s.validBinanceAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(s.validBinanceAddr.MarshalPB(), resp.From)
	s.Equal(s.validDAppAddr3.MarshalPB(), resp.To)

	resp, err = amContract.GetMapping(ctx, &GetMappingRequest{
		From: s.validDAppAddr3.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(s.validDAppAddr3.MarshalPB(), resp.From)
	s.Equal(s.validBinanceAddr.MarshalPB(), resp.To)
}

func (s *AddressMapperTestSuite) TestListMapping() {
	r := s.Require()
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(s.validDAppAddr /*caller*/, loom.RootAddress("chain") /*contract*/),
	)

	amContract := &AddressMapper{}
	r.NoError(amContract.Init(ctx, &InitRequest{}))

	sig, err := SignIdentityMapping(s.validEthAddr, s.validDAppAddr, s.validEthKey, sigType)
	r.NoError(err)
	r.NoError(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validEthAddr.MarshalPB(),
		To:        s.validDAppAddr.MarshalPB(),
		Signature: sig,
	}))

	resp, err := amContract.ListMapping(ctx, &ListMappingRequest{})
	r.NoError(err)
	s.Equal(1, len(resp.Mappings))
}

// Same as the other test case but the from/to inverted when adding the mapping,
// since the mapping is bi-directional the end result should be identical to the first test case.
func (s *AddressMapperTestSuite) TestAddressMapperAddNewInvertedIdentityMapping() {
	r := s.Require()
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(s.validDAppAddr /*caller*/, loom.RootAddress("chain") /*contract*/),
	)

	amContract := &AddressMapper{}
	r.NoError(amContract.Init(ctx, &InitRequest{}))

	sig, err := SignIdentityMapping(s.validDAppAddr, s.validEthAddr, s.validEthKey, sigType)
	r.NoError(err)
	r.NoError(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validDAppAddr.MarshalPB(),
		To:        s.validEthAddr.MarshalPB(),
		Signature: sig,
	}))

	resp, err := amContract.GetMapping(ctx, &GetMappingRequest{
		From: s.validEthAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(s.validEthAddr.MarshalPB(), resp.From)
	s.Equal(s.validDAppAddr.MarshalPB(), resp.To)

	resp, err = amContract.GetMapping(ctx, &GetMappingRequest{
		From: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(s.validDAppAddr.MarshalPB(), resp.From)
	s.Equal(s.validEthAddr.MarshalPB(), resp.To)
}

func (s *AddressMapperTestSuite) TestAddressMapperAddNewInvalidIdentityMapping() {
	r := s.Require()
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(s.validDAppAddr /*caller*/, loom.RootAddress("chain") /*contract*/))

	amContract := &AddressMapper{}
	r.NoError(amContract.Init(ctx, &InitRequest{}))

	r.Error(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From: s.validEthAddr.MarshalPB(),
		To:   s.validDAppAddr.MarshalPB(),
	}), "Should error if not signature provided")

	sig, err := SignIdentityMapping(s.validEthAddr, s.validDAppAddr, s.invalidEthKey, sigType)
	r.NoError(err)
	r.Error(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validEthAddr.MarshalPB(),
		To:        s.validDAppAddr.MarshalPB(),
		Signature: sig,
	}), "Should error if signature doesn't match foreign chain address")

	sig, err = SignIdentityMapping(s.invalidEthAddr, s.validDAppAddr, s.validEthKey, sigType)
	r.NoError(err)
	r.Error(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.invalidEthAddr.MarshalPB(),
		To:        s.validDAppAddr.MarshalPB(),
		Signature: sig,
	}), "Should error if from address doesn't have a Chain ID")

	sig, err = SignIdentityMapping(s.validEthAddr, s.invalidDAppAddr, s.validEthKey, sigType)
	r.NoError(err)
	r.Error(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validEthAddr.MarshalPB(),
		To:        s.invalidDAppAddr.MarshalPB(),
		Signature: sig,
	}), "Should error if to address doesn't have a Chain ID")

	sig, err = SignIdentityMapping(s.invalidEthAddr, s.invalidDAppAddr, s.validEthKey, sigType)
	r.Error(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.invalidEthAddr.MarshalPB(),
		To:        s.invalidDAppAddr.MarshalPB(),
		Signature: sig,
	}), "Should error if both addresses don't have a Chain ID")

	sig, err = SignIdentityMapping(s.validEthAddr, addr2, s.validEthKey, sigType)
	r.NoError(err)
	r.Error(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validEthAddr.MarshalPB(),
		To:        addr2.MarshalPB(),
		Signature: sig,
	}), "Should error if local chain address doesn't match caller address")
}

func (s *AddressMapperTestSuite) TestGethSigRecovery() {
	r := s.Require()
	// hash was generated by
	// const abi = require('ethereumjs-abi')
	// abi.soliditySHA3(
	//   ['address','address'],
	//   ['0xb16a379ec18d4093666f8f38b11a3071c920207d', '0xfa4c7920accfd66b86f5fd0e69682a79f762d49e'])
	hash, err := hex.DecodeString("e3d1790efd5ae545ed72c97a5c39f5fd489874978896677c3a7c5713219cd1f6")
	r.NoError(err)
	// this is how Address Mapper recreates the hash
	hash2 := ssha.SoliditySHA3(
		ssha.Address("b16a379ec18d4093666f8f38b11a3071c920207d"),
		ssha.Address("fa4c7920accfd66b86f5fd0e69682a79f762d49e"),
	)
	r.Equal(hash, hash2)

	// sig generated by signing the hash with web3.eth.sign()
	sig, err := hex.DecodeString("01e9e091b55ac2b172856c7a963e83503370dbc8cae4ec66ee4c9914588a56b3ce77f21e6686738ff5bbab59ba8817345d4f587b09aea2dd2ebd87b1315e8ad6661c")
	r.NoError(err)
	// prefix hash same way Geth does (https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_sign)
	hash = ssha.SoliditySHA3(
		ssha.String("\x19Ethereum Signed Message:\n32"),
		ssha.Bytes32(hash),
	)
	addr, err := evmcompat.SolidityRecover(hash, sig[1:])
	r.NoError(err)
	r.Equal(common.HexToAddress("0xf17f52151ebef6c7334fad080c5704d77216b732").Hex(), addr.Hex())
}

func (s *AddressMapperTestSuite) TestTronSigRecovery() {
	r := s.Require()
	// hash was generated by
	// const abi = require('ethereumjs-abi')
	// abi.soliditySHA3(
	//   ['address','address'],
	//   ['0xb16a379ec18d4093666f8f38b11a3071c920207d', '0xfa4c7920accfd66b86f5fd0e69682a79f762d49e'])
	hash, err := hex.DecodeString("e3d1790efd5ae545ed72c97a5c39f5fd489874978896677c3a7c5713219cd1f6")
	r.NoError(err)
	// this is how Address Mapper recreates the hash
	hash2 := ssha.SoliditySHA3(
		ssha.Address("b16a379ec18d4093666f8f38b11a3071c920207d"),
		ssha.Address("fa4c7920accfd66b86f5fd0e69682a79f762d49e"),
	)
	r.Equal(hash, hash2)

	// sig generated by signing the hash with
	sig, err := hex.DecodeString("03b78c2b9b1df9b8a2472343d810421a0593ed75e5960a37b91fdc5ac47648cded2c276c00d069f1a74686818bf5f8b1ad317b712cb39a9d813c0ecc43266a95231c")
	r.NoError(err)
	// prefix hash same way tronweb does https://github.com/TRON-US/tronweb/blob/027817d54fca8a4189f4f25aca291cc9415f6e43/src/lib/trx.js#L706
	hash = ssha.SoliditySHA3(
		ssha.String("\x19TRON Signed Message:\n32"),
		ssha.Bytes32(hash),
	)
	addr, err := evmcompat.SolidityRecover(hash, sig[1:])
	r.NoError(err)
	r.Equal(common.HexToAddress("0x96fd14f2c0da10916b972ef60c9f07a20ee7f99e").Hex(), addr.Hex())
}

func (s *AddressMapperTestSuite) TestBinanceSigRecovery() {
	r := s.Require()
	hash, err := hex.DecodeString("b838682b8bb99b475629d0ea622288cf58feec7f72c8d80bbe8b927c4f73df26")
	r.NoError(err)
	// generate hash from message using SHA256(address, address)
	hash2 := evmcompat.GenSHA256(
		ssha.Address("b16a379ec18d4093666f8f38b11a3071c920207d"),
		ssha.Address("fa4c7920accfd66b86f5fd0e69682a79f762d49e"),
	)
	r.Equal(hash, hash2)

	// sig generated by signing the message
	sig, err := hex.DecodeString("047ef4f8a3dce68e0f79956b123bb808f3b95fe731cc2490bd396b69e1f8d28967453c46204c8b29aa0f9555d2f8972abcdc14d99773e3d8f1a6cbe0fdf73d7fc31c")
	r.NoError(err)

	addr, err := evmcompat.BitcoinRecover(hash2, sig[1:])
	r.NoError(err)
	r.Equal(common.HexToAddress("0x131cD1A71cBc107b773c1763e7c9E11b26548F0c").Hex(), addr.Hex())
}

func (s *AddressMapperTestSuite) TestMultiChainAddressMapperHasIdentityMapping() {
	r := s.Require()
	fakeCtx := plugin.CreateFakeContext(s.validDAppAddr /*caller*/, loom.RootAddress("chain") /*contract*/)
	fakeCtx.SetFeature(features.AddressMapperVersion1_1, true)
	fakeCtx.SetFeature(features.AddressMapperVersion1_2, true)
	ctx := contract.WrapPluginContext(fakeCtx)

	amContract := &AddressMapper{}
	r.NoError(amContract.Init(ctx, &InitRequest{}))
	// check if validDAppAddr is not already mapped
	hasMappingResponse, err := amContract.HasMapping(ctx, &HasMappingRequest{
		From: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(false, hasMappingResponse.HasMapping)

	//Mapp eth --> dapp

	nonce, err := amContract.GetNonce(ctx, &GetNonceRequest{
		Address: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(uint64(0), nonce.Nonce)

	sig, err := MultiChainSignIdentityMapping(s.validEthAddr, s.validDAppAddr, s.validEthKey, sigType, nonce.Nonce)
	r.NoError(err)
	r.NoError(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validEthAddr.MarshalPB(),
		To:        s.validDAppAddr.MarshalPB(),
		Signature: sig,
	}))

	// check eth must mapped
	hasMappingResponse, err = amContract.HasMapping(ctx, &HasMappingRequest{
		From: s.validEthAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(true, hasMappingResponse.HasMapping)

	// check dapp must mapped
	hasMappingResponse, err = amContract.HasMapping(ctx, &HasMappingRequest{
		From: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(true, hasMappingResponse.HasMapping)

	// Check if GetMapping Compatible ?
	getMapping, err := amContract.GetMapping(ctx, &GetMappingRequest{
		From: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(s.validEthAddr.Local.String(), getMapping.To.Local.String())
	s.Equal(s.validEthAddr.ChainID, getMapping.To.ChainId)

	getMapping, err = amContract.GetMapping(ctx, &GetMappingRequest{
		From: s.validEthAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(s.validDAppAddr.Local.String(), getMapping.To.Local.String())
	s.Equal(s.validDAppAddr.ChainID, getMapping.To.ChainId)

	getMultiChainResp, err := amContract.GetMultiChainMapping(ctx, &GetMultiChainMappingRequest{
		From: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(s.validEthAddr.Local.String(), getMultiChainResp.Mappings[0].To.Local.String())
	s.Equal(s.validEthAddr.ChainID, getMultiChainResp.Mappings[0].To.ChainId)

	getMultiChainResp, err = amContract.GetMultiChainMapping(ctx, &GetMultiChainMappingRequest{
		From: s.validEthAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(s.validDAppAddr.Local.String(), getMultiChainResp.Mappings[0].To.Local.String())
	s.Equal(s.validDAppAddr.ChainID, getMultiChainResp.Mappings[0].To.ChainId)

	// binance must not be mapped before
	hasMappingResponse, err = amContract.HasMapping(ctx, &HasMappingRequest{
		From: s.validBinanceAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(false, hasMappingResponse.HasMapping)

	// map binance --> dapp
	nonce, err = amContract.GetNonce(ctx, &GetNonceRequest{
		Address: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(uint64(1), nonce.Nonce)

	sig, err = MultiChainSignIdentityMapping(s.validBinanceAddr, s.validDAppAddr, s.validBinanceKey, evmcompat.SignatureType_BINANCE, nonce.Nonce)
	r.NoError(err)
	r.NoError(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validBinanceAddr.MarshalPB(),
		To:        s.validDAppAddr.MarshalPB(),
		Signature: sig,
	}))

	hasMappingResponse, err = amContract.HasMapping(ctx, &HasMappingRequest{
		From: s.validBinanceAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(true, hasMappingResponse.HasMapping)

	hasMappingResponse, err = amContract.HasMapping(ctx, &HasMappingRequest{
		From: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(true, hasMappingResponse.HasMapping)

	getMultiChainResp, err = amContract.GetMultiChainMapping(ctx, &GetMultiChainMappingRequest{
		From: s.validBinanceAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(s.validDAppAddr.Local.String(), getMultiChainResp.Mappings[0].To.Local.String())
	s.Equal(s.validDAppAddr.ChainID, getMultiChainResp.Mappings[0].To.ChainId)

	dappMultichainMapping, err := amContract.GetMultiChainMapping(ctx, &GetMultiChainMappingRequest{
		From: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	r.Len(dappMultichainMapping.Mappings, 2)

	// tronAddr must not be mapped before
	hasMappingResponse, err = amContract.HasMapping(ctx, &HasMappingRequest{
		From: s.validTronAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(false, hasMappingResponse.HasMapping)
	// map tron --> dapp
	nonce, err = amContract.GetNonce(ctx, &GetNonceRequest{
		Address: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(uint64(2), nonce.Nonce)
	sig, err = MultiChainSignIdentityMapping(s.validTronAddr, s.validDAppAddr, s.validTronKey, evmcompat.SignatureType_TRON, nonce.Nonce)
	r.NoError(err)
	r.NoError(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validTronAddr.MarshalPB(),
		To:        s.validDAppAddr.MarshalPB(),
		Signature: sig,
	}))

	hasMappingResponse, err = amContract.HasMapping(ctx, &HasMappingRequest{
		From: s.validTronAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(true, hasMappingResponse.HasMapping)

	hasMappingResponse, err = amContract.HasMapping(ctx, &HasMappingRequest{
		From: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(true, hasMappingResponse.HasMapping)

	getMultiChainResp, err = amContract.GetMultiChainMapping(ctx, &GetMultiChainMappingRequest{
		From: s.validBinanceAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(s.validDAppAddr.Local.String(), getMultiChainResp.Mappings[0].To.Local.String())
	s.Equal(s.validDAppAddr.ChainID, getMultiChainResp.Mappings[0].To.ChainId)

	dappMultichainMapping, err = amContract.GetMultiChainMapping(ctx, &GetMultiChainMappingRequest{
		From: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	r.Len(dappMultichainMapping.Mappings, 3)

	//Mapp eth2 -- >dapp
	nonce, err = amContract.GetNonce(ctx, &GetNonceRequest{
		Address: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(uint64(3), nonce.Nonce)
	sig, err = MultiChainSignIdentityMapping(s.validEthAddr2, s.validDAppAddr, s.validEthKey2, sigType, nonce.Nonce)
	r.NoError(err)
	r.NoError(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validEthAddr2.MarshalPB(),
		To:        s.validDAppAddr.MarshalPB(),
		Signature: sig,
	}))

	// check eth must already mapped
	hasMappingResponse, err = amContract.HasMapping(ctx, &HasMappingRequest{
		From: s.validEthAddr2.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(true, hasMappingResponse.HasMapping)

	getMultiChainResp, err = amContract.GetMultiChainMapping(ctx, &GetMultiChainMappingRequest{
		From: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Len(getMultiChainResp.Mappings, 4)

	getMultiChainResp, err = amContract.GetMultiChainMapping(ctx, &GetMultiChainMappingRequest{
		From: s.validEthAddr2.MarshalPB(),
	})

	r.NoError(err)
	s.Equal(s.validDAppAddr.Local.String(), getMultiChainResp.Mappings[0].To.Local.String())
	s.Equal(s.validDAppAddr.ChainID, getMultiChainResp.Mappings[0].To.ChainId)

	//Mapp eth2-->dapp again expect error
	sig, err = MultiChainSignIdentityMapping(s.validEthAddr2, s.validDAppAddr, s.validEthKey2, sigType, 4)
	r.NoError(err)
	r.Equal(ErrAlreadyRegistered, amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validEthAddr2.MarshalPB(),
		To:        s.validDAppAddr.MarshalPB(),
		Signature: sig,
	}))

	//Mapp dapp-->eth2 again expect error
	sig, err = MultiChainSignIdentityMapping(s.validDAppAddr, s.validEthAddr2, s.validEthKey2, sigType, 4)
	r.NoError(err)
	r.Equal(ErrAlreadyRegistered, amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validDAppAddr.MarshalPB(),
		To:        s.validEthAddr2.MarshalPB(),
		Signature: sig,
	}))

	getMultiChainResp, err = amContract.GetMultiChainMapping(ctx, &GetMultiChainMappingRequest{
		From: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Len(getMultiChainResp.Mappings, 4)

	nonce, err = amContract.GetNonce(ctx, &GetNonceRequest{
		Address: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	r.Equal(nonce.Address, s.validDAppAddr.MarshalPB())
	r.Equal(uint64(4), nonce.Nonce)

	sig, err = MultiChainSignIdentityMapping(s.validBinanceAddr2, s.validDAppAddr, s.validBinanceKey2, evmcompat.SignatureType_BINANCE, nonce.Nonce)
	r.NoError(err)
	r.NoError(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validBinanceAddr2.MarshalPB(),
		To:        s.validDAppAddr.MarshalPB(),
		Signature: sig,
	}))

	hasMappingResponse, err = amContract.HasMapping(ctx, &HasMappingRequest{
		From: s.validBinanceAddr2.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(true, hasMappingResponse.HasMapping)

	hasMappingResponse, err = amContract.HasMapping(ctx, &HasMappingRequest{
		From: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(true, hasMappingResponse.HasMapping)

	getMultiChainResp, err = amContract.GetMultiChainMapping(ctx, &GetMultiChainMappingRequest{
		From: s.validBinanceAddr2.MarshalPB(),
	})
	r.NoError(err)
	s.Equal(s.validDAppAddr.Local.String(), getMultiChainResp.Mappings[0].To.Local.String())
	s.Equal(s.validDAppAddr.ChainID, getMultiChainResp.Mappings[0].To.ChainId)

	dappMultichainMapping, err = amContract.GetMultiChainMapping(ctx, &GetMultiChainMappingRequest{
		From: s.validDAppAddr.MarshalPB(),
	})
	r.NoError(err)
	r.Len(dappMultichainMapping.Mappings, 5)

	dappMultichainMapping, err = amContract.GetMultiChainMapping(ctx, &GetMultiChainMappingRequest{
		From:    s.validDAppAddr.MarshalPB(),
		ChainId: s.validBinanceAddr.ChainID,
	})
	r.NoError(err)
	r.Len(dappMultichainMapping.Mappings, 2)
}
