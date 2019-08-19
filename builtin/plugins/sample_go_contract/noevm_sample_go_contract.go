// +build !evm

package sample_go_contract

import (
	types "github.com/loomnetwork/go-loom/builtin/types/testing"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/pkg/errors"
)

func (k *SampleGoContract) TestNestedEvmCalls(ctx contractpb.Context, req *types.TestingNestedEvmRequest) error {
	return errors.New("testing evm in non evm build")
}
