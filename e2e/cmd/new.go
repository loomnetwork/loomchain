package main

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/loomnetwork/loomchain/e2e/common"
)

func newNewCommand() *cobra.Command {
	var validators, altValidators uint64
	var k, numEthAccounts int
	var baseDir, contractDir, loomPath, altLoomPath, name string
	var logLevel, logDest string
	var genesisFile, configFile string
	var logAppDb bool
	var force bool
	var useFnConsensus, checkAppHash bool
	command := &cobra.Command{
		Use:           "new",
		Short:         "Create n nodes to run loom",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(ccmd *cobra.Command, args []string) error {
			var err error
			contractdirAbs := ""
			if contractDir != "" {
				contractdirAbs, err = filepath.Abs(contractDir)
				if err != nil {
					return err
				}
			}
			_, err = common.GenerateConfig(
				name, "", genesisFile, configFile, baseDir, contractdirAbs, loomPath, altLoomPath,
				validators, altValidators,
				k, numEthAccounts,
				useFnConsensus, force, checkAppHash,
			)
			return err
		},
	}

	flags := command.Flags()
	flags.Uint64VarP(&validators, "validators", "n", 4, "The number of validators")
	flags.Uint64VarP(&altValidators, "alt-validators", "m", 0, "The number of validators on alternate build")
	flags.StringVar(&name, "name", "default", "Cluster name")
	flags.StringVar(&baseDir, "base-dir", "", "Base directory")
	flags.StringVar(&contractDir, "contract-dir", "contracts", "Contract directory")
	flags.StringVar(&loomPath, "loom-path", "loom", "Loom binary path")
	flags.StringVar(&altLoomPath, "alt-loom-path", "loom", "Alternate loom binary path")
	flags.IntVarP(&k, "account", "k", 1, "Number of accounts to be created")
	flags.IntVarP(&numEthAccounts, "num-eth-accounts", "e", 1, "Number of ethereum accounts to be created")
	flags.BoolVarP(&logAppDb, "log-app-db", "a", false, "Log the app state database usage")
	flags.BoolVarP(&useFnConsensus, "fnconsensus", "", false, "Enable fnconsensus via the reactor")
	flags.BoolVarP(&force, "force", "f", false, "Force to create new cluster")
	flags.StringVar(&logLevel, "log-level", "debug", "Log level")
	flags.StringVar(&logDest, "log-destination", "file://loom.log", "Log Destination")
	flags.StringVarP(&genesisFile, "genesis-template", "g", "", "Path to genesis.json")
	flags.StringVarP(&configFile, "config-template", "c", "", "Path to loom.yml")
	flags.BoolVarP(&checkAppHash, "check-apphash", "p", false, "Check apphash on exit from test")
	return command
}
