// +build evm

package factory

import (
	ethvm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/evm"
	"github.com/loomnetwork/loomchain/migrations"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/throttle"
	"github.com/loomnetwork/loomchain/tx_handler"
	"github.com/loomnetwork/loomchain/vm"
)

func NewTxHandlerFactory(
	cfg config.Config,
	vmManager *vm.Manager,
	chainID string,
	store store.VersionedKVStore,
	createRegistry factory.RegistryFactoryFunc,
) loomchain.TxHandlerFactory {
	return txHandleFactory{
		cfg:            cfg,
		vmManager:      vmManager,
		chainID:        chainID,
		store:          store,
		createRegistry: createRegistry,
	}
}

type txHandleFactory struct {
	cfg            config.Config
	vmManager      *vm.Manager
	chainID        string
	store          store.VersionedKVStore
	createRegistry factory.RegistryFactoryFunc
}

func (f txHandleFactory) Copy(newStore store.VersionedKVStore) loomchain.TxHandlerFactory {
	return txHandleFactory{
		cfg:            f.cfg,
		vmManager:      f.vmManager,
		chainID:        f.chainID,
		store:          newStore,
		createRegistry: f.createRegistry,
	}
}

// Creates a handle with an entirely new vmManager with dummy account balance manager factory and receipt handler.
func (f txHandleFactory) TxHandlerWithTracerAndDefaultVmManager(tracer ethvm.Tracer, metrics bool) (loomchain.TxHandler, error) {
	f.vmManager = createVmManager(tracer)
	return f.TxHandler(metrics)
}

func (f txHandleFactory) TxHandler(metrics bool) (loomchain.TxHandler, error) {
	nonceTxHandler := auth.NewNonceHandler()

	txMiddleware, err := txMiddleWare(f.cfg, *f.vmManager, nonceTxHandler, f.chainID, f.store, metrics)
	if err != nil {
		return nil, err
	}
	postCommitMiddlewares, err := postCommitMiddleWAre(f.cfg, *f.vmManager, nonceTxHandler)
	if err != nil {
		return nil, err
	}

	return loomchain.MiddlewareTxHandler(
		txMiddleware,
		router(f.cfg, *f.vmManager, f.createRegistry),
		postCommitMiddlewares,
	), nil
}

func createVmManager(tracer ethvm.Tracer) *vm.Manager {
	managerWithTracer := vm.NewManager()
	managerWithTracer.Register(vm.VMType_EVM, func(_state loomchain.State) (vm.VM, error) {
		var createABM evm.AccountBalanceManagerFactoryFunc
		lvm := evm.NewLoomVm(_state, nil, createABM)
		return lvm.WithTracer(tracer), nil
	})
	return managerWithTracer
}

func txMiddleWare(
	cfg config.Config,
	vmManager vm.Manager,
	nonceTxHandler *auth.NonceHandler,
	chainID string,
	appStore store.VersionedKVStore,
	metrics bool,
) ([]loomchain.TxMiddleware, error) {
	txMiddleWare := []loomchain.TxMiddleware{
		loomchain.LogTxMiddleware,
		loomchain.RecoveryTxMiddleware,
	}

	txMiddleWare = append(txMiddleWare, auth.NewChainConfigMiddleware(
		cfg.Auth,
		getContractStaticCtx("addressmapper", vmManager),
	))

	if cfg.Karma.Enabled {
		txMiddleWare = append(txMiddleWare, throttle.GetKarmaMiddleWare(
			cfg.Karma.Enabled,
			cfg.Karma.MaxCallCount,
			cfg.Karma.SessionDuration,
			getContractCtx("karma", vmManager),
		))
	}

	if cfg.TxLimiter.Enabled {
		txMiddleWare = append(txMiddleWare, throttle.NewTxLimiterMiddleware(cfg.TxLimiter))
	}

	if cfg.ContractTxLimiter.Enabled {
		udwCtxFactory := getContractCtx("user-deployer-whitelist", vmManager)
		txMiddleWare = append(
			txMiddleWare, throttle.NewContractTxLimiterMiddleware(cfg.ContractTxLimiter, udwCtxFactory),
		)
	}

	if cfg.DeployerWhitelist.ContractEnabled {
		dwMiddleware, err := throttle.NewDeployerWhitelistMiddleware(getContractCtx("deployerwhitelist", vmManager))
		if err != nil {
			return nil, err
		}
		txMiddleWare = append(txMiddleWare, dwMiddleware)

	}

	txMiddleWare = append(txMiddleWare, nonceTxHandler.TxMiddleware(appStore))

	if cfg.GoContractDeployerWhitelist.Enabled {
		goDeployers, err := cfg.GoContractDeployerWhitelist.DeployerAddresses(chainID)
		if err != nil {
			return nil, errors.Wrapf(err, "getting list of users allowed go deploys")
		}
		txMiddleWare = append(txMiddleWare, throttle.GetGoDeployTxMiddleWare(goDeployers))
	}

	if metrics {
		txMiddleWare = append(txMiddleWare, loomchain.NewInstrumentingTxMiddleware())
	}

	return txMiddleWare, nil
}

func router(
	cfg config.Config,
	vmManager vm.Manager,
	createRegistry factory.RegistryFactoryFunc,
) loomchain.TxHandler {
	router := loomchain.NewTxRouter()
	isEvmTx := func(txID uint32, _ loomchain.State, txBytes []byte, isCheckTx bool) bool {
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
		Manager:                &vmManager,
		CreateRegistry:         createRegistry,
		AllowNamedEVMContracts: cfg.AllowNamedEvmContracts,
	}

	callTxHandler := &vm.CallTxHandler{
		Manager: &vmManager,
	}

	ethTxHandler := &tx_handler.EthTxHandler{
		Manager:        &vmManager,
		CreateRegistry: createRegistry,
	}

	migrationTxHandler := &tx_handler.MigrationTxHandler{
		Manager:        &vmManager,
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
	router.HandleDeliverTx(4, loomchain.GeneratePassthroughRouteHandler(ethTxHandler))

	// TODO: Write this in more elegant way
	router.HandleCheckTx(1, loomchain.GenerateConditionalRouteHandler(isEvmTx, loomchain.NoopTxHandler, deployTxHandler))
	router.HandleCheckTx(2, loomchain.GenerateConditionalRouteHandler(isEvmTx, loomchain.NoopTxHandler, callTxHandler))
	router.HandleCheckTx(3, loomchain.GenerateConditionalRouteHandler(isEvmTx, loomchain.NoopTxHandler, migrationTxHandler))
	router.HandleCheckTx(4, loomchain.GenerateConditionalRouteHandler(isEvmTx, loomchain.NoopTxHandler, ethTxHandler))

	return router
}

func postCommitMiddleWAre(
	cfg config.Config,
	vmManager vm.Manager,
	nonceTxHandler *auth.NonceHandler,
) ([]loomchain.PostCommitMiddleware, error) {
	postCommitMiddlewares := []loomchain.PostCommitMiddleware{
		loomchain.LogPostCommitMiddleware,
	}

	if cfg.UserDeployerWhitelist.ContractEnabled {
		udwContractFactory := getContractCtx("user-deployer-whitelist", vmManager)
		evmDeployRecorderMiddleware, err := throttle.NewEVMDeployRecorderPostCommitMiddleware(udwContractFactory)
		if err != nil {
			return nil, err
		}
		postCommitMiddlewares = append(postCommitMiddlewares, evmDeployRecorderMiddleware)
	}

	// We need to make sure nonce post commit middleware is last as
	// it doesn't pass control to other middlewares after it.
	postCommitMiddlewares = append(postCommitMiddlewares, nonceTxHandler.PostCommitMiddleware())

	return postCommitMiddlewares, nil
}

func getContractCtx(pluginName string, vmManager vm.Manager) func(_ loomchain.State) (contractpb.Context, error) {
	return func(_state loomchain.State) (contractpb.Context, error) {
		pvm, err := vmManager.InitVM(vm.VMType_PLUGIN, _state)
		if err != nil {
			return nil, err
		}
		return plugin.NewInternalContractContext(pluginName, pvm.(*plugin.PluginVM), false)
	}
}

func getContractStaticCtx(pluginName string, vmManager vm.Manager) func(_ loomchain.State) (contractpb.StaticContext, error) {
	return func(_state loomchain.State) (contractpb.StaticContext, error) {
		pvm, err := vmManager.InitVM(vm.VMType_PLUGIN, _state)
		if err != nil {
			return nil, err
		}
		return plugin.NewInternalContractContext(pluginName, pvm.(*plugin.PluginVM), true)
	}
}
