package vm

import (
	"context"
	"testing"

	abci "github.com/tendermint/abci/types"

	"github.com/loomnetwork/loom"
	"github.com/loomnetwork/loom/store"
)

func mockState() loom.State {
	header := abci.Header{}
	return loom.NewStoreState(context.Background(), store.NewMemStore(), header)
}

func TestProcessDeployTx(t *testing.T) {

}
