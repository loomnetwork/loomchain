// +build evm

package gateway

import (
	"encoding/hex"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"

	"fmt"
	"strings"

	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"

	"github.com/ethereum/go-ethereum/ethclient"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/go-loom/client"
	am "github.com/loomnetwork/go-loom/client/address_mapper"
	"github.com/loomnetwork/go-loom/client/dposv2"
	gw "github.com/loomnetwork/go-loom/client/gateway"
	"github.com/loomnetwork/go-loom/client/native_coin"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const GatewayName = "gateway"
const LoomGatewayName = "loomcoin-gateway"

const getOraclesCmdExample = `
./loom gateway get-oracles gateway --key path/to/loom_priv.key
`

const getStateCmdExample = `
./loom gateway get-state gateway --key path/to/loom_priv.key
`

const addOracleCmdExample = `
./loom gateway add-oracle <owner hex address> gateway --key path/to/loom_priv.key
`

const removeOracleCmdExample = `
./loom gateway remove-oracle <owner hex address> gateway --key path/to/loom_priv.key
`

const replaceOwnerCmdExample = `
./loom gateway replace-owner <owner hex address> gateway --key path/to/loom_priv.key
`

const withdrawFundsCmdExample = `
./loom gateway withdraw-funds -u http://plasma.dappchains.com:80 --chain default --key path/to/loom_priv.key OR
./loom gateway withdraw-funds -u http://plasma.dappchains.com:80 --chain default --hsm path/to/hsm.json
`

func newReplaceOwnerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "replace-owner <new-owner> <gateway-name>",
		Short:   "Replaces gateway owner. Only callable by current gateway owner",
		Example: replaceOwnerCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			loomKeyPath := gatewayCmdFlags.PrivKeyPath
			hsmPath := gatewayCmdFlags.HSMConfigPath
			algo := gatewayCmdFlags.Algo
			signer, err := cli.GetSigner(loomKeyPath, hsmPath, algo)
			if err != nil {
				return err
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
				return errors.New("Invalid gateway name")
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
	return cmd
}

func newRemoveOracleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove-oracle <oracle-address> <gateway-name>",
		Short:   "Removes an oracle. Only callable by current gateway owner",
		Example: removeOracleCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			loomKeyPath := gatewayCmdFlags.PrivKeyPath
			hsmPath := gatewayCmdFlags.HSMConfigPath
			algo := gatewayCmdFlags.Algo
			signer, err := cli.GetSigner(loomKeyPath, hsmPath, algo)
			if err != nil {
				return err
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
				return errors.New("Invalid gateway name")
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
	return cmd
}

func newAddOracleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add-oracle <oracle-address> <gateway-name>",
		Short:   "Adds an oracle. Only callable by current gateway owner",
		Example: addOracleCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			loomKeyPath := gatewayCmdFlags.PrivKeyPath
			hsmPath := gatewayCmdFlags.HSMConfigPath
			algo := gatewayCmdFlags.Algo
			signer, err := cli.GetSigner(loomKeyPath, hsmPath, algo)
			if err != nil {
				return err
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
				return errors.New("Invalid gateway name")
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
	return cmd
}

func newGetStateCommand() *cobra.Command {
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
				return errors.New("Invalid gateway name")
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
	return cmd
}

func newGetOraclesCommand() *cobra.Command {
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
				return errors.New("Invalid gateway name")
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
	return cmd
}

func newWithdrawFundsToMainnetCommand() *cobra.Command {
	var onlyRewards bool
	cmd := &cobra.Command{
		Use:     "withdraw-funds",
		Short:   "Withdraw your rewards to mainnet. Process: First claims any unclaimed rewards of a user, then it deposits the user's funds to the dappchain gateway, which provides the user with a signature that's used for transferring funds to Ethereum. The user is prompted to make the call by being provided with the full transaction data that needs to be pasted to the browser.",
		Example: withdrawFundsCmdExample,
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

			mainnetLoomAddress := "0xa4e8c3ec456107ea67d3075bf9e3df3a75823db0"
			mainnetGatewayAddress := "0x8f8E8b3C4De76A31971Fe6a87297D8f703bE8570"
			ethereumUri := "https://mainnet.infura.io/"
			privateKeyPath := gatewayCmdFlags.PrivKeyPath
			hsmPath := gatewayCmdFlags.HSMConfigPath
			algo := gatewayCmdFlags.Algo

			signer, err := cli.GetSigner(privateKeyPath, hsmPath, algo)
			if err != nil {
				return err
			}

			// Create identity with an ephemeral mainnet key since we're going to use ledger
			ephemKey, err := crypto.GenerateKey()
			if err != nil {
				return err
			}
			id, err := client.CreateIdentity(ephemKey, signer, "default")
			if err != nil {
				return err
			}

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

			addressMapper, err := am.ConnectToDAppChainAddressMapper(rpcClient)
			if err != nil {
				return err
			}

			ethClient, err := ethclient.Dial(ethereumUri)
			if err != nil {
				return err
			}

			mainnetGateway, err := gw.ConnectToMainnetGateway(ethClient, mainnetGatewayAddress)
			if err != nil {
				return err
			}

			// Prompt the user to withdraw from a specific account:
			ethAddr, err := addressMapper.GetMappedAccount(id.LoomAddr)
			if err != nil {
				return err
			}

			balanceBefore, err := loomcoin.BalanceOf(id)
			if err != nil {
				return err
			}
			fmt.Println("User balance before:", balanceBefore)

			unclaimedRewards, err := dpos.CheckDistributions(id)
			if err != nil {
				return err
			}
			fmt.Println("Unclaimed rewards:", unclaimedRewards)

			balanceAfter := balanceBefore
			if unclaimedRewards != nil {
				resp, err := dpos.ClaimRewards(id, id.LoomAddr)
				if err != nil {
					return err
				}
				fmt.Println("Claimed rewards:", resp)

				balanceAfter, err = loomcoin.BalanceOf(id)
				if err != nil {
					return err
				}
				fmt.Println("User balance after:", balanceAfter)
			}

			gatewayAddr, err := rpcClient.Resolve("loomcoin-gateway")
			if err != nil {
				return errors.Wrap(err, "failed to resolve DAppChain Gateway address")
			}

			receipt, err := gateway.WithdrawalReceipt(id)
			if err != nil {
				return err
			}

			if receipt == nil {
				var amount *big.Int
				if onlyRewards {
					amount = unclaimedRewards
				} else {
					amount = balanceAfter
				}
				fmt.Println("No pending withdrwal found...")
				// Approve
				err = loomcoin.Approve(id, gatewayAddr, amount)
				if err != nil {
					return err
				}

				fmt.Println("Approved deposit on dappchain for ...", amount)

				// Get the loom tokens to the gateway
				err = gateway.WithdrawLoom(id, amount, common.HexToAddress(mainnetLoomAddress))
				if err != nil {
					return err
				}

				fmt.Println("Withdrawal initiated for...", amount)
			}

			for {
				// Get the receipt
				receipt, err := gateway.WithdrawalReceipt(id)
				if err != nil {
					return err
				}

				if receipt != nil && receipt.OracleSignature != nil {
					break
				}

				fmt.Println("Waiting for receipt...")
				time.Sleep(2 * time.Second)

			}

			fmt.Println("\nGot withdrawal receipt!")
			receipt, err = gateway.WithdrawalReceipt(id) // need to get the receipt again
			if err != nil {
				return err
			}
			fmt.Println("Receipt owner:", receipt.TokenOwner.Local.String())
			fmt.Println("Token Contract:", receipt.TokenContract.Local.String())
			fmt.Println("Token Kind:", receipt.TokenKind)
			fmt.Println("Token Amount:", receipt.TokenAmount.Value.Int)
			fmt.Println("Oracle Sig", hex.EncodeToString(receipt.OracleSignature))

			sig := receipt.OracleSignature

			tx, err := mainnetGateway.UnsignedWithdrawERC20(id, receipt.TokenAmount.Value.Int, sig, common.HexToAddress(mainnetLoomAddress))
			if err != nil {
				return err
			}

			fmt.Println("\nPlease go to https://www.myetherwallet.com/interface/send-offline. Fill the 'To Address', 'GasLimit and 'Data' fields with the values prompted below")
			fmt.Println("To Address:", tx.To().String())
			fmt.Println("Data:", hex.EncodeToString(tx.Data()))
			fmt.Println("Gas Limit:", tx.Gas())
			fmt.Println("Sign it with the account", ethAddr.Local.String(), "and it will authorize a LOOM token withdrawal to you.")

			return nil

		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.BoolVar(&onlyRewards, "only-rewards", false, "Withdraw only the rewards from the gatewy to mainnet if set to true. If false (default), it'll try to claim rewards and then withdraw the whole user balance")
	return cmd
}

func formatJSON(pb proto.Message) (string, error) {
	marshaler := jsonpb.Marshaler{
		Indent:       "  ",
		EmitDefaults: true,
	}
	return marshaler.MarshalToString(pb)
}
