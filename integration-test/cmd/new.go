package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/loomnetwork/loomchain/integration-test/lib"
	"github.com/loomnetwork/loomchain/integration-test/node"
	"github.com/spf13/cobra"
)

func newNewCommand() *cobra.Command {
	var n, k int
	var basedir, contractdir, loompath, name string
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
			loompathAbs, err := filepath.Abs(loompath)
			if err != nil {
				return err
			}
			contractdirAbs, err := filepath.Abs(contractdir)
			if err != nil {
				return err
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
				acct, err := node.CreateAccount(i, conf.BaseDir, conf.LoomPath)
				if err != nil {
					return err
				}
				accounts = append(accounts, acct)
			}

			var nodes []*node.Node
			for i := 0; i < n; i++ {
				node := node.NewNode(int64(i), conf.BaseDir, conf.LoomPath, conf.ContractDir)
				nodes = append(nodes, node)
			}

			for _, node := range nodes {
				if err := node.Init(); err != nil {
					return err
				}
			}

			if err = node.CreateCluster(nodes, accounts); err != nil {
				return err
			}

			for _, node := range nodes {
				conf.Nodes[fmt.Sprintf("%d", node.ID)] = node
				conf.NodeAddressList = append(conf.NodeAddressList, node.Address)
			}
			for _, account := range accounts {
				conf.AccountAddressList = append(conf.AccountAddressList, account.Address)
				conf.AccountPrivKeyList = append(conf.AccountPrivKeyList, account.PrivKeyPath)
			}
			conf.Accounts = accounts
			if err := lib.WriteConfig(conf, "runner.toml"); err != nil {
				return err
			}

			return nil
		},
	}

	flags := command.Flags()
	flags.IntVarP(&n, "validators", "n", 4, "The number of validators")
	flags.StringVar(&name, "name", "default", "Cluster name")
	flags.StringVar(&basedir, "base-dir", "", "Base directory")
	flags.StringVar(&contractdir, "contract-dir", "contracts", "Contract directory")
	flags.StringVar(&loompath, "loom-path", "loom", "Loom binary path")
	flags.IntVarP(&k, "account", "k", 1, "Number of account to be created")
	return command
}
