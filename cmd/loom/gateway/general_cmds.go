// +build evm

package gateway

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	ssha "github.com/miguelmota/go-solidity-sha3"

	"fmt"
	"strconv"
	"strings"

	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/go-loom/types"

	"github.com/ethereum/go-ethereum/ethclient"
	loom "github.com/loomnetwork/go-loom"
	dpostypes "github.com/loomnetwork/go-loom/builtin/types/dposv3"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/go-loom/client"
	am "github.com/loomnetwork/go-loom/client/address_mapper"
	"github.com/loomnetwork/go-loom/client/dposv3"
	gw "github.com/loomnetwork/go-loom/client/gateway"
	"github.com/loomnetwork/go-loom/client/native_coin"
	lcrypto "github.com/loomnetwork/go-loom/crypto"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	GatewayName        = "gateway"
	LoomGatewayName    = "loomcoin-gateway"
	BinanceGatewayName = "binance-gateway"
	TronGatewayName    = "tron-gateway"
)

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
./loom gateway withdraw-funds \
	--eth-uri https://mainnet.infura.io/v3/<project_id> \
	-u http://plasma.dappchains.com:80 --chain default --hsm path/to/hsm.json
`

const setWithdrawFeeCmdExample = `
./loom gateway set-withdraw-fee 37500 binance-gateway --key path/to/loom_priv.key
`

const setWithdrawLimitCmdExample = `
./loom gateway set-withdrawal-limit gateway --total-limit 1000000 --account-limit 500000 --key path/to/loom_priv.key
./loom gateway set-withdrawal-limit loomcoin-gateway --total-limit 1000000 --account-limit 500000 --key path/to/loom_priv.key
./loom gateway set-withdrawal-limit binance-gateway --total-limit 1000000 --account-limit 500000 --decimals 8 --key path/to/loom_priv.key
`

const updateMainnetAddressCmdExample = `
./loom gateway update-mainnet-address <mainnet-hex-address> gateway --key path/to/loom_priv.key
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

func newUpdateTrustedValidatorsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update-trusted-validators <trusted-validators-file> <gateway-name>",
		Short:   "Updates the trusted validators which can submit signatures to the gateway",
		Example: "loom gateway update-trusted-validators /path/to/trusted_validators_file loomcoin-gateway",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			loomKeyPath := gatewayCmdFlags.PrivKeyPath
			hsmPath := gatewayCmdFlags.HSMConfigPath
			algo := gatewayCmdFlags.Algo
			signer, err := cli.GetSigner(loomKeyPath, hsmPath, algo)
			if err != nil {
				return err
			}

			file, err := os.Open(args[0])
			if err != nil {
				return err
			}
			defer file.Close()

			var validators []*loom.Address
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				pubKey, err := base64.StdEncoding.DecodeString(scanner.Text())
				if err != nil {
					return err
				}

				validators = append(
					validators,
					&loom.Address{
						ChainID: gatewayCmdFlags.ChainID,
						Local:   loom.LocalAddressFromPublicKey(pubKey),
					},
				)
			}

			if err := scanner.Err(); err != nil {
				return err
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

			trustedVals := make([]*types.Address, len(validators))
			for i, v := range validators {
				trustedVals[i] = v.MarshalPB()
			}

			trustedValidators := tgtypes.TransferGatewayTrustedValidators{
				Validators: trustedVals,
			}

			req := &tgtypes.TransferGatewayUpdateTrustedValidatorsRequest{
				TrustedValidators: &trustedValidators,
			}

			_, err = gateway.Call("UpdateTrustedValidators", req, signer, nil)
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
			} else if strings.Compare(args[0], BinanceGatewayName) == 0 {
				name = BinanceGatewayName
			} else if strings.Compare(args[0], TronGatewayName) == 0 {
				name = TronGatewayName
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
	var ethURI string
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
			dpos, err := dposv3.ConnectToDAppChainDPOSContract(rpcClient)
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

			ethClient, err := ethclient.Dial(ethURI)
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

			unclaimedRewards, err := dpos.CheckRewardsFromAllValidators(id, id.LoomAddr)
			if err != nil {
				return err
			}
			fmt.Println("Unclaimed rewards:", unclaimedRewards)

			balanceAfter := balanceBefore
			if unclaimedRewards != nil {
				resp, err := dpos.ClaimRewardsFromAllValidators(id)
				if err != nil {
					return err
				}
				fmt.Println("Started claiming of rewards:", resp)

				// Need to wait until the rewards delegation is unbonded.
				timeToElections, err := dpos.TimeUntilElections(id)
				if err != nil {
					return err
				}
				fmt.Println("Time until elections: ", timeToElections)

				sleepTime := int64(30)
				for {
					timeToElections, err := dpos.TimeUntilElections(id)
					if err != nil {
						return err
					}

					fmt.Println("Sleeping...")
					if timeToElections < sleepTime {
						time.Sleep(time.Duration(timeToElections) * time.Second)
					} else {
						time.Sleep(time.Duration(sleepTime) * time.Second)
					}

					fmt.Println("Time until elections: ", timeToElections)

					// Get delegation state after we slept
					rewardsDelegation, err := dpos.GetRewardsDelegation(id, id.LoomAddr)
					if err != nil {
						return err
					}

					// Stop sleeping after the delegation has been unbonded
					if rewardsDelegation.State == dpostypes.Delegation_BONDED {
						break
					}
				}

				fmt.Println("Rewards have been claimed.")

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
	cmdFlags.StringVar(&ethURI, "eth-uri", "https://mainnet.infura.io/v3/a5a5151fecba45229aa77f0725c10241", "Ethereum URI")
	return cmd
}

func newCreateWithdrawalReceipt() *cobra.Command {
	var ethPrivateKeyPath, amountStr, mainnetGatewayAddr, withdrawerAddr, mainnetTokenAddr string
	var nonce int
	cmd := &cobra.Command{
		Use:   "create-withdrawal-receipt",
		Short: "Create a withdrawal receipt",
		RunE: func(cmd *cobra.Command, args []string) error {
			if mainnetGatewayAddr == "" {
				return errors.New("Ethereum Gateway address must be specified")
			}
			privKey, err := lcrypto.LoadSecp256k1PrivKey(ethPrivateKeyPath)
			if err != nil {
				return errors.Wrap(err, "failed to load Ethereum private key")
			}
			amount, _ := big.NewInt(0).SetString(amountStr, 10)
			var hash []byte
			if mainnetTokenAddr != "" {
				hash = ssha.SoliditySHA3(
					ssha.Uint256(amount),
					ssha.Address(common.HexToAddress(mainnetTokenAddr)),
				)
			} else { // ETH withdrawals don't need a contract address
				hash = ssha.SoliditySHA3(ssha.Uint256(amount))
			}
			hash = ssha.SoliditySHA3(
				ssha.Address(common.HexToAddress(withdrawerAddr)),
				ssha.Uint256(new(big.Int).SetUint64(uint64(nonce))),
				ssha.Address(common.HexToAddress(mainnetGatewayAddr)),
				hash,
			)
			sig, err := lcrypto.SoliditySign(hash, privKey)
			if err != nil {
				return errors.Wrap(err, "failed to sign withdrawal receipt")
			}
			sig = append(make([]byte, 1, 66), sig...)
			type receipt struct {
				Withdrawer string
				Amount     string
				Signature  string
			}
			r, err := json.MarshalIndent(&receipt{
				Withdrawer: withdrawerAddr,
				Amount:     amountStr,
				Signature:  hex.EncodeToString(sig),
			}, "", "  ")
			if err != nil {
				return errors.Wrap(err, "failed to marshal receipt")
			}
			fmt.Println(string(r))
			return nil
		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.StringVar(&ethPrivateKeyPath, "eth-key", "", "Path to file containing Ethereum private key for signing")
	cmdFlags.StringVar(&amountStr, "amount", "0", "Amount to withdraw")
	cmdFlags.IntVar(&nonce, "nonce", 0, "Withdrawal nonce")
	cmdFlags.StringVar(&mainnetGatewayAddr, "mainnet-gateway", "", "Address of Ethereum Gateway to withdraw from")
	cmdFlags.StringVar(&withdrawerAddr, "withdrawer", "", "Ethereum address of account to withdraw to")
	cmdFlags.StringVar(&mainnetTokenAddr, "token", "", "Ethereum token contract address")
	return cmd
}

func formatJSON(pb proto.Message) (string, error) {
	marshaler := jsonpb.Marshaler{
		Indent:       "  ",
		EmitDefaults: true,
	}
	return marshaler.MarshalToString(pb)
}

func newSetWithdrawFeeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "set-withdraw-fee <fee> [gateway]",
		Short:   "Sets the fee the gateway should charge per withdrawal",
		Example: setWithdrawFeeCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			loomKeyPath := gatewayCmdFlags.PrivKeyPath
			hsmPath := gatewayCmdFlags.HSMConfigPath
			algo := gatewayCmdFlags.Algo
			signer, err := cli.GetSigner(loomKeyPath, hsmPath, algo)
			if err != nil {
				return err
			}

			var name string
			if strings.Compare(args[1], BinanceGatewayName) == 0 {
				name = BinanceGatewayName
			} else {
				return errors.New("only Binance gateway has withdrawal fees.")
			}

			var transferFee *types.BigUInt
			fee, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return err
			}

			if fee >= 0 {
				transferFee = &types.BigUInt{Value: *loom.NewBigUIntFromInt(fee)}
			} else {
				return errors.New("Invalid fee argument")
			}

			rpcClient := getDAppChainClient()
			gatewayAddr, err := rpcClient.Resolve(name)
			if err != nil {
				return errors.Wrap(err, "failed to resolve DAppChain Gateway address")
			}
			gateway := client.NewContract(rpcClient, gatewayAddr.Local)

			req := &tgtypes.TransferGatewayUpdateBinanceTransferFeeRequest{
				TransferFee: transferFee,
			}

			_, err = gateway.Call("SetTransferFee", req, signer, nil)
			if err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func newSetWithdrawLimitCommand() *cobra.Command {
	var totalLimit, accountLimit, decimals uint64
	cmd := &cobra.Command{
		Use:     "set-withdrawal-limit <gateway>",
		Short:   "Sets maximum ETH or LOOM amount the gateway should allow users to withdraw per day",
		Example: setWithdrawLimitCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			loomKeyPath := gatewayCmdFlags.PrivKeyPath
			hsmPath := gatewayCmdFlags.HSMConfigPath
			algo := gatewayCmdFlags.Algo
			signer, err := cli.GetSigner(loomKeyPath, hsmPath, algo)
			if err != nil {
				return err
			}

			var name string
			if len(args) == 0 || strings.EqualFold(args[0], GatewayName) {
				name = GatewayName
			} else if strings.EqualFold(args[0], LoomGatewayName) {
				name = LoomGatewayName
			} else if strings.EqualFold(args[0], BinanceGatewayName) {
				name = BinanceGatewayName
			} else {
				return fmt.Errorf("withdrawal limits not supported by %s", name)
			}

			// create amounts with decimals
			maxTotalAmount := sciNot(int64(totalLimit), int64(decimals))
			maxPerAccountAmount := sciNot(int64(accountLimit), int64(decimals))

			rpcClient := getDAppChainClient()
			gatewayAddr, err := rpcClient.Resolve(name)
			if err != nil {
				return errors.Wrap(err, "failed to resolve DAppChain Gateway address")
			}
			gateway := client.NewContract(rpcClient, gatewayAddr.Local)

			// fetch the current limits
			stateReq := &tgtypes.TransferGatewayStateRequest{}
			stateResp := &tgtypes.TransferGatewayStateResponse{}
			if _, err := gateway.StaticCall("GetState", stateReq, gatewayAddr, stateResp); err != nil {
				return errors.Wrap(err, "failed to fetch current withdrawal limits")
			}

			req := &tgtypes.TransferGatewaySetMaxWithdrawalLimitRequest{
				MaxTotalDailyWithdrawalAmount:      &types.BigUInt{Value: *maxTotalAmount},
				MaxPerAccountDailyWithdrawalAmount: &types.BigUInt{Value: *maxPerAccountAmount},
			}

			state := stateResp.State
			// Unless a new non-zero limit was provided keep the existing limit
			if totalLimit == 0 {
				req.MaxTotalDailyWithdrawalAmount = state.MaxTotalDailyWithdrawalAmount
			}
			if accountLimit == 0 {
				req.MaxPerAccountDailyWithdrawalAmount = state.MaxPerAccountDailyWithdrawalAmount
			}

			if _, err = gateway.Call("SetMaxWithdrawalLimit", req, signer, nil); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().Uint64Var(
		&totalLimit, "total-limit", 0,
		"Max total amount the gateway should allow to be withdrawn per day",
	)
	cmd.Flags().Uint64Var(
		&accountLimit, "account-limit", 0,
		"Max total amount the gateway should allow to be withdrawn per day by any one account",
	)
	cmd.Flags().Uint64Var(
		&decimals, "decimals", 18,
		"The number of decimals to append to input amounts",
	)
	return cmd
}

func newUpdateMainnetGatewayAddressCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update-mainnet-address <mainnet-address> <gateway-name>",
		Short:   "Update mainnet gateway address. Only callable by current gateway owner",
		Example: updateMainnetAddressCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			loomKeyPath := gatewayCmdFlags.PrivKeyPath
			hsmPath := gatewayCmdFlags.HSMConfigPath
			algo := gatewayCmdFlags.Algo
			signer, err := cli.GetSigner(loomKeyPath, hsmPath, algo)
			if err != nil {
				return err
			}

			var hexAddr string
			var name string
			var foreignChainId string

			if len(args) <= 1 || strings.EqualFold(args[1], GatewayName) {
				name = GatewayName
				foreignChainId = "eth"
			} else if strings.EqualFold(args[1], LoomGatewayName) {
				name = LoomGatewayName
				foreignChainId = "eth"
			} else if strings.EqualFold(args[1], BinanceGatewayName) {
				name = BinanceGatewayName
				foreignChainId = "binance"
			} else if strings.EqualFold(args[1], TronGatewayName) {
				name = TronGatewayName
				foreignChainId = "tron"
			} else {
				return errors.New("invalid gateway name")
			}

			if !common.IsHexAddress(args[0]) {
				hexAddr, err = binanceAddressToHexAddress(args[0])
				if err != nil {
					return errors.Wrap(err, "invalid gateway address")
				}
			} else {
				hexAddr = args[0]
			}

			mainnetAddress, err := loom.ParseAddress(foreignChainId + ":" + hexAddr)
			if err != nil {
				return errors.Wrap(err, "invalid gateway address")
			}

			rpcClient := getDAppChainClient()
			gatewayAddr, err := rpcClient.Resolve(name)
			if err != nil {
				return errors.Wrap(err, "failed to resolve DAppChain Gateway address")
			}
			gateway := client.NewContract(rpcClient, gatewayAddr.Local)

			req := &tgtypes.TransferGatewayUpdateMainnetGatewayRequest{
				MainnetGatewayAddress: mainnetAddress.MarshalPB(),
			}

			_, err = gateway.Call("UpdateMainnetGatewayAddress", req, signer, nil)
			return err
		},
	}
	return cmd
}

func newUpdateMainnetHotWalletAddressCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update-hot-wallet-address <hot-wallet-address> <gateway-name>",
		Short:   "Update mainnet hot wallet address. Only callable by current gateway owner",
		Example: updateMainnetAddressCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			loomKeyPath := gatewayCmdFlags.PrivKeyPath
			hsmPath := gatewayCmdFlags.HSMConfigPath
			algo := gatewayCmdFlags.Algo
			signer, err := cli.GetSigner(loomKeyPath, hsmPath, algo)
			if err != nil {
				return err
			}

			var hexAddr string
			var name string
			var foreignChainId string

			if len(args) <= 1 || strings.EqualFold(args[1], GatewayName) {
				name = GatewayName
				foreignChainId = "eth"
			} else if strings.EqualFold(args[1], LoomGatewayName) {
				name = LoomGatewayName
				foreignChainId = "eth"
			} else if strings.EqualFold(args[1], BinanceGatewayName) {
				name = BinanceGatewayName
				foreignChainId = "binance"
			} else if strings.EqualFold(args[1], TronGatewayName) {
				name = TronGatewayName
				foreignChainId = "tron"
			} else {
				return errors.New("invalid gateway name")
			}

			if !common.IsHexAddress(args[0]) {
				hexAddr, err = binanceAddressToHexAddress(args[0])
				if err != nil {
					return errors.Wrap(err, "invalid gateway address")
				}
			} else {
				hexAddr = args[0]
			}

			walletAddress, err := loom.ParseAddress(foreignChainId + ":" + hexAddr)
			if err != nil {
				return errors.Wrap(err, "invalid gateway address")
			}

			rpcClient := getDAppChainClient()
			gatewayAddr, err := rpcClient.Resolve(name)
			if err != nil {
				return errors.Wrap(err, "failed to resolve DAppChain Gateway address")
			}
			gateway := client.NewContract(rpcClient, gatewayAddr.Local)

			req := &tgtypes.TransferGatewayUpdateMainnetHotWalletRequest{
				MainnetHotWalletAddress: walletAddress.MarshalPB(),
			}

			_, err = gateway.Call("UpdateMainnetHotWalletAddress", req, signer, nil)
			return err
		},
	}
	return cmd
}

func sciNot(m, n int64) *loom.BigUInt {
	ret := loom.NewBigUIntFromInt(10)
	ret.Exp(ret, loom.NewBigUIntFromInt(n), nil)
	ret.Mul(ret, loom.NewBigUIntFromInt(m))
	return ret
}
