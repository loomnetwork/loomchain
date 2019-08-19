// +build !evm

package sample_go_contract

import (
	types "github.com/loomnetwork/go-loom/builtin/types/sample_go_contract"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/pkg/errors"
)

func (k *SampleGoContract) TestNestedEvmCalls(ctx contractpb.Context, req *types.SampleGoContractNestedEvmRequest) error {
	return errors.New("testing evm in non evm build")
}
