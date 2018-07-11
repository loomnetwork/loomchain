package main

import (
	"encoding/base64"
	"fmt"
	"strings"

	loom "github.com/loomnetwork/go-loom"
	"github.com/spf13/cobra"
)

func newPubKeyCommand() *cobra.Command {
	command := &cobra.Command{
		Use:           "pubkey",
		Short:         "Convert public key to loom's address hex format",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(ccmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("at least one argument required")
			}
			encoder := base64.StdEncoding
			for _, pubkey := range args {
				pk := strings.TrimSpace(pubkey)
				data, err := encoder.DecodeString(pk)
				if err != nil {
					return err
				}
				address := loom.LocalAddressFromPublicKey(data)
				fmt.Printf("loom address: %s\n", address)
			}
			return nil
		},
	}
	return command
}
