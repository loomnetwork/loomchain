package main

import (
	"fmt"
	"io/ioutil"

	"github.com/eosspark/eos-go/crypto/ecc"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/loomnetwork/go-loom"
	amtypes "github.com/loomnetwork/go-loom/builtin/types/address_mapper"
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
	var callFlags cli.ContractCallFlags
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
				signature, to, err = sigEthMapping(args[1], chainId, user)
			case "tron":
				signature, to, err = sigEthMapping(args[1], chainId, user)
			case "eos":
				signature, to, err = sigEosMapping(args[1], chainId, user)
			default:
				err = errors.Errorf("unrecognised tx type %s", txType)
			}
			if err != nil {
				return err
			}

			mapping := address_mapper.AddIdentityMappingRequest{
				From:      user.MarshalPB(),
				To:        to.MarshalPB(),
				Signature: signature,
			}

			err = cli.CallContractWithFlags(&callFlags, AddressMapperName, "AddIdentityMapping", &mapping, nil)
			if err != nil {
				return errors.Wrap(err, "call contract")
			} else {
				fmt.Println("mapping successful")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&callFlags.URI, "uri", "u", "http://localhost:46658", "DAppChain base URI")
	cmd.Flags().StringVarP(&callFlags.MainnetURI, "ethereum", "e", "http://localhost:8545", "URI for talking to Ethereum")
	cmd.Flags().StringVar(&callFlags.ContractAddr, "contract", "", "contract address")
	cmd.Flags().StringVarP(&callFlags.ChainID, "chain", "", "default", "chain ID")
	cmd.Flags().StringVarP(&callFlags.PrivFile, "key", "k", "", "private key file")
	cmd.Flags().StringVar(&callFlags.HsmConfigFile, "hsm", "", "hsm config file")
	cmd.Flags().StringVar(&callFlags.Algo, "algo", "ed25519", "Signing algo: ed25519, secp256k1, tron")
	cmd.Flags().StringVarP(&chainId, "mapped-chain-id", "c", "eth", "ethereum chain id")
	cmd.Flags().StringVarP(&txType, "tx-type", "t", "eth", "ethereum chain id")
	return cmd
}

func GetMapping() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "get-mapping",
		Short: "Get mapping address",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp amtypes.AddressMapperGetMappingResponse
			from, err := cli.ParseAddress(args[0])
			if err != nil {
				return err
			}
			err = cli.StaticCallContractWithFlags(&flags, AddressMapperName, "GetMapping",
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

	AddContractCallFlags(cmd.Flags(), &flags)
	return cmd
}

func sigEthMapping(keyFile string, chainId string, user loom.Address) ([]byte, loom.Address, error) {
	ethKey, err := crypto.LoadECDSA(keyFile)
	if err != nil {
		return nil, loom.Address{}, errors.Wrapf(err, "read ethereum private key from file %v", keyFile)
	}
	ethLocalAddr, err := loom.LocalAddressFromHexString(crypto.PubkeyToAddress(ethKey.PublicKey).Hex())
	if err != nil {
		return nil, loom.Address{}, errors.Wrapf(err, "bad ethereum private key from file%v", keyFile)
	}
	ethAddr := loom.Address{ChainID: chainId, Local: ethLocalAddr}
	signature, err := address_mapper.SignIdentityMapping(user, ethAddr, ethKey)
	if err != nil {
		return nil, loom.Address{}, errors.Wrapf(err, "signing identity mapping")
	}
	return signature, ethAddr, nil
}

func sigEosMapping(keyFile string, chainId string, user loom.Address) ([]byte, loom.Address, error) {
	keyString, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, loom.Address{}, fmt.Errorf("cannot read private key %s", keyFile)
	}

	eccKey, err := ecc.NewPrivateKey(string(keyString))
	if err != nil {
		return nil, loom.Address{}, fmt.Errorf("cannot make private key %s", keyFile)
	}

	local, err := auth.LocalAddressFromEosPublicKey(eccKey.PublicKey())
	if err != nil {
		return nil, loom.Address{}, fmt.Errorf("cannot get local address from public key %s", keyFile)
	}

	addr := loom.Address{ChainID: chainId, Local: local}
	signature, err := address_mapper.SignIdentityMappingEos(user, addr, *eccKey)
	if err != nil {
		return nil, loom.Address{}, errors.Wrapf(err, "signing identity mapping")
	}
	return signature, addr, nil
}

func ListMappingCmd() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "list-mappings",
		Short: "list user account mappings",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp address_mapper.ListMappingResponse
			err := cli.StaticCallContractWithFlags(&flags, AddressMapperName, "ListMapping",
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

	AddContractCallFlags(cmd.Flags(), &flags)
	return cmd

}

func NewAddressMapperCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addressmapper <command>",
		Short: "Methods available in addressmapper contract",
	}
	cmd.AddCommand(
		AddIdentityMappingCmd(),
		GetMapping(),
		ListMappingCmd(),
	)
	return cmd
}
