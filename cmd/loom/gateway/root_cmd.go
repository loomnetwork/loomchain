package gateway

import (
	loom "github.com/loomnetwork/go-loom"
	"github.com/spf13/cobra"
)

type gatewayFlags struct {
	ChainID       string
	URI           string
	HSMConfigPath string
	PrivKeyPath   string
	Algo          string
}

var gatewayCmdFlags gatewayFlags

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gateway <command>",
		Short: "Transfer Gateway Administration",
	}
	pflags := cmd.PersistentFlags()
	pflags.StringVarP(&gatewayCmdFlags.ChainID, "chain", "c", "default", "DAppChain ID")
	pflags.StringVarP(&gatewayCmdFlags.URI, "uri", "u", "http://localhost:46658", "DAppChain base URI")
	pflags.StringVarP(&gatewayCmdFlags.PrivKeyPath, "key", "k", "", "Validator Private Key.")
	pflags.StringVarP(&gatewayCmdFlags.HSMConfigPath, "hsm", "", "", "Validator HSM file")
	pflags.StringVarP(&gatewayCmdFlags.Algo, "algo", "", "ed25519", "Validator HSM file")
	return cmd
}

func hexToLoomAddress(hexStr string) (loom.Address, error) {
	addr, err := loom.LocalAddressFromHexString(hexStr)
	if err != nil {
		return loom.Address{}, err
	}
	return loom.Address{
		ChainID: gatewayCmdFlags.ChainID,
		Local:   addr,
	}, nil
}
