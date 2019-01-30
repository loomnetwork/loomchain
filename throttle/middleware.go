package throttle

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/pkg/errors"
)

func GetThrottleTxMiddleWare(
	deployEnabled func(blockHeight int64) bool,
	callEnabled func(blockHeight int64) bool,
	oracle loom.Address,
) loomchain.TxMiddlewareFunc {
	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (res loomchain.TxHandlerResult, err error) {
		blockHeight := state.Block().Height
		isDeployEnabled := deployEnabled(blockHeight)
		isCallEnabled := callEnabled(blockHeight)

		if !isDeployEnabled || !isCallEnabled {
			origin := auth.Origin(state.Context())
			if origin.IsEmpty() {
				return res, errors.New("throttle: transaction has no origin")
			}

			var tx loomchain.Transaction
			if err := proto.Unmarshal(txBytes, &tx); err != nil {
				return res, errors.New("throttle: unmarshal tx")
			}

			if tx.Id == 1 && !deployEnabled(blockHeight) {
				if 0 != origin.Compare(oracle) {
					return res, errors.New("throttle: deploy transactions not enabled")
				}
			}

			if tx.Id == 2 && !callEnabled(blockHeight) {
				if 0 != origin.Compare(oracle) {
					return res, errors.New("throttle: call transactions not enabled")
				}
			}
		}
		return next(state, txBytes, isCheckTx)
	})
}

func GetGoDeployTxMiddleWare(enabled bool, allowedDeployers []loom.Address) loomchain.TxMiddlewareFunc {
	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (res loomchain.TxHandlerResult, err error) {
		if enabled {
			return next(state, txBytes, isCheckTx)
		}

		var tx loomchain.Transaction
		if err := proto.Unmarshal(txBytes, &tx); err != nil {
			return res, errors.Wrapf(err, "unmarshal tx", txBytes)
		}

		if tx.Id != 1 {
			return next(state, txBytes, isCheckTx)
		}

		var msg vm.MessageTx
		if err := proto.Unmarshal(tx.Data, &msg); err != nil {
			return res, errors.Wrapf(err, "unmarshal message tx %v", tx.Data)
		}

		var deployTx vm.DeployTx
		if err := proto.Unmarshal(msg.Data, &deployTx); err != nil {
			return res, errors.Wrapf(err, "unmarshal call tx %v", msg.Data)
		}

		if deployTx.VmType == vm.VMType_PLUGIN {
			origin := auth.Origin(state.Context())
			for _, allowed := range allowedDeployers {
				if 0 == origin.Compare(allowed) {
					return next(state, txBytes, isCheckTx)
				}
			}
			return res, errors.New("origin not permitted go deploy transactions")
		}
		return next(state, txBytes, isCheckTx)

	})
}

type GoDeploy struct {
	Enabled             bool
	DeployerAddressList []string
}

func DefaultGoDeploy() *GoDeploy {
	return &GoDeploy{
		Enabled: true,
	}
}

func (c *GoDeploy) DeployerAddresses() ([]loom.Address, error) {
	var deployerAddressList []loom.Address
	for _, addrStr := range c.DeployerAddressList {
		addr, err := loom.ParseAddress(addrStr)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing deploy address %s", addrStr)
		}
		deployerAddressList = append(deployerAddressList, addr)
	}
	return deployerAddressList, nil
}
