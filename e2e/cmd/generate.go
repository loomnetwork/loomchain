package main

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/loomnetwork/loomchain/e2e/lib"
	"github.com/loomnetwork/loomchain/e2e/node"
	"github.com/spf13/cobra"
)

func newGenerateCommand() *cobra.Command {
	var k int
	var basedir, keydir, loompath, name, genesis string
	var urls string
	var force, logAppDb bool
	command := &cobra.Command{
		Use:           "generate ",
		Short:         "Generate genesis and account for testing",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(ccmd *cobra.Command, args []string) error {
			basedirAbs, err := filepath.Abs(basedir)
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
			keydirAbs := path.Join(basedirAbs, keydir)
			if err := os.MkdirAll(keydirAbs, os.ModePerm); err != nil {
				return err
			}
			if err := os.MkdirAll(keydirAbs, os.ModePerm); err != nil {
				return err
			}

			conf := lib.Config{
				Name:     name,
				LoomPath: loompathAbs,
				Nodes:    make(map[string]*node.Node),
				BaseDir:  basedirAbs,
				LogAppDb: logAppDb,
			}

			var accounts []*node.Account
			for i := 0; i < k; i++ {
				acct, err := node.CreateAccount(i, keydirAbs, loompathAbs)
				if err != nil {
					return err
				}
				accounts = append(accounts, acct)
			}

			ps := strings.Split(urls, ",")
			var nodes []*node.Node
			for i, p := range ps {
				u, err := url.Parse(p)
				if err != nil {
					return err
				}
				if u.Scheme != "http" {
					return fmt.Errorf("invalid scheme '%s'", u.Scheme)
				}
				node := &node.Node{
					ID:         int64(i),
					RPCAddress: u.String(),
				}
				nodes = append(nodes, node)
			}

			for _, node := range nodes {
				conf.Nodes[fmt.Sprintf("%d", node.ID)] = node
				conf.NodeAddressList = append(conf.NodeAddressList, node.Address)
				conf.NodeBase64AddressList = append(conf.NodeBase64AddressList, node.Local)
				conf.NodePubKeyList = append(conf.NodePubKeyList, node.PubKey)
				conf.NodePrivKeyPathList = append(conf.NodePrivKeyPathList, node.PrivKeyPath)
				conf.NodeRPCAddressList = append(conf.NodeRPCAddressList, node.RPCAddress)
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

			err = node.GenesisFromTemplate(genesis, path.Join(basedir, "genesis.json"), accounts...)
			return err
		},
	}

	flags := command.Flags()
	flags.StringVar(&basedir, "base-dir", "coin", "Base directory")
	flags.StringVar(&name, "name", "coin", "Contract name")
	flags.IntVarP(&k, "account", "k", 1, "Number of account to be created")
	flags.StringVar(&keydir, "key-dir", "keys", "Key directory")
	flags.StringVar(&loompath, "loom-path", "loom", "Loom binary path")
	flags.BoolVarP(&logAppDb, "log-app-db", "a", false, "Log the app state database usage")
	flags.BoolVarP(&force, "force", "f", false, "Force to create new genesis")
	flags.StringVar(&urls, "urls", "", "URL for read and write")
	flags.StringVar(&genesis, "genesis", "genesis.json", "Existing genesis file")
	return command
}
