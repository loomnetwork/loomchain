package vm

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/abci/types"

	"github.com/loomnetwork/loom"
	"github.com/loomnetwork/loom/store"
)

func mockState() loom.State {
	header := abci.Header{}
	return loom.NewStoreState(context.Background(), store.NewMemStore(), header)
}

func TestProcessDeployTx(t *testing.T) {
	var local loom.LocalAddress

	tx := &DeployTx{
		To: &loom.Address{
			ChainID: "mock",
			Local:   local,
		},
		Code: []byte{4, 5, 6},
	}
	b, err := proto.Marshal(tx)
	require.Nil(t, err)

	_, err = ProcessDeployTx(mockState(), b)
	require.Nil(t, err)
}
