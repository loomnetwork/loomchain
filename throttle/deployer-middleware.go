package throttle

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	dw "github.com/loomnetwork/loomchain/builtin/plugins/deployer_whitelist"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/pkg/errors"
)

var (
	// ErrDeployerWhitelistContractNotFound indicates that the DeployerWhitelist contract hasn't been deployed yet.
	ErrDeployerWhitelistContractNotFound = errors.New("[DeployerWhitelistMiddleware] DeployerWhitelist contract not found")
	// ErrrNotAuthorized indicates that the deployment failed because the caller didn't have
	// the permission to deploy contract.
	ErrNotAuthorized = errors.New("[DeployerWhitelistMiddleware] not authorized")
)

func GetDeployerWhitelistMiddleWare(
	createDeployerWhitelistCtx func(state loomchain.State) (contractpb.Context, error),
) (loomchain.TxMiddlewareFunc, error) {
	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (res loomchain.TxHandlerResult, err error) {
		var tx loomchain.Transaction
		if err := proto.Unmarshal(txBytes, &tx); err != nil {
			return res, errors.Wrapf(err, "unmarshal tx %v", txBytes)
		}

		if tx.Id != deployId {
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
			ctx, err := createDeployerWhitelistCtx(state)
			if err != nil {
				return res, err
			}
			if err := isAllowedToDeployGo(ctx, origin); err != nil {
				return res, err
			}
		}

		if deployTx.VmType == vm.VMType_EVM {
			origin := auth.Origin(state.Context())
			ctx, err := createDeployerWhitelistCtx(state)
			if err != nil {
				return res, err
			}
			if err := isAllowedToDeployEVM(ctx, origin); err != nil {
				return res, err
			}
		}

		return next(state, txBytes, isCheckTx)
	}), nil
}

func isAllowedToDeployGo(ctx contractpb.Context, deployerAddr loom.Address) error {
	deployer, err := dw.GetDeployer(ctx, deployerAddr)
	if err != nil {
		return ErrNotAuthorized
	}
	if deployer.Permission == dw.BOTHDeployer || deployer.Permission == dw.GODeployer {
		return nil
	}
	return ErrNotAuthorized
}

func isAllowedToDeployEVM(ctx contractpb.Context, deployerAddr loom.Address) error {
	deployer, err := dw.GetDeployer(ctx, deployerAddr)
	if err != nil {
		return ErrNotAuthorized
	}
	if deployer.Permission == dw.BOTHDeployer || deployer.Permission == dw.EVMDeployer {
		return nil
	}
	return ErrNotAuthorized
}
