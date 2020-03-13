// +build evm

package db

import (
	"context"
	"fmt"
	"math"
	"path"
	"path/filepath"
	"strings"

	gcommon "github.com/ethereum/go-ethereum/common"
	gstate "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
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

			fmt.Printf("\n--- EVM state at app height %d ---\n%s\n", appStore.Version(), string(vm.RawDump()))
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

			fmt.Printf("\n--- EVM state at app height %d ---\n%s\n", appStore.Version(), string(vm.RawDump()))
			return nil
		},
	}

	cmdFlags := cmd.Flags()
	cmdFlags.Int64Var(&appHeight, "app-height", 0, "Dump EVM state as it was the specified app height")
	cmdFlags.StringVar(&evmDBName, "evmdb-name", "evm", "Name of EVM state database")
	return cmd
}

func newDumpEVMStateFromEvmDB() *cobra.Command {
	var appHeight int64
	var evmDBName string
	var dumpStorageTrie bool
	cmd := &cobra.Command{
		Use:   "evm-dump-3",
		Short: "Dumps EVM state stored at a specific block height from evm.db",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := common.ParseConfig()
			if err != nil {
				return err
			}

			if appHeight == 0 {
				appHeight = math.MaxInt64
			}

			evmDB, err := cdb.LoadDB(
				"goleveldb", evmDBName,
				cfg.RootPath(), 256, 4, false,
			)
			if err != nil {
				return err
			}

			evmStore := store.NewEvmStore(evmDB, 100)
			if err := evmStore.LoadVersion(appHeight); err != nil {
				return err
			}
			root, version := evmStore.Version()
			evmRoot := gcommon.BytesToHash(root)

			fmt.Printf("version: %d, root: %x\n", version, root)

			// TODO: This should use snapshot obtained from appStore.ReadOnlyState()
			storeTx := store.WrapAtomic(evmStore).BeginTx()
			state := loomchain.NewStoreState(
				context.Background(),
				storeTx,
				abci.Header{
					Height: appHeight,
				},
				// it is possible to load the block hash from the TM block store, but probably don't
				// need it for just dumping the EVM state
				nil,
				nil,
			)

			srcStateDB := gstate.NewDatabase(evm.NewLoomEthdb(state, nil))
			srcStateDBTrie, err := srcStateDB.OpenTrie(evmRoot)
			if err != nil {
				fmt.Printf("cannot open trie, %s\n", evmRoot.Hex())
				return err
			}
			srcTrie, err := trie.New(evmRoot, srcStateDB.TrieDB())
			if err != nil {
				return err
			}
			srcState, err := gstate.New(evmRoot, srcStateDB)
			if err != nil {
				return err
			}

			it := trie.NewIterator(srcTrie.NodeIterator(nil))
			for it.Next() {
				addrBytes := srcStateDBTrie.GetKey(it.Key)
				addr := gcommon.BytesToAddress(addrBytes)
				var data gstate.Account
				if err := rlp.DecodeBytes(it.Value, &data); err != nil {
					panic(err)
				}

				fmt.Printf("Account: %s\n", gcommon.Bytes2Hex(addrBytes))
				fmt.Printf("- Balance: %s\n", data.Balance.String())
				fmt.Printf("- Nonce: %d\n", data.Nonce)
				fmt.Printf("- Root: %s\n", gcommon.Bytes2Hex(data.Root[:]))
				fmt.Printf("- CodeHash: %s\n", gcommon.Bytes2Hex((data.CodeHash)))
				fmt.Printf("- Code: %x\n", gcommon.Bytes2Hex(srcState.GetCode(addr)))
				fmt.Printf("- Storage Root: %x\n", srcState.StorageTrie(addr).Hash())

				if dumpStorageTrie {
					fmt.Println("- Storage Data:")
					srcState.ForEachStorage(addr, func(key, value gcommon.Hash) bool {
						fmt.Printf("	%s:%s\n", key.Hex(), value.Hex())
						return true
					})
				}

				fmt.Println("---------------------------------------")
			}
			return nil
		},
	}

	cmdFlags := cmd.Flags()
	cmdFlags.Int64Var(&appHeight, "app-height", 0, "Dump EVM state as it was the specified app height")
	cmdFlags.StringVar(&evmDBName, "evmdb-name", "evm", "Name of EVM state database")
	cmdFlags.BoolVar(&dumpStorageTrie, "storage-trie", false, "Dump all storage tries of accounts")
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
