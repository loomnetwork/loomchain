package throttle

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	dw "github.com/loomnetwork/loomchain/builtin/plugins/deployer_whitelist"
	udw "github.com/loomnetwork/loomchain/builtin/plugins/user_deployer_whitelist"
	"github.com/loomnetwork/loomchain/eth/utils"
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

// NewEVMDeployRecorderPostCommitMiddleware returns post-commit middleware that
// Records deploymentAddress and vmType
func NewEVMDeployRecorderPostCommitMiddleware(
	createDeployerWhitelistCtx func(state loomchain.State) (contractpb.Context, error),
) (loomchain.PostCommitMiddleware, error) {
	return loomchain.PostCommitMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		res loomchain.TxHandlerResult,
		next loomchain.PostCommitHandler,
	) error {
		if !state.FeatureEnabled(loomchain.UserDeployerWhitelistFeature, false) {
			return next(state, txBytes, res)
		}

		// If it isn't EVM deployment, no need to proceed further
		if res.Info != utils.DeployEvm {
			return next(state, txBytes, res)
		}

		// This is checkTx, so bail out early.
		if len(res.Data) == 0 {
			return next(state, txBytes, res)
		}

		var deployResponse vm.DeployResponse
		if err := proto.Unmarshal(res.Data, &deployResponse); err != nil {
			return errors.Wrapf(err, "unmarshal deploy response %v", res.Data)
		}

		origin := auth.Origin(state.Context())
		ctx, err := createDeployerWhitelistCtx(state)
		if err != nil {
			return err
		}

		if err := udw.RecordEVMContractDeployment(ctx, origin, loom.UnmarshalAddressPB(deployResponse.Contract)); err != nil {
			return errors.Wrapf(err, "error while recording deployment")
		}

		return next(state, txBytes, res)
	}), nil
}

func NewDeployerWhitelistMiddleware(
	createDeployerWhitelistCtx func(state loomchain.State) (contractpb.Context, error),
) (loomchain.TxMiddlewareFunc, error) {
	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (res loomchain.TxHandlerResult, err error) {

		if !state.FeatureEnabled(loomchain.DeployerWhitelistFeature, false) {
			return next(state, txBytes, isCheckTx)
		}

		var nonceTx auth.NonceTx
		if err := proto.Unmarshal(txBytes, &nonceTx); err != nil {
			return res, errors.Wrap(err, "throttle: unwrap nonce Tx")
		}

		var tx loomchain.Transaction
		if err := proto.Unmarshal(nonceTx.Inner, &tx); err != nil {
			return res, errors.New("throttle: unmarshal tx")
		}

		switch tx.Id {
		case callId:
		case deployId: {
			var msg vm.MessageTx
			if err := proto.Unmarshal(tx.Data, &msg); err != nil {
				return res, errors.Wrapf(err, "unmarshal message tx %v", tx.Data)
			}

			var deployTx vm.DeployTx
			if err := proto.Unmarshal(msg.Data, &deployTx); err != nil {
				return res, errors.Wrapf(err, "unmarshal deploy tx %v", msg.Data)
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
			} else if deployTx.VmType == vm.VMType_EVM {
				origin := auth.Origin(state.Context())
				ctx, err := createDeployerWhitelistCtx(state)
				if err != nil {
					return res, err
				}
				if err := isAllowedToDeployEVM(ctx, origin); err != nil {
					return res, err
				}
			}
		}
		case ethId: {
			var msg vm.MessageTx
			if err := proto.Unmarshal(tx.Data, &msg); err != nil {
				return res, errors.Wrapf(err, "unmarshal message tx %v", tx.Data)
			}
			isDeploy, err := isEthDeploy(msg.Data)
			if err != nil {
				return res, err
			}
			if !isDeploy {
				return next(state, txBytes, isCheckTx)
			}
			origin := auth.Origin(state.Context())
			ctx, err := createDeployerWhitelistCtx(state)
			if err != nil {
				return res, err
			}
			if err := isAllowedToDeployEVM(ctx, origin); err != nil {
				return res, err
			}
		}
		case migrationId:
			// Process migrationTx, checking for permission to migrate contract
			origin := auth.Origin(state.Context())
			ctx, err := createDeployerWhitelistCtx(state)
			if err != nil {
				return res, err
			}
			if err := isAllowedToMigrate(ctx, origin); err != nil {
				return res, err
			}
		default:
			return res, errors.Errorf("unrecognised tx id %v", tx.Id)
		}
		return next(state, txBytes, isCheckTx)
	}), nil
}

func isAllowedToDeployGo(ctx contractpb.Context, deployerAddr loom.Address) error {
	deployer, err := dw.GetDeployer(ctx, deployerAddr)
	if err != nil {
		return err
	}
	if dw.IsFlagSet(uint32(deployer.Flags), uint32(dw.AllowGoDeployFlag)) {
		return nil
	}
	return ErrNotAuthorized
}

func isAllowedToDeployEVM(ctx contractpb.Context, deployerAddr loom.Address) error {
	deployer, err := dw.GetDeployer(ctx, deployerAddr)
	if err != nil {
		return err
	}
	if dw.IsFlagSet(uint32(deployer.Flags), uint32(dw.AllowEVMDeployFlag)) {
		return nil
	}
	return ErrNotAuthorized
}

func isAllowedToMigrate(ctx contractpb.Context, deployerAddr loom.Address) error {
	deployer, err := dw.GetDeployer(ctx, deployerAddr)
	if err != nil {
		return err
	}
	if dw.IsFlagSet(uint32(deployer.Flags), uint32(dw.AllowMigrationFlag)) {
		return nil
	}
	return ErrNotAuthorized
}
