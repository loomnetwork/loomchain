package middleware

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/migrations"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/throttle"
	"github.com/loomnetwork/loomchain/tx_handler"
	"github.com/loomnetwork/loomchain/vm"
)

func TxMiddleWare(
	cfg *config.Config,
	vmManager *vm.Manager,
	chainID string,
	appStore store.VersionedKVStore,
) ([]loomchain.TxMiddleware, error) {
	txMiddleWare := []loomchain.TxMiddleware{
		loomchain.LogTxMiddleware,
		loomchain.RecoveryTxMiddleware,
	}

	txMiddleWare = append(txMiddleWare, auth.NewChainConfigMiddleware(
		cfg.Auth,
		GetContractStaticCtx("addressmapper", vmManager),
	))

	createKarmaContractCtx := GetContractCtx("karma", vmManager)

	if cfg.Karma.Enabled {
		txMiddleWare = append(txMiddleWare, throttle.GetKarmaMiddleWare(
			cfg.Karma.Enabled,
			cfg.Karma.MaxCallCount,
			cfg.Karma.SessionDuration,
			createKarmaContractCtx,
		))
	}

	if cfg.TxLimiter.Enabled {
		txMiddleWare = append(txMiddleWare, throttle.NewTxLimiterMiddleware(cfg.TxLimiter))
	}

	if cfg.ContractTxLimiter.Enabled {
		contextFactory := GetContractCtx("user-deployer-whitelist", vmManager)
		txMiddleWare = append(
			txMiddleWare, throttle.NewContractTxLimiterMiddleware(cfg.ContractTxLimiter, contextFactory),
		)
	}

	if cfg.DeployerWhitelist.ContractEnabled {
		contextFactory := GetContractCtx("deployerwhitelist", vmManager)
		dwMiddleware, err := throttle.NewDeployerWhitelistMiddleware(contextFactory)
		if err != nil {
			return nil, err
		}
		txMiddleWare = append(txMiddleWare, dwMiddleware)

	}

	txMiddleWare = append(txMiddleWare, auth.NonceTxMiddleware(appStore))

	if cfg.GoContractDeployerWhitelist.Enabled {
		goDeployers, err := cfg.GoContractDeployerWhitelist.DeployerAddresses(chainID)
		if err != nil {
			return nil, errors.Wrapf(err, "getting list of users allowed go deploys")
		}
		txMiddleWare = append(txMiddleWare, throttle.GetGoDeployTxMiddleWare(goDeployers))
	}

	txMiddleWare = append(txMiddleWare, loomchain.NewInstrumentingTxMiddleware())

	return txMiddleWare, nil
}

func Router(
	cfg *config.Config,
	vmManager *vm.Manager,
	createRegistry factory.RegistryFactoryFunc,
) loomchain.TxHandler {
	router := loomchain.NewTxRouter()

	isEvmTx := func(txID uint32, s state.State, txBytes []byte, isCheckTx bool) bool {
		var msg vm.MessageTx
		err := proto.Unmarshal(txBytes, &msg)
		if err != nil {
			return false
		}

		switch txID {
		case 1:
			var tx vm.DeployTx
			err = proto.Unmarshal(msg.Data, &tx)
			if err != nil {
				// In case of error, let's give safest response,
				// let's TxHandler down the line, handle it.
				return false
			}
			return tx.VmType == vm.VMType_EVM
		case 2:
			var tx vm.CallTx
			err = proto.Unmarshal(msg.Data, &tx)
			if err != nil {
				// In case of error, let's give safest response,
				// let's TxHandler down the line, handle it.
				return false
			}
			return tx.VmType == vm.VMType_EVM
		case 3:
			return false
		default:
			return false
		}
	}

	deployTxHandler := &vm.DeployTxHandler{
		Manager:                vmManager,
		CreateRegistry:         createRegistry,
		AllowNamedEVMContracts: cfg.AllowNamedEvmContracts,
	}

	callTxHandler := &vm.CallTxHandler{
		Manager: vmManager,
	}

	migrationTxHandler := &tx_handler.MigrationTxHandler{
		Manager:        vmManager,
		CreateRegistry: createRegistry,
		Migrations: map[int32]tx_handler.MigrationFunc{
			1: migrations.DPOSv3Migration,
			2: migrations.GatewayMigration,
			3: migrations.GatewayMigration,
		},
	}

	router.HandleDeliverTx(1, loomchain.GeneratePassthroughRouteHandler(deployTxHandler))
	router.HandleDeliverTx(2, loomchain.GeneratePassthroughRouteHandler(callTxHandler))
	router.HandleDeliverTx(3, loomchain.GeneratePassthroughRouteHandler(migrationTxHandler))

	// TODO: Write this in more elegant way
	router.HandleCheckTx(1, loomchain.GenerateConditionalRouteHandler(isEvmTx, loomchain.NoopTxHandler, deployTxHandler))
	router.HandleCheckTx(2, loomchain.GenerateConditionalRouteHandler(isEvmTx, loomchain.NoopTxHandler, callTxHandler))
	router.HandleCheckTx(3, loomchain.GenerateConditionalRouteHandler(isEvmTx, loomchain.NoopTxHandler, migrationTxHandler))

	return router
}

func PostCommitMiddleWAre(cfg *config.Config, vmManager *vm.Manager) ([]loomchain.PostCommitMiddleware, error) {
	postCommitMiddlewares := []loomchain.PostCommitMiddleware{
		loomchain.LogPostCommitMiddleware,
	}

	if cfg.UserDeployerWhitelist.ContractEnabled {
		contextFactory := GetContractCtx("user-deployer-whitelist", vmManager)
		evmDeployRecorderMiddleware, err := throttle.NewEVMDeployRecorderPostCommitMiddleware(contextFactory)
		if err != nil {
			return nil, err
		}
		postCommitMiddlewares = append(postCommitMiddlewares, evmDeployRecorderMiddleware)
	}

	// We need to make sure nonce post commit middleware is last as
	// it doesn't pass control to other middlewares after it.
	postCommitMiddlewares = append(postCommitMiddlewares, auth.NonceTxPostNonceMiddleware)

	return postCommitMiddlewares, nil
}

type contextFactory func(s state.State) (contractpb.Context, error)

type staticContextFactory func(s state.State) (contractpb.StaticContext, error)

func GetContractCtx(pluginName string, vmManager *vm.Manager) contextFactory {
	return func(s state.State) (contractpb.Context, error) {
		pvm, err := vmManager.InitVM(vm.VMType_PLUGIN, s)
		if err != nil {
			return nil, err
		}
		return plugin.NewInternalContractContext(pluginName, pvm.(*plugin.PluginVM), false)
	}
}

func GetContractStaticCtx(pluginName string, vmManager *vm.Manager) staticContextFactory {
	return func(s state.State) (contractpb.StaticContext, error) {
		pvm, err := vmManager.InitVM(vm.VMType_PLUGIN, s)
		if err != nil {
			return nil, err
		}
		return plugin.NewInternalContractContext(pluginName, pvm.(*plugin.PluginVM), true)
	}
}
