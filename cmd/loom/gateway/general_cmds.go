// +build evm

package gateway

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"

	"fmt"
	"strings"

	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"

	"github.com/ethereum/go-ethereum/ethclient"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/go-loom/client"
	"github.com/loomnetwork/go-loom/client/dposv2"
	gw "github.com/loomnetwork/go-loom/client/gateway"
	"github.com/loomnetwork/go-loom/client/native_coin"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const GatewayName = "gateway"
const LoomGatewayName = "loomcoin-gateway"

const getOraclesCmdExample = `
./loom gateway get-oracles gateway --key file://path/to/loom_priv.key
`

const getStateCmdExample = `
./loom gateway get-state gateway --key file://path/to/loom_priv.key
`

const addOracleCmdExample = `
./loom gateway add-oracle <owner hex address> gateway --key file://path/to/loom_priv.key
`

const removeOracleCmdExample = `
./loom gateway remove-oracle <owner hex address> gateway --key file://path/to/loom_priv.key
`

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

func newRemoveOracleCommand() *cobra.Command {
	var loomKeyStr string
	cmd := &cobra.Command{
		Use:     "remove-oracle <oracle-address> <gateway-name>",
		Short:   "Removes an oracle. Only callable by current gateway owner",
		Example: removeOracleCmdExample,
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

			req := &tgtypes.TransferGatewayRemoveOracleRequest{
				Oracle: oracleAddress.MarshalPB(),
			}

			_, err = gateway.Call("RemoveOracle", req, signer, nil)
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
		Short:   "Adds an oracle. Only callable by current gateway owner",
		Example: addOracleCmdExample,
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

func newGetStateCommand() *cobra.Command {
	var loomKeyStr string
	cmd := &cobra.Command{
		Use:     "get-state <gateway-name>",
		Short:   "Queries the gateway's state",
		Example: getStateCmdExample,
		Args:    cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(args) == 0 || (strings.Compare(args[0], GatewayName) == 0) {
				name = GatewayName
			} else if strings.Compare(args[0], LoomGatewayName) == 0 {
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

			req := &tgtypes.TransferGatewayStateRequest{}
			resp := &tgtypes.TransferGatewayStateResponse{}
			_, err = gateway.StaticCall("GetState", req, gatewayAddr, resp)
			fmt.Println(formatJSON(resp))
			return err
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.StringVarP(&loomKeyStr, "key", "k", "", "DAppChain private key of contract creator")
	cmd.MarkFlagRequired("key")
	return cmd
}

func newGetOraclesCommand() *cobra.Command {
	var loomKeyStr string
	cmd := &cobra.Command{
		Use:     "get-oracles <gateway-name>",
		Short:   "Queries the gateway's state",
		Example: getOraclesCmdExample,
		Args:    cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(args) == 0 || (strings.Compare(args[0], GatewayName) == 0) {
				name = GatewayName
			} else if strings.Compare(args[0], LoomGatewayName) == 0 {
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

			req := &tgtypes.TransferGatewayGetOraclesRequest{}
			resp := &tgtypes.TransferGatewayGetOraclesResponse{}
			_, err = gateway.StaticCall("GetOracles", req, gatewayAddr, resp)
			oracles := resp.Oracles
			for i, oracle := range oracles {
				fmt.Printf("Oracle %d: %s\n", i, loom.UnmarshalAddressPB(oracle.Address))
			}
			return err
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.StringVarP(&loomKeyStr, "key", "k", "", "DAppChain private key of contract creator")
	cmd.MarkFlagRequired("key")
	return cmd
}

func newWithdrawRewardsToMainnetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "withdraw-rewards",
		Short:   "Links a DAppChain account to an Ethereum account via the Transfer Gateway. Requires interaction for the user to provide the ethereum signature instead of doing it in the node.",
		Example: mapAccountsCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {

			/**
			 * 1 Check dappchain balance before
			 * 2. Claim rewards on dappchain
			 * 3. Check balance aftr (should be bigger)
			 * 4. Call approve transactino on dappchain
			 * 5. Call withdrawLoom transaction on the dappchain
			 * 6. Check dappchain balance, check dappchain gateway balance
			 * 7. Check account receipt
			 * 8. Create unsigned transaction and print it. GG:)
			 */

			cmdFlags := cmd.Flags()

			mainnetLoomAddress := "0xa4e8c3ec456107ea67d3075bf9e3df3a75823db0"
			mainnetGatewayAddress := "0x8f8E8b3C4De76A31971Fe6a87297D8f703bE8570"
			ethereumUri := "https://mainnet.infura.io/"
			privateKeyPath, _ := cmdFlags.GetString("key")
			hsmPath, _ := cmdFlags.GetString("hsm")
			algo, _ := cmdFlags.GetString("algo")

			// HACK: Create a dummy identity and overwrite its signer
			// ETH Key isn't utilized for anything.
			ethKey := "0x60f4fd8797df0a5a391618d9f3e67f2ba77ac53eb511b2e935cfccbf8079b465"
			dappchainKey := "wbbTq5dsaI26X6ddDlj5OeAD47Ib1S+ie1eojTjVTBEoTQdKVYb/gDyrfpKVSxTScQfUVhy2ytwPRJ86uIBejA=="
			id, err := client.CreateIdentity("dummy", ethKey, dappchainKey, "default")
			if err != nil {
				return err
			}

			signer, err := cli.GetSigner(privateKeyPath, hsmPath, algo)
			if err != nil {
				return err
			}

			id.LoomSigner = signer

			// fmt.Println("SIGNER ADDRESS", loom.LocalAddressFromPublicKey(id.LoomSigner.PublicKey()))

			rpcClient := getDAppChainClient()
			loomcoin, err := native_coin.ConnectToDAppChainLoomContract(rpcClient)
			if err != nil {
				return err
			}

			// Connect to DPOS - REPLACE ALL DPOS IDENTITIES WITH SIGNERS
			dpos, err := dposv2.ConnectToDAppChainDPOSContract(rpcClient)
			if err != nil {
				return err
			}

			gateway, err := gw.ConnectToDAppChainLoomGateway(rpcClient, "")
			if err != nil {
				return err
			}

			ethClient, err := ethclient.Dial(ethereumUri)
			if err != nil {
				return err
			}

			mainnetGateway, err := gw.ConnectToMainnetGateway(ethClient, mainnetGatewayAddress)

			balanceBefore, err := loomcoin.BalanceOf(id)
			if err != nil {
				return err
			}
			fmt.Println("User balance before:", balanceBefore)

			unclaimedRewards, err := dpos.CheckDistributions(id)

			fmt.Println("Unclaimed rewards:", unclaimedRewards)

			resp, err := dpos.ClaimRewards(id, withdrawalAddr)
			fmt.Println("Claimed rewards:", resp)

			balanceAfter, err := loomcoin.BalanceOf(id)
			if err != nil {
				return err
			}
			fmt.Println("User balance after:", balanceAfter)

			if balanceAfter.Cmp(big.NewInt(0)) == 0 {
				fmt.Println("No rewards to be claimed back to mainnet")
				return nil
			}

			gatewayAddr, err := rpcClient.Resolve("loomcoin-gateway")
			if err != nil {
				return errors.Wrap(err, "failed to resolve DAppChain Gateway address")
			}

			rewards := resp.Amount.Value.Int
			fmt.Println("Claimed", rewards)

			// Approve
			err = loomcoin.Approve(id, gatewayAddr, balanceAfter)
			if err != nil {
				return err
			}

			// Get the loom tokens to the gateway
			err = gateway.WithdrawLoom(id, balanceAfter, common.HexToAddress(mainnetLoomAddress))
			if err != nil {
				return err
			}

			// Get the receipt
			receipt, err := gateway.WithdrawalReceipt(id)
			if err != nil {
				return err
			}
			fmt.Println("Got withdrawal receipt:", receipt)

			sig := receipt.OracleSignature

			tx, err := mainnetGateway.UnsignedWithdrawERC20(id, balanceAfter, sig, common.HexToAddress(mainnetLoomAddress))
			if err != nil {
				return err
			}

			fmt.Println("Please paste the unsigned transaction below to your wallet. Sign it and it will authorize a LOOM token withdrawal to your account.\n", tx)

			return nil

		},
	}
	return cmd
}

func formatJSON(pb proto.Message) (string, error) {
	marshaler := jsonpb.Marshaler{
		Indent:       "  ",
		EmitDefaults: true,
	}
	return marshaler.MarshalToString(pb)
}
