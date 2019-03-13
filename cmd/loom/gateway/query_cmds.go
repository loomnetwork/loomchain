// +build evm

package gateway

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/go-loom/client"
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
