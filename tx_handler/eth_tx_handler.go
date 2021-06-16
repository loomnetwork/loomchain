// +build evm

package tx_handler

import (
	"fmt"

	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/golang/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/features"
	"github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/pkg/errors"
)

// EthTxHandler handles signed Ethereum txs that are wrapped inside SignedTx
type EthTxHandler struct {
	*vm.Manager
	CreateRegistry factory.RegistryFactoryFunc
	GasConsumer    func() loomchain.GasConsumer
}

func (h *EthTxHandler) ProcessTx(
	state loomchain.State,
	txBytes []byte,
	isCheckTx bool,
) (loomchain.TxHandlerResult, error) {
	var r loomchain.TxHandlerResult

	if !state.FeatureEnabled(features.EthTxFeature, false) {
		return r, errors.New("ethereum transactions feature not enabled")
	}

	var msg vm.MessageTx
	if err := proto.Unmarshal(txBytes, &msg); err != nil {
		return r, err
	}

	origin := auth.Origin(state.Context())
	caller := loom.UnmarshalAddressPB(msg.From)

	if caller.Compare(origin) != 0 {
		return r, fmt.Errorf("Origin doesn't match caller: - %v != %v", origin, caller)
	}

	// TODO: move the marshalling & validation above this line into middleware
	var ethTx etypes.Transaction
	if err := rlp.DecodeBytes(msg.Data, &ethTx); err != nil {
		return r, err
	}

	// Set r.Info at the earliest opportunity so it can be used by the middleware to figure out how
	// to handle the tx even when the handler doesn't successfully process the tx.
	if ethTx.To() == nil {
		r.Info = utils.DeployEvm
	} else {
		r.Info = utils.CallEVM
	}

	if ethTx.Value().Sign() == -1 {
		return r, errors.New("tx value can't be negative")
	}

	gasConsumer := h.GasConsumer()
	if err := gasConsumer.ApproveGasPurchase(origin, ethTx.Gas(), ethTx.GasPrice()); err != nil {
		return r, err
	}

	// Only do basic validation in CheckTx, don't execute the actual EVM deploy/call
	if isCheckTx {
		return r, nil
	}

	gasConsumer.BuyGas(origin, ethTx.Gas(), ethTx.GasPrice())

	// TODO: create an atomic tx wrapper of state and pass that to InitVM, that way if the tx fails
	// the state changes made in it are all reverted, but the gas fee deduction is not.
	vmInstance, err := h.Manager.InitVM(vm.VMType_EVM, state)
	if err != nil {
		return r, err
	}

	if ethTx.To() == nil { // deploy
		retCreate, addr, err := vmInstance.Create(origin, ethTx.Data(), loom.NewBigUInt(ethTx.Value()), gasConsumer)
		if err != nil {
			return r, errors.Wrap(err, "failed to create contract")
		}

		response, err := proto.Marshal(&vm.DeployResponse{
			Contract: &types.Address{
				ChainId: addr.ChainID,
				Local:   addr.Local,
			},
			Output: retCreate,
		})
		if err != nil {
			return r, errors.Wrap(err, "failed to marshal deploy response")
		}
		r.Data = response

		reg := h.CreateRegistry(state)
		if err := reg.Register("", addr, caller); err != nil {
			return r, errors.Wrap(err, "failed to register contract")
		}
	} else { // call
		to := loom.UnmarshalAddressPB(msg.To)
		r.Data, err = vmInstance.Call(origin, to, ethTx.Data(), loom.NewBigUInt(ethTx.Value()), gasConsumer)
		if err != nil {
			return r, errors.Wrap(err, "contract call failed")
		}
	}
	return r, nil
}
