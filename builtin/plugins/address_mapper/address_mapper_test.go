package address_mapper

import (
	"testing"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")

	dappAccAddr1 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	ethAccAddr1  = loom.MustParseAddress("eth:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")

	ethTokenAddr = loom.RootAddress("eth")
)

func TestAddressMapperAddNewMapping(t *testing.T) {
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/),
	)

	ethAddrPB := ethAccAddr1.MarshalPB()
	dappAddrPB := addr2.MarshalPB()

	amContract := &AddressMapper{}
	err := amContract.Init(ctx, &InitRequest{})
	require.NoError(t, err)

	err = amContract.AddMapping(ctx, &AddMappingRequest{
		From: ethAddrPB,
		To:   dappAddrPB,
	})
	require.NoError(t, err)

	resp, err := amContract.GetMapping(ctx, &GetMappingRequest{
		From: ethAddrPB,
	})
	require.NoError(t, err)
	assert.Equal(t, ethAddrPB, resp.From)
	assert.Equal(t, dappAddrPB, resp.To)

	resp, err = amContract.GetMapping(ctx, &GetMappingRequest{
		From: dappAddrPB,
	})
	require.NoError(t, err)
	assert.Equal(t, dappAddrPB, resp.From)
	assert.Equal(t, ethAddrPB, resp.To)
}
