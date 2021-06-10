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
}

var gatewayCmdFlags gatewayFlags

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gateway <command>",
		Short: "Transfer gateway administration",
	}
	pflags := cmd.PersistentFlags()
	pflags.StringVarP(&gatewayCmdFlags.ChainID, "chain", "c", "default", "Loom Protocol chain ID")
	pflags.StringVarP(&gatewayCmdFlags.URI, "uri", "u", "http://localhost:46658", "Loom Protocol base URI")
	pflags.StringVarP(&gatewayCmdFlags.PrivKeyPath, "key", "k", "", "Path to the Loom Protocol private key")
	pflags.StringVarP(&gatewayCmdFlags.EthPrivKeyPath, "eth-key", "", "", "Path to the Ethereum private key")
	pflags.StringVarP(&gatewayCmdFlags.HSMConfigPath, "hsm", "", "", "Path to the HSM configuration file")
	pflags.StringVarP(&gatewayCmdFlags.Algo, "algo", "", "ed25519", "Signing algorithm")
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
