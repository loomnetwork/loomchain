// +build evm

package main

import (
	"crypto/ecdsa"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	amtypes "github.com/loomnetwork/go-loom/builtin/types/address_mapper"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	lcrypto "github.com/loomnetwork/go-loom/crypto"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	AddressMapperName = "addressmapper"
)

func AddIdentityMappingCmd() *cobra.Command {
	var chainId string
	var callFlags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "add-identity-mapping <loom-addr> <foreign-chain-key-file>",
		Short: "Adds a mapping between a Loom Protocol account and a foreign chain account.",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var mapping address_mapper.AddIdentityMappingRequest
			user, err := cli.ParseAddress(args[0], callFlags.ChainID)
			if err != nil {
				return errors.Wrapf(err, "failed to parse address %v", args[0])
			}
			mapping.From = user.MarshalPB()

			var privkey *ecdsa.PrivateKey
			var foreignLocalAddr loom.LocalAddress
			var sigType = evmcompat.SignatureType_EIP712

			switch strings.TrimSpace(chainId) {
			case "eth":
				privkey, err = crypto.LoadECDSA(args[1])
				if err != nil {
					return errors.Wrapf(err, "read the Ethereum private key from file %v", args[1])
				}
				foreignLocalAddr, err = loom.LocalAddressFromHexString(crypto.PubkeyToAddress(privkey.PublicKey).Hex())
				if err != nil {
					return errors.Wrapf(err, "can't load the Ethereum private key from %v", args[1])
				}
			case "tron":
				privkey, err = lcrypto.LoadBtecSecp256k1PrivKey(args[1])
				if err != nil {
					return errors.Wrapf(err, "read the Tron private key from file %v", args[1])
				}
				foreignLocalAddr, err = loom.LocalAddressFromHexString(crypto.PubkeyToAddress(privkey.PublicKey).Hex())
				if err != nil {
					return errors.Wrapf(err, "can't load the Tron private key from % v", args[1])
				}
				sigType = evmcompat.SignatureType_TRON
			case "binance":
				privkey, err = crypto.LoadECDSA(args[1])
				if err != nil {
					return errors.Wrapf(err, "read the Binance private key from file %v", args[1])
				}
				signer := auth.NewBinanceSigner(crypto.FromECDSA(privkey))
				foreignLocalAddr, err = loom.LocalAddressFromHexString(evmcompat.BitcoinAddress(signer.PublicKey()).Hex())
				if err != nil {
					return errors.Wrapf(err, "can't load the Binance private key from file %v", args[1])
				}
				sigType = evmcompat.SignatureType_BINANCE
			}

			foreignAddr := loom.Address{ChainID: chainId, Local: foreignLocalAddr}
			mapping.To = foreignAddr.MarshalPB()
			mapping.Signature, err = address_mapper.SignIdentityMapping(user, foreignAddr, privkey, sigType)
			if err != nil {
				return errors.Wrapf(err, "signing mapping with %s key", chainId)
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
	cmd.Flags().StringVarP(&callFlags.URI, "uri", "u", "http://localhost:46658", "Loom Protocol base URI")
	cmd.Flags().StringVar(&callFlags.ContractAddr, "contract", "", "contract address")
	cmd.Flags().StringVarP(&callFlags.ChainID, "chain", "", "default", "chain ID")
	cmd.Flags().StringVarP(&callFlags.PrivFile, "key", "k", "", "private key file")
	cmd.Flags().StringVar(&callFlags.HsmConfigFile, "hsm", "", "HSM configuration file")
	cmd.Flags().StringVar(&callFlags.Algo, "algo", "ed25519", "Signing algorithm: ed25519, secp256k1, tron")
	cmd.Flags().StringVarP(&chainId, "mapped-chain-id", "c", "eth", "ethereum chain id")
	return cmd
}

func GetMapping() *cobra.Command {
	var flags cli.ContractCallFlags
	cmd := &cobra.Command{
		Use:   "get-mapping",
		Short: "Get mapped address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp amtypes.AddressMapperGetMappingResponse
			from, err := cli.ParseAddress(args[0], flags.ChainID)
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

			fmt.Printf("%-*s -> %-*s \n",
				50, loom.UnmarshalAddressPB(resp.From).String(),
				50, loom.UnmarshalAddressPB(resp.To).String())
			return nil
		},
	}
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
	return cmd
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
			type maxLength struct {
				From int
				To   int
			}
			ml := maxLength{From: 50, To: 50}

			fmt.Printf("%-*s | %-*s \n", ml.From, "From", ml.To, "To")
			for _, value := range resp.Mappings {
				fmt.Printf("%-*s | %-*s\n",
					ml.From, loom.UnmarshalAddressPB(value.From).String(),
					ml.To, loom.UnmarshalAddressPB(value.To).String())
			}
			return nil
		},
	}
	cli.AddContractStaticCallFlags(cmd.Flags(), &flags)
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
