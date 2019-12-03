// +build evm

package db

import (
	"fmt"
	"math"
	"path"
	"path/filepath"
	"strings"

	gcommon "github.com/ethereum/go-ethereum/common"
	gstate "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/loomnetwork/loomchain/cmd/loom/common"
	cdb "github.com/loomnetwork/loomchain/db"
	"github.com/loomnetwork/loomchain/store"
	"github.com/spf13/cobra"
)

func newDumpEVMStateFromEvmDB() *cobra.Command {
	var appHeight int64
	var evmDBName string
	var dumpStorageTrie bool
	cmd := &cobra.Command{
		Use:   "evm-dump",
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

			evmStore := store.NewEvmStore(evmDB, 100, cfg.AppStore.IAVLFlushInterval)
			if err := evmStore.LoadVersion(appHeight); err != nil {
				return err
			}
			root, version := evmStore.Version()
			evmRoot := gcommon.BytesToHash(root)

			fmt.Printf("version: %d, root: %x\n", version, root)

			srcStateDB := gstate.NewDatabase(store.NewLoomEthDB(evmStore))
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

			evmStore := store.NewEvmStore(db, 100, 0)
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
