package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/loomnetwork/loomchain/e2e/lib"
	"github.com/loomnetwork/loomchain/e2e/node"
	"github.com/spf13/cobra"
)

func newNewCommand() *cobra.Command {
	var validators, altValidators, k int
	var basedir, contractdir, loompath, loompath2, name string
	var logLevel, logDest string
	var genesisFile, configFile string
	var logAppDb bool
	var force bool
	var useFnConsensus bool
	command := &cobra.Command{
		Use:           "new",
		Short:         "Create n nodes to run loom",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(ccmd *cobra.Command, args []string) error {
			basedirAbs, err := filepath.Abs(path.Join(basedir, name))
			if err != nil {
				return err
			}

			_, err = os.Stat(basedirAbs)
			if !force && err == nil {
				return fmt.Errorf("directory %s exists; please use the flag --force to create new nodes", basedirAbs)
			}

			if force {
				err = os.RemoveAll(basedirAbs)
				if err != nil {
					return err
				}
			}

			loompathAbs, err := filepath.Abs(loompath)

			if err != nil {
				return err
			}
			var contractdirAbs string
			if contractdir != "" {
				contractdirAbs, err = filepath.Abs(contractdir)
				if err != nil {
					return err
				}
			}

			conf := lib.Config{
				Name:        name,
				BaseDir:     basedirAbs,
				LoomPath:    loompathAbs,
				ContractDir: contractdirAbs,
				Nodes:       make(map[string]*node.Node),
			}

			if err := os.MkdirAll(conf.BaseDir, os.ModePerm); err != nil {
				return err
			}

			var accounts []*node.Account
			for i := 0; i < k; i++ {
				acct, err := node.CreateAccount(i, conf.BaseDir, loompathAbs)
				if err != nil {
					return err
				}
				accounts = append(accounts, acct)
			}

			var nodes []*node.Node
			for i := 0; i < validators; i++ {
				node := node.NewNode(int64(i), conf.BaseDir, loompathAbs, conf.ContractDir, genesisFile, configFile)
				node.LogLevel = logLevel
				node.LogDestination = logDest
				node.LogAppDb = logAppDb
				nodes = append(nodes, node)
			}

			loompathAbs2, err := filepath.Abs(loompath2)
			for i := validators ; i < validators + altValidators; i++ {
				newNode := node.NewNode(int64(i), conf.BaseDir, loompathAbs2, conf.ContractDir, genesisFile, configFile)
				newNode.LogLevel = logLevel
				newNode.LogDestination = logDest
				newNode.LogAppDb = logAppDb
				nodes = append(nodes, newNode)
			}

			for _, node := range nodes {
				if err := node.Init(accounts); err != nil {
					return err
				}
			}

			if err = node.CreateCluster(nodes, accounts, useFnConsensus); err != nil {
				return err
			}

			for _, node := range nodes {
				conf.Nodes[fmt.Sprintf("%d", node.ID)] = node
				conf.NodeAddressList = append(conf.NodeAddressList, node.Address)
				conf.NodeBase64AddressList = append(conf.NodeBase64AddressList, node.Local)
				conf.NodePubKeyList = append(conf.NodePubKeyList, node.PubKey)
				conf.NodePrivKeyPathList = append(conf.NodePrivKeyPathList, node.PrivKeyPath)
				conf.NodeProxyAppAddressList = append(conf.NodeProxyAppAddressList, node.ProxyAppAddress)
			}
			for _, account := range accounts {
				conf.AccountAddressList = append(conf.AccountAddressList, account.Address)
				conf.AccountPrivKeyPathList = append(conf.AccountPrivKeyPathList, account.PrivKeyPath)
				conf.AccountPubKeyList = append(conf.AccountPubKeyList, account.PubKey)
			}
			conf.Accounts = accounts
			if err := lib.WriteConfig(conf, "runner.toml"); err != nil {
				return err
			}

			return nil
		},
	}

	flags := command.Flags()
	flags.IntVarP(&validators, "validators", "n", 4, "The number of validators")
	flags.IntVarP(&altValidators, "alt-validators", "m", 0, "The number of validators on alternate build")
	flags.StringVar(&name, "name", "default", "Cluster name")
	flags.StringVar(&basedir, "base-dir", "", "Base directory")
	flags.StringVar(&contractdir, "contract-dir", "contracts", "Contract directory")
	flags.StringVar(&loompath, "loom-path", "loom", "Loom binary path")
	flags.StringVar(&loompath2, "loom-path2", "loom", "Alternate loom binary path")
	flags.IntVarP(&k, "account", "k", 1, "Number of account to be created")
	flags.BoolVarP(&logAppDb, "log-app-db", "a", false, "Log the app state database usage")
	flags.BoolVarP(&useFnConsensus, "fnconsensus", "", false, "Enable fnconsensus via the reactor")
	flags.BoolVarP(&force, "force", "f", false, "Force to create new cluster")
	flags.StringVar(&logLevel, "log-level", "debug", "Log level")
	flags.StringVar(&logDest, "log-destination", "file://loom.log", "Log Destination")
	flags.StringVarP(&genesisFile, "genesis-template", "g", "", "Path to genesis.json")
	flags.StringVarP(&configFile, "config-template", "c", "", "Path to loom.yml")
	return command
}
