package debug

import (
	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain"
)

func TraceTransaction(state loomchain.State, blockNumber, txIndex uint64, config JsonTraceConfig) (interface{}, error) {

	return nil, errors.New("not implemented")
}
