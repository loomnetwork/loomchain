package plasma_cash

import (
	"testing"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	addr3 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

func TestPlasmaCashSMT(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)

	contract := &PlasmaCash{}
	err := contract.Init(ctx, &InitRequest{})

	require.Nil(t, err)

	//verify blockheight is zero
	pbk := &PlasmaBookKeeping{}
	ctx.Get(blockHeightKey, pbk)

	assert.Equal(t, uint64(0), pbk.CurrentHeight.Value.Uint64(), "height should be 1")

	req := &PlasmaTxRequest{}
	err = contract.PlasmaTxRequest(ctx, req)
	require.Nil(t, err)

	//verify that blockheight is now one
	ctx.Get(blockHeightKey, pbk)

	assert.Equal(t, uint64(1), pbk.CurrentHeight.Value.Uint64(), "height should be 1")

}
