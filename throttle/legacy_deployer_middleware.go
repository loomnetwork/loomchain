package throttle

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/types"
	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain/auth/keys"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	appstate "github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/txhandler"
	"github.com/loomnetwork/loomchain/vm"
)

// GetGoDeployTxMiddleWare creates middlware that only allows Go contract deployment tx to go through
// if they originate from one of the allowed deployer accounts. This middleware has been superseded
// by the DeployerWhitelist contract & middleware, though it's still in use on some clusters.
func GetGoDeployTxMiddleWare(allowedDeployers []loom.Address) txhandler.TxMiddlewareFunc {
	return txhandler.TxMiddlewareFunc(func(
		state appstate.State,
		txBytes []byte,
		next txhandler.TxHandlerFunc,
		isCheckTx bool,
	) (res txhandler.TxHandlerResult, err error) {
		var tx types.Transaction
		if err := proto.Unmarshal(txBytes, &tx); err != nil {
			return res, errors.Wrapf(err, "unmarshal tx %v", txBytes)
		}

		if types.TxID(tx.Id) != types.TxID_DEPLOY {
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
			origin := keys.Origin(state.Context())
			for _, allowed := range allowedDeployers {
				if 0 == origin.Compare(allowed) {
					return next(state, txBytes, isCheckTx)
				}
			}
			return res, fmt.Errorf(`%s not authorized to deploy Go contract`, origin.String())
		}
		return next(state, txBytes, isCheckTx)
	})
}
