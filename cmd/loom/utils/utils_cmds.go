// !+build evm

package utils

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/loomnetwork/go-loom"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type utilsFlags struct {
	HexAddres string `json:"hex"`
	ChainID   string `json:chainid`
}

var utilsFlagsCmd utilsFlags

func NewUtilsCommand() *cobra.Command {
	cmd := newRootCommand()
	cmd.AddCommand(
		newConvertHexToB64Command(),
	)
	return cmd
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "utils <command>",
		Short: "Utility function",
	}
	pflags := cmd.PersistentFlags()
	pflags.StringVarP(&utilsFlagsCmd.ChainID, "chain", "c", "default", "DAppChain ID")
	return cmd
}

func newConvertHexToB64Command() *cobra.Command {
	converter := &cobra.Command{
		Use:   "hextob64",
		Short: "convert hex to b64",
		RunE: func(cmd *cobra.Command, args []string) error {
			var addr loom.Address
			var err error

			if strings.HasPrefix(args[0], "eth:") {
				addr, err = loom.ParseAddress(args[0])
			} else {
				if strings.HasPrefix(args[0], utilsFlagsCmd.ChainID+":") {
					addr, err = loom.ParseAddress(args[0])
				} else {
					addr, err = hexToLoomAddress(args[0])
				}
			}
			if err != nil {
				return errors.Wrap(err, "invalid account address")
			}
			encoder := base64.StdEncoding

			fmt.Printf("local address base64: %s\n", encoder.EncodeToString([]byte(addr.Local)))
			return nil
		},
	}
	converter.Flags().StringVarP(&utilsFlagsCmd.ChainID, "chain", "c", "default", "DAppChain ID")
	return converter
}

//nolint:unused
func hexToLoomAddress(hexStr string) (loom.Address, error) {
	addr, err := loom.LocalAddressFromHexString(hexStr)
	if err != nil {
		return loom.Address{}, err
	}
	return loom.Address{
		ChainID: utilsFlagsCmd.ChainID,
		Local:   addr,
	}, nil
}
