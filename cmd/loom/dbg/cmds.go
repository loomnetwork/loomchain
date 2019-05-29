package dbg

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/client"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/blockchain"
	dbm "github.com/tendermint/tendermint/libs/db"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
	"github.com/tendermint/tmlibs/db"
)

var (
	appHeightKey = []byte("appheight")
)

func newDumpMempoolCommand() *cobra.Command {
	var nodeURI string
	var limit int
	var showExtraInfo bool
	cmd := &cobra.Command{
		Use:     "dump-mempool",
		Short:   "Displays all the txs in a node's mempool (currently limited to first 100)",
		Example: "loom debug dump-mempool --uri http://hostname:port --ext --limit 50",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.NewJSONRPCClient(nodeURI + "/rpc")
			cdc := amino.NewCodec()
			ctypes.RegisterAmino(cdc)
			var rm json.RawMessage
			if showExtraInfo {
				var result ctypes.ResultMempoolTxs
				err := c.Call("mempool_txs", map[string]interface{}{"limit": strconv.Itoa(limit)}, "1", &rm)
				if err != nil {
					return errors.Wrap(err, "failed to call mempool_txs")
				}
				if err := cdc.UnmarshalJSON(rm, &result); err != nil {
					return errors.Wrap(err, "failed to unmarshal rpc response result")
				}
				for _, tx := range result.Txs {
					str, err := decodeMessageTx(tx.Tx)
					if err != nil {
						log.Error("failed to decode tx", "err", err)
					} else {
						fmt.Printf("[h] %8d %s\n", tx.Height, str)
					}
				}
				fmt.Printf("fetched %d/%d txs\n", len(result.Txs), result.N)
			} else {
				var result ctypes.ResultUnconfirmedTxs
				err := c.Call("unconfirmed_txs", map[string]interface{}{"limit": strconv.Itoa(limit)}, "1", &rm)
				if err != nil {
					return errors.Wrap(err, "failed to call unconfirmed_txs")
				}
				if err := cdc.UnmarshalJSON(rm, &result); err != nil {
					return errors.Wrap(err, "failed to unmarshal rpc response result")
				}
				for _, tx := range result.Txs {
					str, err := decodeMessageTx(tx)
					if err != nil {
						log.Error("failed to decode tx", "err", err)
					} else {
						fmt.Println(str)
					}
				}
				fmt.Printf("fetched %d txs\n", result.N)
			}
			return nil
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.StringVarP(&nodeURI, "uri", "u", "http://localhost:46658", "DAppChain base URI")
	cmdFlags.IntVarP(&limit, "limit", "l", 100, "Max number of txs to display")
	cmdFlags.BoolVarP(&showExtraInfo, "ext", "e", false, "Show extra info for each tx")
	return cmd
}

func newDumpBlockTxsCommand() *cobra.Command {
	var nodeURI string
	var height int
	cmd := &cobra.Command{
		Use:     "dump-block-txs",
		Short:   "Displays all the txs in a block",
		Example: "loom dump-block-txs --height 12345 --uri http://host:port",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.NewJSONRPCClient(nodeURI + "/rpc")
			cdc := amino.NewCodec()
			ctypes.RegisterAmino(cdc)
			var rm json.RawMessage

			var result ctypes.ResultBlock
			params := map[string]interface{}{}
			if height > 0 {
				params["height"] = strconv.Itoa(height)
			}
			if err := c.Call("block", params, "1", &rm); err != nil {
				return errors.Wrap(err, "failed to call mempool_txs")
			}
			if err := cdc.UnmarshalJSON(rm, &result); err != nil {
				return errors.Wrap(err, "failed to unmarshal rpc response result")
			}
			for _, tx := range result.Block.Data.Txs {
				str, err := decodeMessageTx(tx)
				if err != nil {
					log.Error("failed to decode tx", "err", err)
				} else {
					fmt.Println(str)
				}
			}
			fmt.Printf("fetched %d txs from block %d\n", len(result.Block.Data.Txs), height)
			return nil
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.StringVarP(&nodeURI, "uri", "u", "http://localhost:46658", "DAppChain base URI")
	cmdFlags.IntVar(&height, "height", 1, "Block height for which txs should be displayed")
	return cmd
}

func newDumpBlockStoreTxsCommand() *cobra.Command {
	var height int64
	cmd := &cobra.Command{
		Use:     "dump-block-store-txs",
		Short:   "Displays all the txs from a block in blockstore.db",
		Example: "loom dump-block-store-txs <path/to/chaindata> --height 12345",
		RunE: func(cmd *cobra.Command, args []string) error {
			blockStoreDB := dbm.NewDB("blockstore", "leveldb", path.Join(args[0], "data"))
			defer blockStoreDB.Close()

			blockStore := blockchain.NewBlockStore(blockStoreDB)
			latestHeight := blockchain.LoadBlockStoreStateJSON(blockStoreDB).Height

			if height > latestHeight {
				return fmt.Errorf(
					"failed to load block at height %d, latest block store height is %d",
					height, latestHeight,
				)
			}

			block := blockStore.LoadBlock(height)
			for _, tx := range block.Txs {
				str, err := decodeMessageTx(tx)
				if err != nil {
					log.Error("failed to decode tx", "err", err)
				} else {
					fmt.Println(str)
				}
			}
			fmt.Printf("Found %d txs in block %d\n", block.NumTxs, height)
			return nil
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.Int64Var(&height, "height", 1, "Block height for which txs should be displayed")
	return cmd
}

func newSetAppHeightCommand() *cobra.Command {
	var height int64
	cmd := &cobra.Command{
		Use:     "set-app-height",
		Short:   "Save a specific height for the next run into app.db",
		Example: "loom debug set-app-height <path/to/app.db> --height 12345",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcDBPath, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("Failed to resolve app.db path '%s'", args[0])
			}
			dbName := strings.TrimSuffix(path.Base(srcDBPath), ".db")
			dbDir := path.Dir(srcDBPath)
			appDB, err := db.NewGoLevelDB(dbName, dbDir)
			if err != nil {
				return err
			}
			defer appDB.Close()
			heightBuffer := make([]byte, 8)
			binary.BigEndian.PutUint64(heightBuffer, uint64(height))
			appDB.SetSync(appHeightKey, heightBuffer)
			return nil
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.Int64Var(&height, "height", 0, "Block height to be loaded for the next run")
	cmd.MarkFlagRequired("height")
	return cmd
}

func newGetAppHeightCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get-app-height",
		Short:   "Get app height to be loaded on the next run from app.db",
		Example: "loom debug get-app-height <path/to/app.db>",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcDBPath, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("Failed to resolve app.db path '%s'", args[0])
			}
			dbName := strings.TrimSuffix(path.Base(srcDBPath), ".db")
			dbDir := path.Dir(srcDBPath)
			appDB, err := db.NewGoLevelDB(dbName, dbDir)
			if err != nil {
				return err
			}
			defer appDB.Close()
			height := appDB.Get(appHeightKey)
			if height == nil {
				return fmt.Errorf("app height not found")
			}
			fmt.Println(binary.BigEndian.Uint64(height))
			return nil
		},
	}
	return cmd
}

func newDeleteAppHeightCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete-app-height",
		Short:   "Delete app height from app.db",
		Example: "loom debug delete-app-height <path/to/app.db>",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcDBPath, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("Failed to resolve app.db path '%s'", args[0])
			}
			dbName := strings.TrimSuffix(path.Base(srcDBPath), ".db")
			dbDir := path.Dir(srcDBPath)
			appDB, err := db.NewGoLevelDB(dbName, dbDir)
			if err != nil {
				return err
			}
			defer appDB.Close()
			height := appDB.Get(appHeightKey)
			if height == nil {
				return fmt.Errorf("app height not found")
			}
			appDB.DeleteSync(appHeightKey)
			return nil
		},
	}
	return cmd
}

// NewDebugCommand creates a new instance of the top-level debug command
func NewDebugCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug <command>",
		Short: "Node Debugging Tools",
	}
	cmd.AddCommand(
		newDumpMempoolCommand(),
		newDumpBlockTxsCommand(),
		newDumpBlockStoreTxsCommand(),
		newSetAppHeightCommand(),
		newGetAppHeightCommand(),
		newDeleteAppHeightCommand(),
	)
	return cmd
}

func decodeMessageTx(tx tmtypes.Tx) (string, error) {
	var signedTx auth.SignedTx
	if err := proto.Unmarshal(tx, &signedTx); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal SignedTx")
	}

	var nonceTx auth.NonceTx
	if err := proto.Unmarshal(signedTx.Inner, &nonceTx); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal NonceTx")
	}

	var loomTx types.Transaction
	if err := proto.Unmarshal(nonceTx.Inner, &loomTx); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal Transaction")
	}

	var msgTx vm.MessageTx
	if err := proto.Unmarshal(loomTx.Data, &msgTx); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal MessageTx")
	}

	var vmType vm.VMType
	if loomTx.Id == 1 {
		var deployTx vm.DeployTx
		if err := proto.Unmarshal(msgTx.Data, &deployTx); err != nil {
			return "", errors.Wrap(err, "failed to unmarshal DeployTx")
		}
		vmType = deployTx.VmType
	} else if loomTx.Id == 2 {
		var callTx vm.CallTx
		if err := proto.Unmarshal(msgTx.Data, &callTx); err != nil {
			return "", errors.Wrap(err, "failed to unmarshal CallTx")
		}
		vmType = callTx.VmType
	}

	var vmName string
	if vmType == vm.VMType_PLUGIN {
		vmName = "go"
	} else if vmType == vm.VMType_EVM {
		vmName = "evm"
	}

	return fmt.Sprintf(
		"[txh] %X [sndr] %s [n] %5d [tid] %d [to] %s [vm] %s",
		tx.Hash(),
		loom.UnmarshalAddressPB(msgTx.From).String(),
		nonceTx.Sequence,
		loomTx.Id,
		loom.UnmarshalAddressPB(msgTx.To).String(),
		vmName,
	), nil
}
