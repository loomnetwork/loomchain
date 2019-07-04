// +build !evm

package vm

import (
	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/registry/factory"
)

type EthTxHandler struct {
	*Manager
	CreateRegistry factory.RegistryFactoryFunc
}

func (h *EthTxHandler) ProcessTx(
	state loomchain.State,
	txBytes []byte,
	isCheckTx bool,
) (loomchain.TxHandlerResult, error) {
	return loomchain.TxHandlerResult{}, errors.New("eth transaction not supported in non evm build")
}