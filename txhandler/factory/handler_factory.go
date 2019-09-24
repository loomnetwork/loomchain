package factory

import (
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/pkg/errors"

	ethvm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/gogo/protobuf/proto"

	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/evm"
	"github.com/loomnetwork/loomchain/migrations"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/throttle"
	"github.com/loomnetwork/loomchain/tx_handler"
	"github.com/loomnetwork/loomchain/txhandler"
	"github.com/loomnetwork/loomchain/txhandler/middleware"
	"github.com/loomnetwork/loomchain/vm"
)

func NewTxHandlerFactory(
	cfg config.Config,
	vmManager *vm.Manager,
	chainID string,
	store store.VersionedKVStore,
	createRegistry factory.RegistryFactoryFunc,
) txhandler.TxHandlerFactory {
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

func (f txHandleFactory) TxHandler(tracer ethvm.Tracer, metrics bool) (txhandler.TxHandler, error) {
	vmManager := createVmManager(f.vmManager, tracer)

	txMiddleware, err := txMiddleWare(f.cfg, vmManager, f.chainID, f.store, metrics)
	if err != nil {
		return nil, err
	}
	postCommitMiddlewares, err := postCommitMiddleWAre(f.cfg, vmManager)
	if err != nil {
		return nil, err
	}

	return txhandler.MiddlewareTxHandler(
		txMiddleware,
		router(f.cfg, vmManager, f.createRegistry),
		postCommitMiddlewares,
	), nil
}

func createVmManager(vmManager *vm.Manager, tracer ethvm.Tracer) vm.Manager {
	if tracer == nil && vmManager != nil {
		return *vmManager
	}
	managerWithTracer := vm.NewManager()
	managerWithTracer.Register(vm.VMType_EVM, func(_state state.State) (vm.VM, error) {
		var createABM evm.AccountBalanceManagerFactoryFunc
		return evm.NewLoomVm(_state, nil, createABM, false, tracer), nil
	})
	return *managerWithTracer
}

func txMiddleWare(
	cfg config.Config,
	vmManager vm.Manager,
	chainID string,
	appStore store.VersionedKVStore,
	metrics bool,
) ([]txhandler.TxMiddleware, error) {
	txMiddleWare := []txhandler.TxMiddleware{
		txhandler.LogTxMiddleware,
		txhandler.RecoveryTxMiddleware,
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
		txMiddleWare = append(txMiddleWare, middleware.NewTxLimiterMiddleware(cfg.TxLimiter))
	}

	if cfg.ContractTxLimiter.Enabled {
		udwCtxFactory := getContractCtx("user-deployer-whitelist", vmManager)
		txMiddleWare = append(
			txMiddleWare, middleware.NewContractTxLimiterMiddleware(cfg.ContractTxLimiter, udwCtxFactory),
		)
	}

	if cfg.DeployerWhitelist.ContractEnabled {
		dwMiddleware, err := middleware.NewDeployerWhitelistMiddleware(getContractCtx("deployerwhitelist", vmManager))
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

	if metrics {
		txMiddleWare = append(txMiddleWare, txhandler.NewInstrumentingTxMiddleware())
	}

	return txMiddleWare, nil
}

func router(
	cfg config.Config,
	vmManager vm.Manager,
	createRegistry factory.RegistryFactoryFunc,
) txhandler.TxHandler {
	router := middleware.NewTxRouter()
	isEvmTx := func(txID uint32, _ state.State, txBytes []byte, isCheckTx bool) bool {
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

	migrationTxHandler := &tx_handler.MigrationTxHandler{
		Manager:        &vmManager,
		CreateRegistry: createRegistry,
		Migrations: map[int32]tx_handler.MigrationFunc{
			1: migrations.DPOSv3Migration,
			2: migrations.GatewayMigration,
			3: migrations.GatewayMigration,
		},
	}

	router.HandleDeliverTx(1, middleware.GeneratePassthroughRouteHandler(deployTxHandler))
	router.HandleDeliverTx(2, middleware.GeneratePassthroughRouteHandler(callTxHandler))
	router.HandleDeliverTx(3, middleware.GeneratePassthroughRouteHandler(migrationTxHandler))

	// TODO: Write this in more elegant way
	router.HandleCheckTx(1, middleware.GenerateConditionalRouteHandler(isEvmTx, txhandler.NoopTxHandler, deployTxHandler))
	router.HandleCheckTx(2, middleware.GenerateConditionalRouteHandler(isEvmTx, txhandler.NoopTxHandler, callTxHandler))
	router.HandleCheckTx(3, middleware.GenerateConditionalRouteHandler(isEvmTx, txhandler.NoopTxHandler, migrationTxHandler))

	return router
}

func postCommitMiddleWAre(cfg config.Config, vmManager vm.Manager) ([]txhandler.PostCommitMiddleware, error) {
	postCommitMiddlewares := []txhandler.PostCommitMiddleware{
		txhandler.LogPostCommitMiddleware,
	}

	if cfg.UserDeployerWhitelist.ContractEnabled {
		udwContractFactory := getContractCtx("user-deployer-whitelist", vmManager)
		evmDeployRecorderMiddleware, err := middleware.NewEVMDeployRecorderPostCommitMiddleware(udwContractFactory)
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

func getContractCtx(pluginName string, vmManager vm.Manager) func(_ state.State) (contractpb.Context, error) {
	return func(_state state.State) (contractpb.Context, error) {
		pvm, err := vmManager.InitVM(vm.VMType_PLUGIN, _state)
		if err != nil {
			return nil, err
		}
		return plugin.NewInternalContractContext(pluginName, pvm.(*plugin.PluginVM), false)
	}
}

func getContractStaticCtx(pluginName string, vmManager vm.Manager) func(_ state.State) (contractpb.StaticContext, error) {
	return func(_state state.State) (contractpb.StaticContext, error) {
		pvm, err := vmManager.InitVM(vm.VMType_PLUGIN, _state)
		if err != nil {
			return nil, err
		}
		return plugin.NewInternalContractContext(pluginName, pvm.(*plugin.PluginVM), true)
	}
}
