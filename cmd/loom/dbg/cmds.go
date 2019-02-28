package dbg

import (
	"encoding/json"
	"fmt"
	"strconv"

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
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

func newDumpMempoolCommand() *cobra.Command {
	var nodeURI string
	var limit int
	cmd := &cobra.Command{
		Use:   "dump-mempool",
		Short: "Displays all the txs in a node's mempool (currently limited to first 100)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.NewJSONRPCClient(nodeURI + "/rpc")
			params := map[string]interface{}{}
			params["limit"] = strconv.Itoa(limit)
			var rm json.RawMessage
			var result ctypes.ResultUnconfirmedTxs
			if err := c.Call("unconfirmed_txs", params, "1", &rm); err != nil {
				return err
			}
			cdc := amino.NewCodec()
			ctypes.RegisterAmino(cdc)
			if err := cdc.UnmarshalJSON(rm, &result); err != nil {
				return errors.Errorf("failed to unmarshal rpc response result: %v", err)
			}

			for _, tx := range result.Txs {
				str, err := decodeMessageTx(tx)
				if err != nil {
					log.Error("failed to decode tx", "err", err)
				} else {
					fmt.Println(str)
				}
			}
			fmt.Printf("mempool size: %d\n", result.N)
			return nil
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.StringVarP(&nodeURI, "uri", "u", "http://localhost:46658", "DAppChain base URI")
	cmdFlags.IntVarP(&limit, "limit", "l", 100, "Max number of txs to display")
	return cmd
}

// NewDebugCommand creates a new instance of the top-level debug command
func NewDebugCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug <command>",
		Short: "Node Debugging Tools",
	}
	cmd.AddCommand(newDumpMempoolCommand())
	return cmd
}

func decodeMessageTx(txBytes tmtypes.Tx) (string, error) {
	var signedTx auth.SignedTx
	if err := proto.Unmarshal(txBytes, &signedTx); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal SignedTx")
	}

	var nonceTx auth.NonceTx
	if err := proto.Unmarshal(signedTx.Inner, &nonceTx); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal NonceTx")
	}

	var tx types.Transaction
	if err := proto.Unmarshal(nonceTx.Inner, &tx); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal Transaction")
	}

	var msgTx vm.MessageTx
	if err := proto.Unmarshal(tx.Data, &msgTx); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal MessageTx")
	}

	var vmType vm.VMType
	if tx.Id == 1 {
		var deployTx vm.DeployTx
		if err := proto.Unmarshal(msgTx.Data, &deployTx); err != nil {
			return "", errors.Wrap(err, "failed to unmarshal DeployTx")
		}
		vmType = deployTx.VmType
	} else if tx.Id == 2 {
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
		txBytes.Hash(),
		loom.UnmarshalAddressPB(msgTx.From).String(),
		nonceTx.Sequence,
		tx.Id,
		loom.UnmarshalAddressPB(msgTx.To).String(),
		vmName,
	), nil
}
