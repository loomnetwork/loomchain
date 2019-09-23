package middleware

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain/auth/keys"
	dw "github.com/loomnetwork/loomchain/builtin/plugins/deployer_whitelist"
	udw "github.com/loomnetwork/loomchain/builtin/plugins/user_deployer_whitelist"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/features"
	appstate "github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/txhandler"
	"github.com/loomnetwork/loomchain/vm"
)

const (
	deployId    = uint32(1)
	callId      = uint32(2)
	migrationId = uint32(3)
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
	createDeployerWhitelistCtx func(state appstate.State) (contractpb.Context, error),
) (txhandler.PostCommitMiddleware, error) {
	return txhandler.PostCommitMiddlewareFunc(func(
		state appstate.State,
		txBytes []byte,
		res txhandler.TxHandlerResult,
		next txhandler.PostCommitHandler,
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

		origin := keys.Origin(state.Context())
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
	createDeployerWhitelistCtx func(state appstate.State) (contractpb.Context, error),
) (txhandler.TxMiddlewareFunc, error) {
	return txhandler.TxMiddlewareFunc(func(
		state appstate.State,
		txBytes []byte,
		next txhandler.TxHandlerFunc,
		isCheckTx bool,
	) (res txhandler.TxHandlerResult, err error) {

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

		if tx.Id != deployId && tx.Id != migrationId {
			return next(state, txBytes, isCheckTx)
		}

		var msg vm.MessageTx
		if err := proto.Unmarshal(tx.Data, &msg); err != nil {
			return res, errors.Wrapf(err, "unmarshal message tx %v", tx.Data)
		}

		// Process deployTx, checking for permission to deploy contract
		if tx.Id == deployId {
			var deployTx vm.DeployTx
			if err := proto.Unmarshal(msg.Data, &deployTx); err != nil {
				return res, errors.Wrapf(err, "unmarshal deploy tx %v", msg.Data)
			}

			if deployTx.VmType == vm.VMType_PLUGIN {
				origin := keys.Origin(state.Context())
				ctx, err := createDeployerWhitelistCtx(state)
				if err != nil {
					return res, err
				}
				if err := isAllowedToDeployGo(ctx, origin); err != nil {
					return res, err
				}
			} else if deployTx.VmType == vm.VMType_EVM {
				origin := keys.Origin(state.Context())
				ctx, err := createDeployerWhitelistCtx(state)
				if err != nil {
					return res, err
				}
				if err := isAllowedToDeployEVM(ctx, origin); err != nil {
					return res, err
				}
			}

		} else if tx.Id == migrationId {
			// Process migrationTx, checking for permission to migrate contract
			origin := keys.Origin(state.Context())
			ctx, err := createDeployerWhitelistCtx(state)
			if err != nil {
				return res, err
			}
			if err := isAllowedToMigrate(ctx, origin); err != nil {
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
