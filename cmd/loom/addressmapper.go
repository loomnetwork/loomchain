package main

import (
	"fmt"
	"io/ioutil"

	"github.com/eosspark/eos-go/crypto/ecc"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	AddressMapperName = "addressmapper"
)

func AddIdentityMappingCmd() *cobra.Command {
	var chainId string
	var txType string
	cmd := &cobra.Command{
		Use:   "add-identity-mapping <loom-addr> <eth-key-file>",
		Short: "Adds a mapping between a DAppChain account and a Mainnet account.",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			user, err := cli.ParseAddress(args[0])
			if err != nil {
				return errors.Wrapf(err, "resolve address arg %v", args[0])
			}
			var to loom.Address
			var signature []byte
			switch txType {
			case "eth":
				signature, to, err = sigEthMapping(args, chainId, user)
			case "tron":
				signature, to, err = sigEthMapping(args, chainId, user)
			case "eos":
				signature, to, err = sigEosMapping(args, chainId, user)
			default:
				err = errors.Errorf("unrecognised tx type %s", txType)
			}
			if err != nil {
				return err
			}

			mapping := address_mapper.AddIdentityMappingRequest{
				From:           user.MarshalPB(),
				To:             to.MarshalPB(),
				Signature:      signature,
			}

			err = cli.CallContract(AddressMapperName, "AddIdentityMapping", &mapping, nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			} else {
				fmt.Println("mapping successful")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&chainId, "mapped-chain-id", "c", "eth", "ethereum chain id")
	cmd.Flags().StringVarP(&txType, "tx-type", "t", "eth", "ethereum chain id")
	return cmd
}

func sigEthMapping(args []string, chainId string, user loom.Address)  ([]byte, loom.Address, error) {
	ethKey, err := crypto.LoadECDSA(args[1])
	if err != nil {
		return nil, loom.Address{}, errors.Wrapf(err, "read ethereum private key from file %v", args[1])
	}
	ethLocalAddr, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(ethKey.PublicKey).Hex())
	if err != nil {
		return nil, loom.Address{}, errors.Wrapf(err, "bad ethereum private key from file%v", args[1])
	}
	ethAddr := loom.Address{ChainID: chainId, Local: ethLocalAddr}
	signature, err :=  address_mapper.SignIdentityMapping(user, ethAddr, ethKey)
	if err != nil {
		return nil, loom.Address{}, errors.Wrapf(err, "singing identity mapping")
	}
	return signature, ethAddr, nil
}

func sigEosMapping(args []string, chainId string, user loom.Address)  ([]byte, loom.Address, error) {
	keyString, err := ioutil.ReadFile(args[1])
	if err != nil {
		return nil, loom.Address{}, fmt.Errorf("cannot read private key %s", args[0])
	}

	eccKey, err := ecc.NewPrivateKey(string(keyString))
	if err != nil {
		return nil, loom.Address{}, fmt.Errorf("cannot make private key %s", args[0])
	}

	local, err := auth.LocalAddressFromEosPublicKey(eccKey.PublicKey())
	if err != nil {
		return nil, loom.Address{}, fmt.Errorf("cannot get local address from public key %s", args[0])
	}

	addr := loom.Address{ChainID: chainId, Local: local}
	signature, err := address_mapper.SignIdentityMappingEos(user, addr, *eccKey)
	if err != nil {
		return nil, loom.Address{}, errors.Wrapf(err, "singing identity mapping")
	}
	return signature, addr, nil
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


