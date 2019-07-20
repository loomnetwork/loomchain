// +build evm

package address_mapper

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	ssha "github.com/miguelmota/go-solidity-sha3"
	"github.com/stretchr/testify/suite"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
)

type AddressMapperTestSuite struct {
	suite.Suite
	validEthKey     *ecdsa.PrivateKey
	invalidEthKey   *ecdsa.PrivateKey
	validEthAddr    loom.Address
	invalidEthAddr  loom.Address
	validDAppAddr   loom.Address
	invalidDAppAddr loom.Address
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

	sig, err := SignIdentityMapping(s.validEthAddr, s.validDAppAddr, s.validEthKey)
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
}

func (s *AddressMapperTestSuite) TestAddressMapperAddSameIdentityMapping() {
	r := s.Require()
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(s.validDAppAddr /*caller*/, loom.RootAddress("chain") /*contract*/),
	)

	amContract := &AddressMapper{}
	r.NoError(amContract.Init(ctx, &InitRequest{}))

	sig, err := SignIdentityMapping(s.validEthAddr, s.validDAppAddr, s.validEthKey)
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

	invertedSig, err := SignIdentityMapping(s.validDAppAddr, s.validEthAddr, s.validEthKey)
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

	sig, err := SignIdentityMapping(s.validEthAddr, s.validDAppAddr, s.validEthKey)
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
}

func (s *AddressMapperTestSuite) TestListMapping() {
	r := s.Require()
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(s.validDAppAddr /*caller*/, loom.RootAddress("chain") /*contract*/),
	)

	amContract := &AddressMapper{}
	r.NoError(amContract.Init(ctx, &InitRequest{}))

	sig, err := SignIdentityMapping(s.validEthAddr, s.validDAppAddr, s.validEthKey)
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

	sig, err := SignIdentityMapping(s.validDAppAddr, s.validEthAddr, s.validEthKey)
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

	sig, err := SignIdentityMapping(s.validEthAddr, s.validDAppAddr, s.invalidEthKey)
	r.NoError(err)
	r.Error(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validEthAddr.MarshalPB(),
		To:        s.validDAppAddr.MarshalPB(),
		Signature: sig,
	}), "Should error if signature doesn't match foreign chain address")

	sig, err = SignIdentityMapping(s.invalidEthAddr, s.validDAppAddr, s.validEthKey)
	r.NoError(err)
	r.Error(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.invalidEthAddr.MarshalPB(),
		To:        s.validDAppAddr.MarshalPB(),
		Signature: sig,
	}), "Should error if from address doesn't have a Chain ID")

	sig, err = SignIdentityMapping(s.validEthAddr, s.invalidDAppAddr, s.validEthKey)
	r.NoError(err)
	r.Error(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.validEthAddr.MarshalPB(),
		To:        s.invalidDAppAddr.MarshalPB(),
		Signature: sig,
	}), "Should error if to address doesn't have a Chain ID")

	sig, err = SignIdentityMapping(s.invalidEthAddr, s.invalidDAppAddr, s.validEthKey)
	r.Error(amContract.AddIdentityMapping(ctx, &AddIdentityMappingRequest{
		From:      s.invalidEthAddr.MarshalPB(),
		To:        s.invalidDAppAddr.MarshalPB(),
		Signature: sig,
	}), "Should error if both addresses don't have a Chain ID")

	sig, err = SignIdentityMapping(s.validEthAddr, addr2, s.validEthKey)
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
