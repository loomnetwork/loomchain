// +build !evm

package sample_go_contract

import (
	types "github.com/loomnetwork/go-loom/builtin/types/sample_go_contract"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/pkg/errors"
)

func (k *SampleGoContract) TestNestedEvmCalls(_ contractpb.Context, _ *types.SampleGoContractNestedEvmRequest) error {
	return errors.New("testing evm in non evm build")
}

func (k *SampleGoContract) TestNestedEvmCalls2(_ contractpb.Context, _ *types.SampleGoContractNestedEvm2Request) error {
	return errors.New("testing evm in non evm build")
}
