package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	AddressMapperName = "addressmapper"
)

func AddIdentityMappingCmd() *cobra.Command {
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

			err = cli.CallContract(AddressMapperName, "AddIdentityMapping", &mapping, nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			} else {
				fmt.Println("mapping successfull")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&chainId, "mapped-chain-id", "c", "eth", "ethereum chain id")
	return cmd
}

func ListMappingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-mappings",
		Short: "list user account mappings",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp address_mapper.ListMappingResponse

			err := cli.StaticCallContract(AddressMapperName, "ListMapping", &address_mapper.ListMappingRequest{}, &resp)
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

// todo  RemoveMapping, HasMapping and GetMapping

func AddAddressMappingMethods(addressMappingCmd *cobra.Command) {
	addressMappingCmd.AddCommand(
		AddIdentityMappingCmd(),
		ListMappingCmd(),
	)
}
