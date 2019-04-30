package main

import (
	"fmt"
	flag "github.com/spf13/pflag"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/loomnetwork/go-loom"
	amtypes "github.com/loomnetwork/go-loom/builtin/types/address_mapper"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	AddressMapperName = "addressmapper"
)


func addContractCallFlags(flagSet *flag.FlagSet, callFlags *cli.ContractCallFlags) {
	flagSet.StringVarP(&callFlags.URI, "uri", "u", "http://localhost:46658", "DAppChain base URI")
	flagSet.StringVarP(&callFlags.MainnetURI, "ethereum", "e", "http://localhost:8545", "URI for talking to Ethereum")
	flagSet.StringVar(&callFlags.ContractAddr, "contract", "", "contract address")
	flagSet.StringVarP(&callFlags.ChainID, "chain", "", "default", "chain ID")
	flagSet.StringVarP(&callFlags.PrivFile, "key", "k", "", "private key file")
	flagSet.StringVar(&callFlags.HsmConfigFile, "hsm", "", "hsm config file")
	flagSet.StringVar(&callFlags.Algo, "algo", "ed25519", "Signing algo: ed25519, secp256k1, tron")
	flagSet.StringVar(&callFlags.CallerChainID, "caller-chain", "", "Overrides chain ID of caller")
}

func AddIdentityMappingCmd(flags *cli.ContractCallFlags) *cobra.Command {
	var chainId string
	cmd := &cobra.Command{
		Use:   "add-identity-mapping <loom-addr> <eth-key-file>",
		Short: "Adds a mapping between a DAppChain account and a Mainnet account.",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var mapping address_mapper.AddIdentityMappingRequest
			user, err := cli.ParseAddress(args[0])
			if err != nil {
				return errors.Wrapf(err, "resolve address arg %v", args[0])
			}
			mapping.From = user.MarshalPB()

			ethKey, err := crypto.LoadECDSA(args[1])
			if err != nil {
				return errors.Wrapf(err, "read ethereum private key from file%v", args[1])
			}
			ethLocalAddr, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(ethKey.PublicKey).Hex())
			if err != nil {
				return errors.Wrapf(err, "bad ethereum private key from file%v", args[1])
			}
			ethAddr := loom.Address{ChainID: chainId, Local: ethLocalAddr}
			mapping.To = ethAddr.MarshalPB()
			mapping.Signature, err = address_mapper.SignIdentityMapping(user, ethAddr, ethKey)
			if err != nil {
				return errors.Wrap(err, "sigining mapping with ethereum key")
			}

			err = cli.CallContractWithFlags(flags,AddressMapperName, "AddIdentityMapping", &mapping, nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			} else {
				fmt.Println("mapping successful")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&chainId, "mapped-chain-id", "c", "eth", "ethereum chain id")
	return cmd
}

func GetMapping(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get-mapping",
		Short: "Get mapping address",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp amtypes.AddressMapperGetMappingResponse
			from, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}
			err = cli.StaticCallContractWithFlags(flags,AddressMapperName, "GetMapping",
				&amtypes.AddressMapperGetMappingRequest{
				From: from.MarshalPB(),
			}, &resp)
			if err != nil {
				return err
			}
			out, err := formatJSON(&resp)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		},
	}
}

func ListMappingCmd(flags *cli.ContractCallFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list-mappings",
		Short: "list user account mappings",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp address_mapper.ListMappingResponse

			err := cli.StaticCallContractWithFlags(flags,AddressMapperName, "ListMapping",
				&address_mapper.ListMappingRequest{}, &resp)
			if err != nil {
				return errors.Wrap(err, "static call contract")
			}
			out, err := formatJSON(&resp)
			if err != nil {
				return errors.Wrap(err, "format JSON response")
			}
			fmt.Println(out)
			return nil
		},
	}
}


func NewAddressMapperCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addressmapper <command>",
		Short: "Methods available in addressmapper contract",
	}
	var flags cli.ContractCallFlags
	addContractCallFlags(cmd.PersistentFlags(), &flags)

	cmd.AddCommand(
		AddIdentityMappingCmd(&flags),
		GetMapping(&flags),
		ListMappingCmd(&flags),
	)
	return cmd
}
