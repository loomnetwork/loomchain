// +build !evm

package tx_handler

import (
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/pkg/errors"
)

type EthTxHandler struct {
	*vm.Manager
	CreateRegistry factory.RegistryFactoryFunc
}

func (h *EthTxHandler) ProcessTx(
	state loomchain.State,
	txBytes []byte,
	isCheckTx bool,
) (loomchain.TxHandlerResult, error) {
	return loomchain.TxHandlerResult{}, errors.New("eth transaction not supported in non evm build")
}
