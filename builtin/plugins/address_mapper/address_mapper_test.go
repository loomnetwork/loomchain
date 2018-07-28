// +build evm

package address_mapper

import (
	"crypto/ecdsa"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
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

func (s *AddressMapperTestSuite) TestAddressMapperAddNewContractMapping() {
	r := s.Require()
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(s.validDAppAddr /*caller*/, loom.RootAddress("chain") /*contract*/),
	)

	amContract := &AddressMapper{}
	r.NoError(amContract.Init(ctx, &InitRequest{}))

	r.NoError(amContract.AddContractMapping(ctx, &AddContractMappingRequest{
		From: s.validEthAddr.MarshalPB(),
		To:   s.validDAppAddr.MarshalPB(),
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

func (s *AddressMapperTestSuite) TestAddressMapperAddNewInvertedContractMapping() {
	r := s.Require()
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(s.validDAppAddr /*caller*/, loom.RootAddress("chain") /*contract*/),
	)

	amContract := &AddressMapper{}
	r.NoError(amContract.Init(ctx, &InitRequest{}))

	r.NoError(amContract.AddContractMapping(ctx, &AddContractMappingRequest{
		From: s.validDAppAddr.MarshalPB(),
		To:   s.validEthAddr.MarshalPB(),
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
