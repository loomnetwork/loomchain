package throttle

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	dw "github.com/loomnetwork/loomchain/builtin/plugins/deployer_whitelist"
	udw "github.com/loomnetwork/loomchain/builtin/plugins/user_deployer_whitelist"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/features"
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
		isCheckTx bool,
	) error {
		if !state.FeatureEnabled(features.UserDeployerWhitelistFeature, false) {
			return next(state, txBytes, res, isCheckTx)
		}

		// If it isn't EVM deployment, no need to proceed further
		if res.Info != utils.DeployEvm {
			return next(state, txBytes, res, isCheckTx)
		}

		// This is checkTx, so bail out early.
		if len(res.Data) == 0 {
			return next(state, txBytes, res, isCheckTx)
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

		return next(state, txBytes, res, isCheckTx)
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

		if !state.FeatureEnabled(features.DeployerWhitelistFeature, false) {
			return next(state, txBytes, isCheckTx)
		}

		var nonceTx auth.NonceTx
		if err := proto.Unmarshal(txBytes, &nonceTx); err != nil {
			return res, errors.Wrap(err, "throttle: unwrap nonce Tx")
		}

		var tx types.Transaction
		if err := proto.Unmarshal(nonceTx.Inner, &tx); err != nil {
			return res, errors.New("throttle: unmarshal tx")
		}

		switch types.TxID(tx.Id) {
		case types.TxID_DEPLOY:
			var msg vm.MessageTx
			if err := proto.Unmarshal(tx.Data, &msg); err != nil {
				return res, errors.Wrap(err, "failed to unmarshal MessageTx")
			}

			var deployTx vm.DeployTx
			if err := proto.Unmarshal(msg.Data, &deployTx); err != nil {
				return res, errors.Wrap(err, "failed to unmarshal DeployTx")
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

		case types.TxID_MIGRATION:
			origin := auth.Origin(state.Context())
			ctx, err := createDeployerWhitelistCtx(state)
			if err != nil {
				return res, err
			}
			if err := isAllowedToMigrate(ctx, origin); err != nil {
				return res, err
			}

		case types.TxID_ETHEREUM:
			if !state.FeatureEnabled(features.EthTxFeature, false) {
				break
			}

			var msg vm.MessageTx
			if err := proto.Unmarshal(tx.Data, &msg); err != nil {
				return res, errors.Wrap(err, "failed to unmarshal MessageTx")
			}
			isDeploy, err := isEthDeploy(msg.Data)
			if err != nil {
				return res, err
			}
			if !isDeploy {
				break
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
