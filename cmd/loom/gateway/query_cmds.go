// +build evm

package gateway

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	loom "github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/go-loom/client"
	"github.com/loomnetwork/go-loom/client/erc20"
	"github.com/loomnetwork/go-loom/client/gateway"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/loomnetwork/loomchain/builtin/plugins/ethcoin"

	"math/big"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type accountInfo struct {
	DAppChainAddress string
	EthereumAddress  string
	LOOM             string
	ETH              string
}

const queryAccountCmdExample = `
# Get info about a DAppChain account
./loom gateway account 0x2a6b071aD396cEFdd16c731454af0d8c95ECD4B2

# Get info about an Ethereum account
./loom gateway account eth:0x5d1ddf5223a412d24901c32d14ef56cb706c0f64
`

func newQueryAccountCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "account <account-addr>",
		Short:   "Displays information about a DAppChain or Ethereum account known to the Transfer Gateway.",
		Example: queryAccountCmdExample,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var addr loom.Address
			var err error
			if strings.HasPrefix(args[0], "eth:") {
				addr, err = loom.ParseAddress(args[0])
			} else {
				if strings.HasPrefix(args[0], gatewayCmdFlags.ChainID+":") {
					addr, err = loom.ParseAddress(args[0])
				} else {
					addr, err = hexToLoomAddress(args[0])
				}
			}
			if err != nil {
				return errors.Wrap(err, "invalid account address")
			}

			rpcClient := getDAppChainClient()
			mapperAddr, err := rpcClient.Resolve("addressmapper")
			if err != nil {
				return errors.Wrap(err, "failed to resolve DAppChain Address Mapper address")
			}
			mapper := client.NewContract(rpcClient, mapperAddr.Local)
			mappedAccount, err := getMappedAccount(mapper, addr)
			if err != nil {
				fmt.Printf("No account information found for %v", addr)
			}

			var localAddr, foreignAddr loom.Address
			if addr.ChainID == "eth" {
				localAddr = mappedAccount
				foreignAddr = addr
			} else {
				localAddr = addr
				foreignAddr = mappedAccount
			}

			info := &accountInfo{
				DAppChainAddress: localAddr.Local.String(),
				EthereumAddress:  common.BytesToAddress(foreignAddr.Local).Hex(),
			}

			coinAddr, err := rpcClient.Resolve("coin")
			if err == nil {
				coinContract := client.NewContract(rpcClient, coinAddr.Local)
				req := &ctypes.BalanceOfRequest{
					Owner: localAddr.MarshalPB(),
				}
				var resp ctypes.BalanceOfResponse
				_, err = coinContract.StaticCall("BalanceOf", req, localAddr, &resp)
				if err != nil {
					return errors.Wrap(err, "failed to call coin.BalanceOf")
				}
				balance := new(big.Int)
				if resp.Balance != nil {
					balance = resp.Balance.Value.Int
				}
				info.LOOM = fmt.Sprintf(
					"%s (%s)",
					formatTokenAmount(balance), balance.String(),
				)
			}

			ethCoinAddr, err := rpcClient.Resolve("ethcoin")
			if err == nil {
				coinContract := client.NewContract(rpcClient, ethCoinAddr.Local)
				req := &ctypes.BalanceOfRequest{
					Owner: localAddr.MarshalPB(),
				}
				var resp ctypes.BalanceOfResponse
				_, err = coinContract.StaticCall("BalanceOf", req, localAddr, &resp)
				if err != nil {
					return errors.Wrap(err, "failed to call ethcoin.BalanceOf")
				}
				balance := new(big.Int)
				if resp.Balance != nil {
					balance = resp.Balance.Value.Int
				}
				info.ETH = fmt.Sprintf(
					"%s (%s)",
					formatTokenAmount(balance), balance.String(),
				)
			}

			output, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil
		},
	}
	return cmd
}

// Converts the given amount to a human readable string by stripping off 18 decimal places.
func formatTokenAmount(amount *big.Int) string {
	divisor := big.NewInt(10)
	divisor.Exp(divisor, big.NewInt(18), nil)
	return new(big.Int).Div(amount, divisor).String()
}

const queryUnclaimedTokensCmdExample = `
# Show unclaimed LOOM in the DAppChain Gateway deposited by an Ethereum account
./loom gateway unclaimed-tokens loomcoin-gateway 0x2a6b071aD396cEFdd16c731454af0d8c95ECD4B2

# Show unclaimed tokens in the DAppChain Gateway deposited by an Ethereum account
./loom gateway unclaimed-tokens eth:0x5d1ddf5223a412d24901c32d14ef56cb706c0f64
`

func newQueryUnclaimedTokensCommand() *cobra.Command {
	var gatewayName string
	cmd := &cobra.Command{
		Use:     "unclaimed-tokens <account-addr> [gateway-name]",
		Short:   "Shows unclaimed tokens in the Transfer Gateway deposited by an Ethereum account",
		Example: queryUnclaimedTokensCmdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var addr loom.Address
			var err error
			if strings.HasPrefix(args[0], "eth:") {
				addr, err = loom.ParseAddress(args[0])
			} else {
				if strings.HasPrefix(args[0], gatewayCmdFlags.ChainID+":") {
					return errors.Wrap(err, "account address is not an Ethereum address")
				} else {
					local, err := loom.LocalAddressFromHexString(args[0])
					if err != nil {
						return errors.Wrap(err, "failed to parse account address")
					}
					addr = loom.Address{ChainID: "eth", Local: local}
				}
			}
			if err != nil {
				return errors.Wrap(err, "invalid account address")
			}

			gatewayName := GatewayName
			if len(args) > 1 {
				if strings.EqualFold(args[1], LoomGatewayName) {
					gatewayName = LoomGatewayName
				} else if !strings.EqualFold(args[1], GatewayName) {
					return errors.New("Invalid gateway name")
				}
			}

			rpcClient := getDAppChainClient()
			gatewayAddr, err := rpcClient.Resolve(gatewayName)
			if err != nil {
				return errors.Wrap(err, "failed to resolve DAppChain Gateway address")
			}
			gateway := client.NewContract(rpcClient, gatewayAddr.Local)

			req := &tgtypes.TransferGatewayGetUnclaimedTokensRequest{
				Owner: addr.MarshalPB(),
			}
			resp := &tgtypes.TransferGatewayGetUnclaimedTokensResponse{}
			_, err = gateway.StaticCall("GetUnclaimedTokens", req, addr, resp)
			if err != nil {
				return errors.Wrap(err, "failed to call GetUnclaimedTokens on Gateway contract")
			}
			output, err := json.MarshalIndent(resp.UnclaimedTokens, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil
		},
	}
	cmd.Flags().StringVarP(
		&gatewayName, "gateway", "g", GatewayName,
		"Which Gateway contract to query, gateway or loomcoin-gateway",
	)
	return cmd
}

type SupplyInfo struct {
	ethCoinTotalSupply string
	LoomTotalSupply    string
	LoomTGApp          string
	LoomTGEthereum     string
	EthTGDApp          string
	EthTGEthereum      string
	EthCirculation     string
	LoomCoinsEthLoomGateway string
}

func newQueryGatewaySupplyCommand() *cobra.Command {
	var ethuri, ethgatewayaddress, loomcoinethaddress, loomgatewayethaddress string
	cmd := &cobra.Command{
		Use:   "supply",
		Short: "Displays the Supply of the Loomcoin,ethcoin",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			SupplyInfo := &SupplyInfo{}
			gatewayName := "loomcoin-gateway"
			gatewayName1 := "gateway"
			rpcClient := getDAppChainClient()

			gatewayAddr, err := rpcClient.Resolve(gatewayName)
			if err != nil {
				return errors.Wrap(err, "failed to resolve loomcoin Gateway address")
			}

			ethclient, err := ethclient.Dial(ethuri)
			gatewayClient, err := gateway.ConnectToMainnetGateway(ethclient, ethgatewayaddress)

			eth, err := gatewayClient.ETHBalance()

			SupplyInfo.EthTGEthereum = fmt.Sprintf(
				"%s (%s)",
				formatTokenAmount(eth), eth.String(),
			)

			gatewayAddr1, err := rpcClient.Resolve(gatewayName1)
			if err != nil {
				return errors.Wrap(err, "failed to resolve Gateway address")
			}

			coinAddr, err := rpcClient.Resolve("coin")

			if err != nil {
				return errors.Wrap(err, "failed to resolve coin address")
			}

			ethCoinAddr, err := rpcClient.Resolve("ethcoin")

			if err != nil {
				return errors.Wrap(err, "failed to resolve ethCoin address")
			}

			coinContract := client.NewContract(rpcClient, coinAddr.Local)

			ethcoinContract := client.NewContract(rpcClient, ethCoinAddr.Local)

			erc20client,err := erc20.ConnectToMainnetERC20(ethclient,loomcoinethaddress)

			loomgatewayethereumaddress := common.HexToAddress(loomgatewayethaddress)
			
			loomcoinsethloomgateway,err := erc20client.BalanceOf(loomgatewayethereumaddress)

			SupplyInfo.LoomCoinsEthLoomGateway = fmt.Sprintf(
				"%s (%s)",
				formatTokenAmount(loomcoinsethloomgateway), loomcoinsethloomgateway.String(),
			)

			tsreq := coin.TotalSupplyRequest{}

			var tsresp coin.TotalSupplyResponse

			_, err = coinContract.StaticCall("TotalSupply", &tsreq, coinAddr, &tsresp)

			coinsupply := tsresp.TotalSupply.Value.Int


			SupplyInfo.LoomTotalSupply = fmt.Sprintf(
				"%s (%s)",
				formatTokenAmount(coinsupply), coinsupply.String(),
			)

			tsreq1 := ethcoin.TotalSupplyRequest{}

			var tsresp1 ethcoin.TotalSupplyResponse

			_, err = coinContract.StaticCall("TotalSupply", &tsreq1, ethCoinAddr, &tsresp1)

			ethcoinsupply := tsresp1.TotalSupply.Value.Int


			SupplyInfo.ethCoinTotalSupply = fmt.Sprintf(
				"%s (%s)",
				formatTokenAmount(ethcoinsupply), ethcoinsupply.String(),
			)

			loomgatewaybalancereq := &ctypes.BalanceOfRequest{
				Owner: gatewayAddr.MarshalPB(),
			}

			var loomgatewaybalanceresp ctypes.BalanceOfResponse
			_, err = coinContract.StaticCall("BalanceOf", loomgatewaybalancereq, gatewayAddr, &loomgatewaybalanceresp)

			if err != nil {
				return errors.Wrap(err, "failed to call coin.BalanceOf")

			}

			gbalancerequest := &ctypes.BalanceOfRequest{
				Owner: gatewayAddr1.MarshalPB(),
			}

			var gbalanceresp ctypes.BalanceOfResponse
			_, err = ethcoinContract.StaticCall("BalanceOf", gbalancerequest, gatewayAddr1, &gbalanceresp)

			if err != nil {
				return errors.Wrap(err, "failed to call ethcoin.BalanceOf")
			}

			loomgatewaycoinbalance := loomgatewaybalanceresp.Balance.Value.Int

			gatewayethcoinbalance := gbalanceresp.Balance.Value.Int

			SupplyInfo.ethCoinTotalSupply = fmt.Sprintf(
				"%s (%s)",
				formatTokenAmount(loomgatewaycoinbalance), loomgatewaycoinbalance.String(),
			)

			SupplyInfo.ethCoinTotalSupply = fmt.Sprintf(
				"%s (%s)",
				formatTokenAmount(gatewayethcoinbalance), gatewayethcoinbalance.String(),
			)

			ethCirculation := ethcoinsupply.Sub(ethcoinsupply, eth)

			SupplyInfo.EthCirculation = fmt.Sprintf(
				"%s (%s)",
				formatTokenAmount(ethCirculation), ethCirculation.String(),
			)

			output, err := json.MarshalIndent(SupplyInfo, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil

		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.StringVar(&ethuri, "eth-uri", "", "Ethereum URI")
	cmdFlags.StringVarP(&ethgatewayaddress, "eth-gateway", "k", "", "Ethereum gateway Address")
	cmdFlags.StringVar(&loomcoinethaddress, "loomcoin-eth-address", "", "Ethereum URI")
	cmdFlags.StringVarP(&loomgatewayethaddress, "loomcoin-eth-gateway", "k", "", "Ethereum gateway Address")
	return cmd
}
