// +build evm

package gateway

import (
	"strings"

	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"

	"github.com/loomnetwork/go-loom/client"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const GatewayName = "gateway"
const LoomGatewayName = "loomcoin-gateway"

const replaceOwnerCmdExample = `
./loom gateway replace-owner <owner hex address> gateway --key file://path/to/loom_priv.key
`

func newReplaceOwnerCommand() *cobra.Command {
	var loomKeyStr string
	cmd := &cobra.Command{
		Use:     "replace-owner <new-owner> <gateway-name>",
		Short:   "Replaces gateway owner. Only callable by current gateway owner",
		Example: replaceOwnerCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			signer, err := getDAppChainSigner(loomKeyStr)
			if err != nil {
				return errors.Wrap(err, "failed to load creator DAppChain key")
			}

			newOwner, err := hexToLoomAddress(args[0])
			if err != nil {
				return errors.Wrap(err, "failed to add new owner")
			}

			var name string
			if len(args) <= 1 || (strings.Compare(args[1], GatewayName) == 0) {
				name = GatewayName
			} else if strings.Compare(args[1], LoomGatewayName) == 0 {
				name = LoomGatewayName
			} else {
				errors.New("Invalid gateway name")
			}

			rpcClient := getDAppChainClient()
			gatewayAddr, err := rpcClient.Resolve(name)
			if err != nil {
				return errors.Wrap(err, "failed to resolve DAppChain Gateway address")
			}
			gateway := client.NewContract(rpcClient, gatewayAddr.Local)

			req := &tgtypes.TransferGatewayAddOracleRequest{
				Oracle: newOwner.MarshalPB(),
			}

			_, err = gateway.Call("ReplaceOwner", req, signer, nil)
			return err
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.StringVarP(&loomKeyStr, "key", "k", "", "DAppChain private key of contract creator")
	cmd.MarkFlagRequired("key")
	return cmd
}

func newAddOracleCommand() *cobra.Command {
	var loomKeyStr string
	cmd := &cobra.Command{
		Use:     "add-oracle <oracle-address> <gateway-name>",
		Short:   "Replaces gateway owner. Only callable by current gateway owner",
		Example: replaceOwnerCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			signer, err := getDAppChainSigner(loomKeyStr)
			if err != nil {
				return errors.Wrap(err, "failed to load creator DAppChain key")
			}

			oracleAddress, err := hexToLoomAddress(args[0])
			if err != nil {
				return errors.Wrap(err, "failed to add new owner")
			}

			var name string
			if len(args) <= 1 || (strings.Compare(args[1], GatewayName) == 0) {
				name = GatewayName
			} else if strings.Compare(args[1], LoomGatewayName) == 0 {
				name = LoomGatewayName
			} else {
				errors.New("Invalid gateway name")
			}

			rpcClient := getDAppChainClient()
			gatewayAddr, err := rpcClient.Resolve(name)
			if err != nil {
				return errors.Wrap(err, "failed to resolve DAppChain Gateway address")
			}
			gateway := client.NewContract(rpcClient, gatewayAddr.Local)

			req := &tgtypes.TransferGatewayAddOracleRequest{
				Oracle: oracleAddress.MarshalPB(),
			}

			_, err = gateway.Call("AddOracle", req, signer, nil)
			return err
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.StringVarP(&loomKeyStr, "key", "k", "", "DAppChain private key of contract creator")
	cmd.MarkFlagRequired("key")
	return cmd
}

