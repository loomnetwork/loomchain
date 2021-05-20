package db

import (
	"bytes"
	"fmt"
	"math"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/cmd/loom/common"
	"github.com/loomnetwork/loomchain/store"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tendermint/libs/db"
)

func newPruneDBCommand() *cobra.Command {
	var numVersions int64
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Deletes older tree versions from app.db",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := common.ParseConfig()
			if err != nil {
				return err
			}
			return store.PruneDatabase(cfg.DBName, cfg.RootPath(), numVersions)
		},
	}
	flags := cmd.Flags()
	flags.Int64VarP(&numVersions, "versions", "n", 0, "Number of tree versions to prune")
	return cmd
}

func newCompactDBCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compact",
		Short: "Compacts app.db to reclaim disk space",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := common.ParseConfig()
			if err != nil {
				return err
			}
			return store.CompactDatabase(cfg.DBName, cfg.RootPath())
		},
	}
	return cmd
}

func newGetAppHeightCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app-height <path/to/app.db>",
		Short: "Show the last height of app.db",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcDBPath, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("Failed to resolve app.db path '%s'", args[0])
			}
			dbName := strings.TrimSuffix(path.Base(srcDBPath), ".db")
			dbDir := path.Dir(srcDBPath)

			db, err := dbm.NewGoLevelDBWithOpts(dbName, dbDir, &opt.Options{
				ReadOnly: true,
			})
			if err != nil {
				return err
			}
			defer db.Close()

			iavlStore, err := store.NewIAVLStore(db, 0, 0, 0)
			if err != nil {
				return err
			}

			fmt.Printf("app.db at height: %d \n", iavlStore.Version())
			return nil
		},
	}
	return cmd
}

type prefixStat struct {
	NumKeys        int
	TotalKeySize   int
	TotalValueSize int
}

type contractPrefix struct {
	Name    string
	Address string
	Prefix  []byte
}

var (
	oldStandardPrefixes = [][]byte{
		[]byte("vm"),
		[]byte("receipt"),
		[]byte("txHash"),
		[]byte("bloomFilter"),
	}
	curStandardPrefixes = [][]byte{
		[]byte("nonce"),
		[]byte("feature"),
		[]byte("registry"),
		[]byte("reg_caddr"),
		[]byte("reg_crec"),
		[]byte("migrationId"),
	}
	curStandardKeys = [][]byte{
		[]byte("config"),
		[]byte("minbuild"),
		[]byte("vmroot"),
	}
	// names of native contracts that can be resolved to an address via the contract registry
	nativeContractNames = []string{
		"addressmapper",
		"coin",
		"ethcoin",
		"dpos",
		"dposV2",
		"dposV3",
		"gateway",
		"loomcoin-gateway",
		"tron-gateway",
		"binance-gateway",
		"bsc-gateway",
		"deployerwhitelist",
		"user-deployer-whitelist",
		"chainconfig",
		"karma",
		"plasmacash",
	}
)

func getContractStorePrefix(tree *iavl.ImmutableTree, contractName string) ([]byte, error) {
	contractAddrKeyPrefix := []byte("reg_caddr") // registry v2
	_, data := tree.Get(util.PrefixKey(contractAddrKeyPrefix, []byte(contractName)))
	if len(data) == 0 {
		return nil, nil // contract is probably not deployed on this chain
	}
	var contractAddrPB types.Address
	if err := proto.Unmarshal(data, &contractAddrPB); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal contract address")
	}
	contractAddr := loom.UnmarshalAddressPB(&contractAddrPB)
	return util.PrefixKey([]byte("contract"), []byte(contractAddr.Local)), nil
}

func newAnalyzeCommand() *cobra.Command {
	var logLevel int64
	cmd := &cobra.Command{
		Use:   "analyze <path/to/app.db>",
		Short: "Analyze how much space is taken up by data under the standard key prefixes",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcDBPath, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("Failed to resolve app.db path '%s'", args[0])
			}
			dbName := strings.TrimSuffix(path.Base(srcDBPath), ".db")
			dbDir := path.Dir(srcDBPath)

			db, err := dbm.NewGoLevelDBWithOpts(dbName, dbDir, &opt.Options{
				ReadOnly: true,
			})
			if err != nil {
				return err
			}
			defer db.Close()

			mutableTree := iavl.NewMutableTree(db, 0)
			treeVersion, err := mutableTree.Load()
			if err != nil {
				return errors.Wrap(err, "failed to load mutable tree")
			}

			immutableTree, err := mutableTree.GetImmutable(treeVersion)
			if err != nil {
				return errors.Wrapf(err, "failed to load immutable tree for version %v", treeVersion)
			}

			leaves := uint(immutableTree.Size())
			var progressInterval uint64
			if logLevel > 0 {
				progressInterval = uint64(leaves / uint(math.Pow(10, float64(logLevel))))
			}

			fmt.Printf("IAVL tree height %v with %v keys\n", immutableTree.Height(), immutableTree.Size())

			prefixes := append([][]byte{}, oldStandardPrefixes...)
			prefixes = append(prefixes, curStandardPrefixes...)
			// each native contract has its own prefix in the store
			contractPrefixToNameMap := map[string]string{}
			for _, contractName := range nativeContractNames {
				prefix, err := getContractStorePrefix(immutableTree, contractName)
				if err != nil {
					return err
				}
				if prefix != nil {
					contractPrefixToNameMap[string(prefix)] = contractName
					prefixes = append(prefixes, prefix)
				}
			}

			rawStats := map[string]*prefixStat{}
			var miscStat prefixStat
			var keyCount uint64
			startTime := time.Now()
			immutableTree.Iterate(func(key, value []byte) bool {
				var stat *prefixStat
				for _, prefix := range prefixes {
					if util.HasPrefix(key, prefix) {
						stat = rawStats[string(prefix)]
						if stat == nil {
							stat = &prefixStat{}
							rawStats[string(prefix)] = stat
						}

						stat.NumKeys++
						stat.TotalKeySize += len(key)
						stat.TotalValueSize += len(value)
						break
					}
				}

				// track anything that didn't fall within one of the standard prefixes
				if stat == nil {
					miscStat.NumKeys++
					miscStat.TotalKeySize += len(key)
					miscStat.TotalValueSize += len(value)
					if !util.HasPrefix(key, []byte("contract")) {
						fmt.Printf("Unprefixed key %x\n", key)
					}
				}

				keyCount++
				if progressInterval > 0 && (keyCount%progressInterval) == 0 {
					elapsed := time.Since(startTime).Minutes()
					fractionDone := float64(keyCount) / float64(leaves)
					expected := elapsed / fractionDone

					fmt.Printf(
						"%v%% done in %v mins. ETA %v mins.\n",
						int(fractionDone*100), int(elapsed), int(expected-elapsed),
					)
				}
				return false
			})

			stats := map[string]*prefixStat{}
			sortedPrefixes := []string{}
			for prefix, stat := range rawStats {
				if util.HasPrefix([]byte(prefix), []byte("contract")) {
					contractName := contractPrefixToNameMap[prefix]
					if contractName != "" {
						stats[contractName] = stat
						sortedPrefixes = append(sortedPrefixes, contractName)
					} else {
						fmt.Printf("Unknown contract prefix %x\n", []byte(prefix))
					}
				} else {
					stats[prefix] = stat
					sortedPrefixes = append(sortedPrefixes, prefix)
				}
			}

			// sort by prefix
			sort.Strings(sortedPrefixes)
			sortedPrefixes = append(sortedPrefixes, "misc")
			stats["misc"] = &miscStat

			var totalKeySize, totalValueSize int
			for _, prefix := range sortedPrefixes {
				if stat := stats[prefix]; stat != nil {
					totalKeySize += stat.TotalKeySize
					totalValueSize += stat.TotalValueSize
				}
			}

			// pretty print
			ml := struct {
				Prefix   int
				Keys     int
				KSize    int
				VSize    int
				KPercent int
				VPercent int
			}{
				Prefix:   20,
				Keys:     20,
				KSize:    20,
				VSize:    20,
				KPercent: 6,
				VPercent: 6,
			}

			// ensure the longest prefix fits the first column
			for _, prefix := range sortedPrefixes {
				if len(prefix) > ml.Prefix {
					ml.Prefix = len(prefix)
				}
			}

			fmt.Printf(
				"%-*s | %-*s | %-*s | %-*s | %-*s | %-*s\n",
				ml.Prefix, "Prefix",
				ml.Keys, "Keys",
				ml.KSize, "K Size",
				ml.VSize, "V Size",
				ml.KPercent, "K %",
				ml.VPercent, "V %",
			)
			fmt.Printf(strings.Repeat("-", ml.Prefix+ml.Keys+ml.KSize+ml.VSize+ml.KPercent+ml.VPercent+16) + "\n")

			for _, prefix := range sortedPrefixes {
				if stat := stats[prefix]; stat != nil {
					keyPercentage := float64(stat.TotalKeySize) / float64(totalKeySize) * 100.0
					valuePercentage := float64(stat.TotalValueSize) / float64(totalValueSize) * 100.0
					fmt.Printf(
						"%-*s | %-*d | %-*d | %-*d | %*.2f | %*.2f\n",
						ml.Prefix, prefix,
						ml.Keys, stat.NumKeys,
						ml.KSize, stat.TotalKeySize,
						ml.VSize, stat.TotalValueSize,
						ml.KPercent, keyPercentage,
						ml.VPercent, valuePercentage,
					)
				}
			}

			return nil
		},
	}
	cmd.Flags().Int64Var(&logLevel, "log", 0, "How often progress output should be printed. 1 - every 10%, 2 - every 1%, 3 - every 0.1%.")
	return cmd
}

// Returns the bytes that mark the end of the key range for the given prefix.
func prefixRangeEnd(prefix []byte) []byte {
	if prefix == nil {
		return nil
	}

	end := make([]byte, len(prefix))
	copy(end, prefix)

	for {
		if end[len(end)-1] != byte(255) {
			end[len(end)-1]++
			break
		} else if len(end) == 1 {
			end = nil
			break
		}
		end = end[:len(end)-1]
	}
	return end
}

func newExtractCurrentStateCommand() *cobra.Command {
	// TODO: batching currently creates new tree versions, which is undesirable if we want to
	//       retain the tree version == block number correspondance
	var batchSize uint64
	cmd := &cobra.Command{
		Use:   "extract-current-state <path/to/src_app.db> <path/to/dest_app.db>",
		Short: "Copy all the keys & values that belong to the current state into a new DB",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("Failed to resolve app.db path '%s'", args[0])
			}
			dbName := strings.TrimSuffix(path.Base(dbPath), ".db")
			dbDir := path.Dir(dbPath)
			srcDB, err := dbm.NewGoLevelDBWithOpts(dbName, dbDir, &opt.Options{
				ReadOnly: true,
			})
			if err != nil {
				return err
			}
			defer srcDB.Close()

			dbPath, err = filepath.Abs(args[1])
			if err != nil {
				return fmt.Errorf("Failed to resolve app.db path '%s'", args[1])
			}
			dbName = strings.TrimSuffix(path.Base(dbPath), ".db")
			dbDir = path.Dir(dbPath)
			destDB, err := dbm.NewGoLevelDB(dbName, dbDir)
			if err != nil {
				return errors.Wrapf(err, "failed to open %v", dbPath)
			}
			defer destDB.Close()

			mutableTree := iavl.NewMutableTree(srcDB, 0)
			treeVersion, err := mutableTree.Load()
			if err != nil {
				return errors.Wrap(err, "failed to load mutable tree")
			}

			immutableTree, err := mutableTree.GetImmutable(treeVersion)
			if err != nil {
				return errors.Wrapf(err, "failed to load immutable tree for version %v", treeVersion)
			}

			// The version of the new tree will be incremented by one before it's saved to disk, so
			// to retain the same version number as the original tree when the new tree is saved to
			// disk we have to initialize the new tree with a lower version than the original.
			newMutableTree := iavl.NewMutableTreeWithVersion(destDB, 0, treeVersion-1)

			fmt.Printf("IAVL tree height %v with %v keys\n", immutableTree.Height(), immutableTree.Size())

			prefixes := append([][]byte{}, curStandardPrefixes...)
			contractPrefixToNameMap := map[string]string{}
			for _, contractName := range nativeContractNames {
				prefix, err := getContractStorePrefix(immutableTree, contractName)
				if err != nil {
					return err
				}
				if prefix != nil {
					contractPrefixToNameMap[string(prefix)] = contractName
					prefixes = append(prefixes, prefix)
				}
			}

			numKeys := uint64(0)
			rawStats := map[string]*prefixStat{}
			startTime := time.Now()

			// copy out all the keys under the prefixes that are still in use
			for _, prefix := range prefixes {
				var itError *error
				immutableTree.IterateRange(
					prefix,
					prefixRangeEnd(prefix),
					true,
					func(key, value []byte) bool {
						// This is just a sanity check, should never actually happen!
						if !util.HasPrefix(key, prefix) {
							err := errors.Errorf(
								"key does not have prefix, skipped key: %x prefix: %x",
								key, prefix,
							)
							fmt.Println(err)
							itError = &err
							return true
						}

						newMutableTree.Set(key, value)
						stat := rawStats[string(prefix)]
						if stat == nil {
							stat = &prefixStat{}
							rawStats[string(prefix)] = stat
						}

						stat.NumKeys++
						stat.TotalKeySize += len(key)
						stat.TotalValueSize += len(value)
						numKeys++

						if (numKeys > 0) && (batchSize > 0) && (numKeys%batchSize == 0) {
							hash, version, err := newMutableTree.SaveVersion()
							if err != nil {
								err := errors.Wrap(err, "failed to save new tree version")
								itError = &err
								return true
							}
							fmt.Printf(
								"%d keys processed in %v mins, saved version %d, hash %x\n",
								numKeys, time.Since(startTime).Minutes(), version, hash,
							)
						}
						return false
					},
				)
				if itError != nil {
					return *itError
				}
			}

			// copy out the misc keys
			var miscStat prefixStat
			for _, key := range curStandardKeys {
				if immutableTree.Has(key) {
					_, value := immutableTree.Get(key)
					newMutableTree.Set(key, value)

					miscStat.NumKeys++
					miscStat.TotalKeySize += len(key)
					miscStat.TotalValueSize += len(value)
					numKeys++
				}
			}

			// write the remaining keys to disk
			if numKeys > 0 {
				hash, version, err := newMutableTree.SaveVersion()
				if err != nil {
					return errors.Wrap(err, "failed to save new tree version")
				}
				fmt.Printf(
					"%d keys processed in %v mins, saved version %d, hash %x\n",
					numKeys, time.Since(startTime).Minutes(), version, hash,
				)
			}

			fmt.Printf("Copy complete, %d keys processed in total\n", numKeys)

			stats := map[string]*prefixStat{}
			sortedPrefixes := []string{}
			for prefix, stat := range rawStats {
				if util.HasPrefix([]byte(prefix), []byte("contract")) {
					contractName := contractPrefixToNameMap[prefix]
					if contractName != "" {
						stats[contractName] = stat
						sortedPrefixes = append(sortedPrefixes, contractName)
					} else {
						fmt.Printf("Unknown contract prefix %x\n", []byte(prefix))
					}
				} else {
					stats[string(prefix)] = stat
					sortedPrefixes = append(sortedPrefixes, string(prefix))
				}
			}

			sort.Strings(sortedPrefixes)
			sortedPrefixes = append(sortedPrefixes, "misc")
			stats["misc"] = &miscStat

			var totalKeySize, totalValueSize int
			for _, prefix := range sortedPrefixes {
				if stat := stats[prefix]; stat != nil {
					totalKeySize += stat.TotalKeySize
					totalValueSize += stat.TotalValueSize
				}
			}

			ml := struct {
				Prefix int
				Keys   int
				KSize  int
				VSize  int
			}{
				Prefix: 20,
				Keys:   20,
				KSize:  20,
				VSize:  20,
			}

			// ensure the longest prefix fits the first column
			for _, prefix := range sortedPrefixes {
				if len(prefix) > ml.Prefix {
					ml.Prefix = len(prefix)
				}
			}

			fmt.Printf(
				"%-*s | %-*s | %-*s | %-*s\n",
				ml.Prefix, "Prefix",
				ml.Keys, "Keys",
				ml.KSize, "K Size",
				ml.VSize, "V Size",
			)
			fmt.Printf(strings.Repeat("-", ml.Prefix+ml.Keys+ml.KSize+ml.VSize+16) + "\n")

			for _, prefix := range sortedPrefixes {
				if stat := stats[prefix]; stat != nil {
					fmt.Printf(
						"%-*s | %-*d | %-*d | %-*d\n",
						ml.Prefix, prefix,
						ml.Keys, stat.NumKeys,
						ml.KSize, stat.TotalKeySize,
						ml.VSize, stat.TotalValueSize,
					)
				}
			}

			fmt.Printf("New IAVL tree height %v with %v keys\n", newMutableTree.Height(), newMutableTree.Size())

			return nil
		},
	}
	cmd.Flags().Uint64Var(&batchSize, "batch-size", 0, "Number of keys to write to disk in each batch, by default no batching is done.")
	return cmd
}

func newCompareCurrentStateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare-current-state <path/to/src_app.db> <path/to/dest_app.db>",
		Short: "Compare all the keys & values that belong to the current state between two DBs.",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("Failed to resolve app.db path '%s'", args[0])
			}
			dbName := strings.TrimSuffix(path.Base(dbPath), ".db")
			dbDir := path.Dir(dbPath)
			srcDB, err := dbm.NewGoLevelDBWithOpts(dbName, dbDir, &opt.Options{
				ReadOnly: true,
			})
			if err != nil {
				return err
			}
			defer srcDB.Close()

			dbPath, err = filepath.Abs(args[1])
			if err != nil {
				return fmt.Errorf("Failed to resolve app.db path '%s'", args[1])
			}
			dbName = strings.TrimSuffix(path.Base(dbPath), ".db")
			dbDir = path.Dir(dbPath)
			destDB, err := dbm.NewGoLevelDBWithOpts(dbName, dbDir, &opt.Options{
				ReadOnly: true,
			})
			if err != nil {
				return errors.Wrapf(err, "failed to open %v", dbPath)
			}
			defer destDB.Close()

			mutableTree := iavl.NewMutableTree(srcDB, 0)
			srcTreeVersion, err := mutableTree.Load()
			if err != nil {
				return errors.Wrap(err, "failed to load mutable tree")
			}

			srcImmutableTree, err := mutableTree.GetImmutable(srcTreeVersion)
			if err != nil {
				return errors.Wrapf(
					err, "failed to load immutable tree for version %v (from %s)", srcTreeVersion, args[0],
				)
			}

			mutableTree = iavl.NewMutableTree(destDB, 0)
			destTreeVersion, err := mutableTree.Load()
			if err != nil {
				return errors.Wrap(err, "failed to load mutable tree")
			}

			destImmutableTree, err := mutableTree.GetImmutable(destTreeVersion)
			if err != nil {
				return errors.Wrapf(
					err, "failed to load immutable tree for version %v (from %s)", destTreeVersion, args[1],
				)
			}

			if srcTreeVersion != destTreeVersion {
				fmt.Printf(
					"WARNING: Tree version mismatch (original tree is at version %d, new tree is at version %d\n",
					srcTreeVersion, destTreeVersion,
				)
			}

			fmt.Printf(
				"Original IAVL tree height %v with %v keys (version %d)\n",
				srcImmutableTree.Height(), srcImmutableTree.Size(), srcTreeVersion,
			)
			fmt.Printf(
				"New IAVL tree height %v with %v keys (version %d)\n",
				destImmutableTree.Height(), destImmutableTree.Size(), destTreeVersion,
			)

			prefixes := append([][]byte{}, curStandardPrefixes...)
			contractPrefixToNameMap := map[string]string{}
			for _, contractName := range nativeContractNames {
				prefix, err := getContractStorePrefix(srcImmutableTree, contractName)
				if err != nil {
					return err
				}
				if prefix != nil {
					contractPrefixToNameMap[string(prefix)] = contractName
					prefixes = append(prefixes, prefix)
				}
			}

			numKeys := uint64(0)
			rawStats := map[string]*prefixStat{}
			startTime := time.Now()

			// compare all the keys under the prefixes that are still in use
			for _, prefix := range prefixes {
				var itError *error
				srcImmutableTree.IterateRange(
					prefix,
					prefixRangeEnd(prefix),
					true,
					func(key, value []byte) bool {
						// This is just a sanity check, should never actually happen!
						if !util.HasPrefix(key, prefix) {
							err := errors.Errorf(
								"key does not have prefix, skipped key: %x prefix: %x",
								key, prefix,
							)
							itError = &err
							return true
						}

						_, v := destImmutableTree.Get(key)
						if bytes.Compare(value, v) != 0 {
							fmt.Printf("Value mismatch for key %x (prefix %x)\n", key, prefix)
						}
						stat := rawStats[string(prefix)]
						if stat == nil {
							stat = &prefixStat{}
							rawStats[string(prefix)] = stat
						}

						stat.NumKeys++
						numKeys++

						return false
					},
				)
				if itError != nil {
					return *itError
				}
			}

			// copy out the misc keys
			var miscStat prefixStat
			for _, key := range curStandardKeys {
				if srcImmutableTree.Has(key) {
					_, srcValue := srcImmutableTree.Get(key)
					_, destValue := destImmutableTree.Get(key)

					if bytes.Compare(srcValue, destValue) != 0 {
						fmt.Printf("Value mismatch for key %x\n", key)
					}
					miscStat.NumKeys++
					numKeys++
				}
			}

			fmt.Printf(
				"Comparison complete, %d keys processed in total in %v mins\n",
				numKeys, time.Since(startTime).Minutes(),
			)

			stats := map[string]*prefixStat{}
			sortedPrefixes := []string{}
			for prefix, stat := range rawStats {
				if util.HasPrefix([]byte(prefix), []byte("contract")) {
					contractName := contractPrefixToNameMap[prefix]
					if contractName != "" {
						stats[contractName] = stat
						sortedPrefixes = append(sortedPrefixes, contractName)
					} else {
						fmt.Printf("Unknown contract prefix %x\n", []byte(prefix))
					}
				} else {
					stats[string(prefix)] = stat
					sortedPrefixes = append(sortedPrefixes, string(prefix))
				}
			}

			sort.Strings(sortedPrefixes)
			sortedPrefixes = append(sortedPrefixes, "misc")
			stats["misc"] = &miscStat

			ml := struct {
				Prefix int
				Keys   int
			}{
				Prefix: 20,
				Keys:   20,
			}

			// ensure the longest prefix fits the first column
			for _, prefix := range sortedPrefixes {
				if len(prefix) > ml.Prefix {
					ml.Prefix = len(prefix)
				}
			}

			fmt.Printf(
				"%-*s | %-*s\n",
				ml.Prefix, "Prefix",
				ml.Keys, "Keys",
			)
			fmt.Printf(strings.Repeat("-", ml.Prefix+ml.Keys+16) + "\n")

			for _, prefix := range sortedPrefixes {
				if stat := stats[prefix]; stat != nil {
					fmt.Printf(
						"%-*s | %-*d\n",
						ml.Prefix, prefix,
						ml.Keys, stat.NumKeys)
				}
			}

			return nil
		},
	}
	return cmd
}
