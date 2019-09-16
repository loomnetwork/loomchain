// +build evm

package db

import (
	"context"
	"fmt"
	"math"
	"path"
	"path/filepath"
	"strings"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/cmd/loom/common"
	cdb "github.com/loomnetwork/loomchain/db"
	"github.com/loomnetwork/loomchain/events"
	"github.com/loomnetwork/loomchain/evm"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/receipts"
	registry "github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/store"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
)

func newDumpEVMStateCommand() *cobra.Command {
	var appHeight int64

	cmd := &cobra.Command{
		Use:   "evm-dump",
		Short: "Dumps EVM state stored at a specific block height",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := common.ParseConfig()
			if err != nil {
				return err
			}

			db, err := dbm.NewGoLevelDB(cfg.DBName, cfg.RootPath())
			if err != nil {
				return err
			}
			appStore, err := store.NewIAVLStore(db, 0, appHeight, 0)
			if err != nil {
				return err
			}

			eventHandler := loomchain.NewDefaultEventHandler(events.NewLogEventDispatcher())

			regVer, err := registry.RegistryVersionFromInt(cfg.RegistryVersion)
			if err != nil {
				return err
			}
			createRegistry, err := registry.NewRegistryFactory(regVer)
			if err != nil {
				return err
			}

			receiptHandlerProvider := receipts.NewReceiptHandlerProvider(
				eventHandler,
				cfg.EVMPersistentTxReceiptsMax,
				nil,
			)

			// TODO: This should use snapshot obtained from appStore.ReadOnlyState()
			storeTx := store.WrapAtomic(appStore).BeginTx()
			state := loomchain.NewStoreState(
				context.Background(),
				storeTx,
				abci.Header{
					Height: appStore.Version(),
				},
				// it is possible to load the block hash from the TM block store, but probably don't
				// need it for just dumping the EVM state
				nil,
				nil,
			)

			var newABMFactory plugin.NewAccountBalanceManagerFactoryFunc
			if evm.EVMEnabled && cfg.EVMAccountsEnabled {
				newABMFactory = plugin.NewAccountBalanceManagerFactory
			}

			var accountBalanceManager evm.AccountBalanceManager
			if newABMFactory != nil {
				pvm := plugin.NewPluginVM(
					common.NewDefaultContractsLoader(cfg),
					state,
					createRegistry(state),
					eventHandler,
					log.Default,
					newABMFactory,
					receiptHandlerProvider.Writer(),
					receiptHandlerProvider.Reader(),
				)
				createABM, err := newABMFactory(pvm)
				if err != nil {
					return err
				}
				accountBalanceManager = createABM(true)
				if err != nil {
					return err
				}
			}

			vm, err := evm.NewLoomEvm(state, accountBalanceManager, nil, false)
			if err != nil {
				return err
			}

			fmt.Printf("\n--- EVM state at app height %d ---\n%s\n", appStore.Version(), string(vm.RawDump(false, false, true)))
			return nil
		},
	}

	cmdFlags := cmd.Flags()
	cmdFlags.Int64Var(&appHeight, "app-height", 0, "Dump EVM state as it was the specified app height")
	return cmd
}

func newDumpEVMStateMultiWriterAppStoreCommand() *cobra.Command {
	var appHeight int64
	var evmDBName string
	cmd := &cobra.Command{
		Use:   "evm-dump-2",
		Short: "Dumps EVM state stored at a specific block height from MultiWriterAppStore",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := common.ParseConfig()
			if err != nil {
				return err
			}

			db, err := dbm.NewGoLevelDB(cfg.DBName, cfg.RootPath())
			if err != nil {
				return err
			}
			evmDB, err := cdb.LoadDB(
				"goleveldb",
				evmDBName,
				cfg.RootPath(),
				256,
				4,
				false,
			)
			if err != nil {
				return err
			}
			iavlStore, err := store.NewIAVLStore(db, 0, appHeight, 0)
			if err != nil {
				return err
			}
			evmStore := store.NewEvmStore(evmDB, 100)
			if err := evmStore.LoadVersion(iavlStore.Version()); err != nil {
				return err
			}

			appStore, err := store.NewMultiWriterAppStore(iavlStore, evmStore, false)
			if err != nil {
				return err
			}
			eventHandler := loomchain.NewDefaultEventHandler(events.NewLogEventDispatcher())

			regVer, err := registry.RegistryVersionFromInt(cfg.RegistryVersion)
			if err != nil {
				return err
			}
			createRegistry, err := registry.NewRegistryFactory(regVer)
			if err != nil {
				return err
			}

			receiptHandlerProvider := receipts.NewReceiptHandlerProvider(
				eventHandler,
				cfg.EVMPersistentTxReceiptsMax,
				nil,
			)

			// TODO: This should use snapshot obtained from appStore.ReadOnlyState()
			storeTx := store.WrapAtomic(appStore).BeginTx()
			state := loomchain.NewStoreState(
				context.Background(),
				storeTx,
				abci.Header{
					Height: appStore.Version(),
				},
				// it is possible to load the block hash from the TM block store, but probably don't
				// need it for just dumping the EVM state
				nil,
				nil,
			)

			var newABMFactory plugin.NewAccountBalanceManagerFactoryFunc
			if evm.EVMEnabled && cfg.EVMAccountsEnabled {
				newABMFactory = plugin.NewAccountBalanceManagerFactory
			}

			var accountBalanceManager evm.AccountBalanceManager
			if newABMFactory != nil {
				pvm := plugin.NewPluginVM(
					common.NewDefaultContractsLoader(cfg),
					state,
					createRegistry(state),
					eventHandler,
					log.Default,
					newABMFactory,
					receiptHandlerProvider.Writer(),
					receiptHandlerProvider.Reader(),
				)
				createABM, err := newABMFactory(pvm)
				if err != nil {
					return err
				}
				accountBalanceManager = createABM(true)
				if err != nil {
					return err
				}
			}

			vm, err := evm.NewLoomEvm(state, accountBalanceManager, nil, false)
			if err != nil {
				return err
			}

			fmt.Printf("\n--- EVM state at app height %d ---\n%s\n", appStore.Version(), string(vm.RawDump(false, false, true)))
			return nil
		},
	}

	cmdFlags := cmd.Flags()
	cmdFlags.Int64Var(&appHeight, "app-height", 0, "Dump EVM state as it was the specified app height")
	cmdFlags.StringVar(&evmDBName, "evmdb-name", "evm", "Dump EVM state as it was the specified app height")
	return cmd
}

func newGetEvmHeightCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "evm-height <path/to/evm.db>",
		Short: "Show the last height of evm.db",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcDBPath, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("Failed to resolve evm.db path '%s'", args[0])
			}
			dbName := strings.TrimSuffix(path.Base(srcDBPath), ".db")
			dbDir := path.Dir(srcDBPath)

			db, err := cdb.LoadDB(
				"goleveldb",
				dbName,
				dbDir,
				256,
				4,
				false,
			)
			if err != nil {
				return err
			}
			defer db.Close()

			evmStore := store.NewEvmStore(db, 100)
			if err := evmStore.LoadVersion(math.MaxInt64); err != nil {
				return err
			}
			root, version := evmStore.Version()

			fmt.Printf("evm.db at height: %d root: %X\n", version, root)
			return nil
		},
	}
	return cmd
}