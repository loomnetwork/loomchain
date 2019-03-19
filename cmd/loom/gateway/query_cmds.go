// +build evm
package gateway

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	loom "github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/go-loom/client"
	"github.com/loomnetwork/go-loom/client/erc20"
	"github.com/loomnetwork/go-loom/client/gateway"
	cmn "github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/loomnetwork/loomchain/builtin/plugins/ethcoin"

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

type Supply struct {
	Ethereum Eth  `json:"ethereum"`
	LoomCoin Loom `json:"loomcoin"`
}

type Eth struct {
	Dappchain_total_supply       string `json:"dappchain_total_supply"`
	Dappchain_circulating_supply string `json:"dappchain_circulating_supply"`
	Dappchain_gateway_total      string `json:"dappchain_gateway_total"`
	Ethereum_gateway_total       string `json:"ethereum_total_supply"`
	Dappchain_gateway_unclaimed  string `json:"dappchain_gateway_unclaimed"`
}

type Loom struct {
	Dappchain_total_supply      string `json:"dappchain_total_supply"`
	Dappchain_gateway_total     string `json:"dappchain_gateway_total"`
	Ethereum_gateway_total      string `json:"ethereum_gateway_total"`
	Dappchain_gateway_unclaimed string `json:"dappchain_gateway_unclaimed"`
}

func newQueryGatewaySupplyCommand() *cobra.Command {
	var ethURI, gatewayAddressEth, loomCoinAddressEth, loomGatewayAddressEth string
	var raw bool
	cmd := &cobra.Command{
		Use:   "supply",
		Short: "Displays holdings of DAppChain & Ethereum Gateways",
		Args:  cobra.MinimumNArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			Supply := &Supply{}
			Eth := &Eth{}
			Loom := &Loom{}
			gatewayName := "loomcoin-gateway"
			gatewayName1 := "gateway"
			rpcClient := getDAppChainClient()

			gatewayAddr, err := rpcClient.Resolve(gatewayName)
			if err != nil {
				return errors.Wrap(err, "failed to resolve loomcoin Gateway address")
			}
			ethClient, err := ethclient.Dial(ethURI)
			if err != nil {
				return err
			}
			gatewayClient, err := gateway.ConnectToMainnetGateway(ethClient, gatewayAddressEth)
			if err != nil {
				return err
			}
			eth, err := gatewayClient.ETHBalance()
			if err != nil {
				return err
			}
			if raw == false {
				Eth.Ethereum_gateway_total = fmt.Sprintf(
					"%s (%s)",
					formatTokenAmount(eth), eth.String(),
				)

			} else {
				Eth.Ethereum_gateway_total = fmt.Sprintf(
					"%s", formatTokenAmount(eth))

			}
			gatewayAddr1, err := rpcClient.Resolve(gatewayName1)
			if err != nil {
				return errors.Wrap(err, "failed to resolve Gateway address")
			}
			gatewayContract := client.NewContract(rpcClient, gatewayAddr.Local)
			ethLocalAddr, err := loom.LocalAddressFromHexString(loomCoinAddressEth)
			if err != nil {
				return err
			}
			ethereumlocalAddr := loom.Address{ChainID: "eth", Local: ethLocalAddr}
			req := &tgtypes.TransferGatewayGetUnclaimedContractTokensRequest{TokenAddress: ethereumlocalAddr.MarshalPB()}
			resp := &tgtypes.TransferGatewayGetUnclaimedContractTokensResponse{}
			_, err = gatewayContract.StaticCall("GetUnclaimedContractTokens", req, gatewayAddr, resp)
			if err != nil {
				return err
			}
			unclaimedLOOM := cmn.BigUInt{big.NewInt(0)}
			loomTokenValue := int64(tgtypes.TransferGatewayTokenKind_value["LOOMCOIN"])
			for _, token := range resp.UnclaimedTokens {
				for _, tokenamount := range token.Amounts {
					if tokenamount.TokenID.Value == *loom.NewBigUIntFromInt(loomTokenValue) {
						unclaimedLOOM = *unclaimedLOOM.Add(&unclaimedLOOM, &tokenamount.TokenAmount.Value)
					}
				}
			}
			Loom.Dappchain_gateway_unclaimed = unclaimedLOOM.String()
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
			erc20client, err := erc20.ConnectToMainnetERC20(ethClient, loomCoinAddressEth)
			if err != nil {
				return err
			}
			loomGatewayEthereumAddress := common.HexToAddress(loomGatewayAddressEth)
			loomCoinsEthLoomGateway, err := erc20client.BalanceOf(loomGatewayEthereumAddress)
			if err != nil {
				return err
			}
			if raw == false {
				Loom.Ethereum_gateway_total = fmt.Sprintf(
					"%s (%s)",
					formatTokenAmount(loomCoinsEthLoomGateway), loomCoinsEthLoomGateway.String(),
				)
			} else {
				Loom.Ethereum_gateway_total = fmt.Sprintf(
					"%s",
					formatTokenAmount(loomCoinsEthLoomGateway))
			}
			tsreq := coin.TotalSupplyRequest{}
			var tsresp coin.TotalSupplyResponse
			_, err = coinContract.StaticCall("TotalSupply", &tsreq, coinAddr, &tsresp)
			if err != nil {
				return err
			}
			coinSupply := tsresp.TotalSupply.Value.Int
			if raw == false {
				Loom.Dappchain_total_supply = fmt.Sprintf(
					"%s (%s)", formatTokenAmount(coinSupply), coinSupply.String(),
				)
			} else {
				Loom.Dappchain_total_supply = fmt.Sprintf(
					"%s", formatTokenAmount(coinSupply))
			}
			tsreq1 := ethcoin.TotalSupplyRequest{}
			var tsresp1 ethcoin.TotalSupplyResponse
			_, err = coinContract.StaticCall("TotalSupply", &tsreq1, ethCoinAddr, &tsresp1)
			if err != nil {
				return err
			}
			ethCoinSupply := tsresp1.TotalSupply.Value.Int
			if raw == false {
				Eth.Dappchain_total_supply = fmt.Sprintf(
					"%s (%s)", formatTokenAmount(ethCoinSupply), ethCoinSupply.String(),
				)
			} else {
				Eth.Dappchain_total_supply = fmt.Sprintf(
					"%s", formatTokenAmount(ethCoinSupply))
			}
			loomGatewayBalanceReq := &ctypes.BalanceOfRequest{
				Owner: gatewayAddr.MarshalPB(),
			}
			var loomGatewayBalanceResp ctypes.BalanceOfResponse
			_, err = coinContract.StaticCall("BalanceOf", loomGatewayBalanceReq, gatewayAddr, &loomGatewayBalanceResp)
			if err != nil {
				return errors.Wrap(err, "failed to call coin.BalanceOf")
			}
			gBalanceRequest := &ctypes.BalanceOfRequest{
				Owner: gatewayAddr1.MarshalPB(),
			}
			var gBalanceResp ctypes.BalanceOfResponse
			_, err = ethcoinContract.StaticCall("BalanceOf", gBalanceRequest, gatewayAddr1, &gBalanceResp)
			if err != nil {
				return errors.Wrap(err, "failed to call ethcoin.BalanceOf")
			}
			loomGatewayCoinBalance := loomGatewayBalanceResp.Balance.Value.Int
			gatewayEthCoinBalance := gBalanceResp.Balance.Value.Int
			if raw == false {
				Loom.Dappchain_gateway_total = fmt.Sprintf(
					"%s (%s)",
					formatTokenAmount(loomGatewayCoinBalance), loomGatewayCoinBalance.String())
			} else {
				Loom.Dappchain_gateway_total = fmt.Sprintf(
					"%s", formatTokenAmount(loomGatewayCoinBalance))
			}
			if raw == false {
				Eth.Dappchain_gateway_total = fmt.Sprintf(
					"%s (%s)",
					formatTokenAmount(gatewayEthCoinBalance), gatewayEthCoinBalance.String(),
				)
			} else {
				Eth.Dappchain_gateway_total = fmt.Sprintf(
					"%s", formatTokenAmount(gatewayEthCoinBalance))
			}
			ethCirculation := ethCoinSupply.Sub(ethCoinSupply, eth)
			if raw == false {
				Eth.Dappchain_circulating_supply = fmt.Sprintf(
					"%s (%s)",
					formatTokenAmount(ethCirculation), ethCirculation.String(),
				)
			} else {
				Eth.Dappchain_circulating_supply = fmt.Sprintf(
					"%s", formatTokenAmount(ethCirculation))

			}
			Supply.LoomCoin = *Loom
			Supply.Ethereum = *Eth
			output, err := json.MarshalIndent(Supply, "", "")
			fmt.Println(string(output))

			return nil

		},
	}
	cmdFlags := cmd.Flags()
	cmdFlags.BoolVar(&raw, "raw", false, "raw format output")
	cmdFlags.StringVar(&ethURI, "eth-uri", "https://mainnet.infura.io/v3/a5a5151fecba45229aa77f0725c10241", "Ethereum URI")
	cmdFlags.StringVar(&gatewayAddressEth, "eth-gateway", "0x223CA78df868367D214b444d561B9123c018963A", "Ethereum Gateway Address")
	cmdFlags.StringVar(&loomCoinAddressEth, "loomcoin-eth-address", "0xa4e8c3ec456107ea67d3075bf9e3df3a75823db0", "LOOM Ethereum Contract Address")
	cmdFlags.StringVar(&loomGatewayAddressEth, "loomcoin-eth-gateway", "0x8f8E8b3C4De76A31971Fe6a87297D8f703bE8570", "LOOM Ethereum Gateway Address")
	return cmd
}
