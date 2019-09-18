package gateway

import (
	loom "github.com/loomnetwork/go-loom"
	"github.com/spf13/cobra"
)

type gatewayFlags struct {
	ChainID        string
	URI            string
	HSMConfigPath  string
	PrivKeyPath    string
	EthPrivKeyPath string
	Algo           string
	FormatRaw      bool
	FormatJSON     bool
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
	pflags.StringVarP(&gatewayCmdFlags.PrivKeyPath, "key", "k", "", "DAppChain Private Key file path")
	pflags.StringVarP(&gatewayCmdFlags.EthPrivKeyPath, "eth-key", "", "", "Ethereum Private Key file path")
	pflags.StringVarP(&gatewayCmdFlags.HSMConfigPath, "hsm", "", "", "HSM file path")
	pflags.StringVarP(&gatewayCmdFlags.Algo, "algo", "", "ed25519", "Signing algorithm")
	pflags.BoolVar(&gatewayCmdFlags.FormatRaw, "raw", false, "Raw output format")
	pflags.BoolVar(&gatewayCmdFlags.FormatJSON, "json", false, "JSON output format")
	return cmd
}

//nolint:unused
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
