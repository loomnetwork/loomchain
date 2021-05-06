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

	"github.com/loomnetwork/go-loom"
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

func contractAddressToPrefix(contractAddr string) []byte {
	addr, err := loom.LocalAddressFromHexString(contractAddr)
	if err != nil {
		panic(err)
	}
	return util.PrefixKey([]byte("contract"), []byte(addr))
}

func newAnalyzeCommand() *cobra.Command {
	var logLevel int64
	prefixes := [][]byte{
		[]byte("nonce"),
		[]byte("vm"),
		[]byte("receipt"),
		[]byte("txHash"),
		[]byte("bloomFilter"),
		[]byte("feature"),
		[]byte("config"),
		[]byte("registry"),
		[]byte("reg_caddr"),
		[]byte("reg_crec"),
	}
	// Native contracts deployed to Basechain
	contracts := []contractPrefix{
		{Name: "address-mapper", Prefix: contractAddressToPrefix("0xb9fA0896573A89cF4065c43563C069b3B3C15c37")},
		{Name: "coin", Prefix: contractAddressToPrefix("0xe288d6eec7150D6a22FDE33F0AA2d81E06591C4d")},
		{Name: "ethcoin", Prefix: contractAddressToPrefix("0xde28fb974f31dFbe759cFcB3d1D44C2eeFDFaDd1")},
		{Name: "dposV1", Prefix: contractAddressToPrefix("0x01D10029c253fA02D76188b84b5846ab3D19510D")},
		{Name: "dposV2", Prefix: contractAddressToPrefix("0x35754161AC4Bfa2A20eacf0EfB0f26CBdC418112")},
		{Name: "dposV3", Prefix: contractAddressToPrefix("0xC72783049049c3D887A85dF8061f3141E2C931Cc")},
		{Name: "gateway", Prefix: contractAddressToPrefix("0xC5d1847a03dA59407F27f8FE7981D240bff2dfD3")},
		{Name: "loomcoin-gateway", Prefix: contractAddressToPrefix("0xbC968be1656396E568736D5a8E364ac8Ca430B43")},
		{Name: "tron-gateway", Prefix: contractAddressToPrefix("0x4Dc5C9Cee0827630039Db5E59dfB18e5c679201c")},
		{Name: "binance-gateway", Prefix: contractAddressToPrefix("0x7E0DF5C9fF8898F0e1B4Af4D133Ef557A0641AA8")},
		{Name: "bsc-gateway", Prefix: contractAddressToPrefix("0x3125ca7E54f096A7898A8E14471b281581231724")},
		{Name: "deployer-whitelist", Prefix: contractAddressToPrefix("0xe06AbE129e3fE698bbAB7E3185C798fa2b1a7A50")},
		{Name: "user-deployer-whitelist", Prefix: contractAddressToPrefix("0x278A1C914c046E2d085a84Ee373091a4FB6e19F4")},
		{Name: "chain-config", Prefix: contractAddressToPrefix("0x938312E21AC551251Bd9fCC5dFaa7A5278302339")},
	}
	cmd := &cobra.Command{
		Use:   "analyze <path/to/app.db>",
		Short: "Analyze how much space is taken up by data under the standard key prefixes",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, contract := range contracts {
				prefixes = append(prefixes, contract.Prefix)
			}

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
					for _, c := range contracts {
						if bytes.Compare(c.Prefix, []byte(prefix)) == 0 {
							stats[c.Name] = stat
							sortedPrefixes = append(sortedPrefixes, c.Name)
							break
						}
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
