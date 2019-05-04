package common

import (
	"github.com/loomnetwork/go-loom/cli"
	flag "github.com/spf13/pflag"
)

//Add Contract call Flags
func AddContractCallFlags(flagSet *flag.FlagSet, callFlags *cli.ContractCallFlags) {
	flagSet.StringVarP(&callFlags.URI, "uri", "u", "http://localhost:46658", "DAppChain base URI")
	flagSet.StringVarP(&callFlags.MainnetURI, "ethereum", "e", "http://localhost:8545", "URI for talking to Ethereum")
	flagSet.StringVar(&callFlags.ContractAddr, "contract", "", "contract address")
	flagSet.StringVarP(&callFlags.ChainID, "chain", "c", "default", "chain ID")
	flagSet.StringVarP(&callFlags.PrivFile, "key", "k", "", "private key file")
	flagSet.StringVar(&callFlags.HsmConfigFile, "hsm", "", "hsm config file")
	flagSet.StringVar(&callFlags.Algo, "algo", "ed25519", "Signing algo: ed25519, secp256k1, tron")
	flagSet.StringVar(&callFlags.CallerChainID, "caller-chain", "", "Overrides chain ID of caller")
}

//Add Contract Static Call Flags
func AddContractStaticCallFlags(flagSet *flag.FlagSet, callFlags *cli.ContractCallFlags) {
	flagSet.StringVarP(&callFlags.URI, "uri", "u", "http://localhost:46658", "DAppChain base URI")
	flagSet.StringVarP(&callFlags.MainnetURI, "ethereum", "e", "http://localhost:8545", "URI for talking to Ethereum")
	flagSet.StringVar(&callFlags.ContractAddr, "contract", "", "contract address")
	flagSet.StringVarP(&callFlags.ChainID, "chain", "c", "default", "chain ID")
	flagSet.StringVarP(&callFlags.PrivFile, "key", "k", "", "private key file")
	flagSet.StringVar(&callFlags.HsmConfigFile, "hsm", "", "hsm config file")
	flagSet.StringVar(&callFlags.Algo, "algo", "ed25519", "Signing algo: ed25519, secp256k1, tron")
	flagSet.StringVar(&callFlags.CallerChainID, "caller-chain", "", "Overrides chain ID of caller")
}
