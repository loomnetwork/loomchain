package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/loomnetwork/loomchain/e2e/lib"
	"github.com/loomnetwork/loomchain/e2e/node"
	"github.com/loomnetwork/loomchain/privval"
	"github.com/spf13/cobra"
)

const usageExample = `
./validators-tool import \
  --nodes 0,1,2,3 \
  --name cluster \
  --base-dir /mnt/c/Projects/work/loom/us1-backup \
  --priv-val-height 1020269
`

// The import command assigns new ports for each node in the cluster, optionally resets the last_*
// fields in priv_validator.json files, and writes out runner.toml so that the validators-tool can
// spin up the cluster. Intended use of this command is to create local clusters from backups of
// remote clusters for testing purposes.
//
// NOTE: loom.yaml and chaindata/config/config.toml will be modified in place for each node, and the
// address book deleted from chaindata/config. It's expected that the cluster directory matches the
// layout below:
// <base-dir>
//     <cluster> # cluster name as specified via name param
//         <0>   # first node base dir
//         <1>   # second node base dir
//         <2>   # etc.
//         <3>
func newImportCommand() *cobra.Command {
	var nodeIDs []int
	var clusterName, baseDir, loomPath string
	var numAccts, privValHeight int

	cmd := &cobra.Command{
		Use:     "import",
		Short:   "Create a validators-tool config from an existing cluster.",
		Example: usageExample,
		RunE: func(ccmd *cobra.Command, args []string) error {
			absBaseDir, err := filepath.Abs(path.Join(baseDir, clusterName))
			if err != nil {
				return err
			}

			if _, err = os.Stat(absBaseDir); os.IsNotExist(err) {
				return fmt.Errorf("directory %s doesn't exist", absBaseDir)
			}

			absLoomPath, err := filepath.Abs(loomPath)
			if err != nil {
				return err
			}

			conf := lib.Config{
				Name:     clusterName,
				BaseDir:  absBaseDir,
				LoomPath: absLoomPath,
				Nodes:    make(map[string]*node.Node),
			}

			var accounts []*node.Account
			for i := 0; i < numAccts; i++ {
				acct, err := node.CreateAccount(i, conf.BaseDir, conf.LoomPath)
				if err != nil {
					return err
				}
				accounts = append(accounts, acct)
			}

			var nodes []*node.Node
			for i := 0; i < len(nodeIDs); i++ {
				node := node.NewNode(int64(nodeIDs[i]), conf.BaseDir, conf.LoomPath, conf.ContractDir, "", "")
				node.LoadNodeKey()
				// remove the address book since we're going to allocate new port numbers & addresses
				// to all the nodes, and we don't want them trying to connect to remote nodes of the
				// original cluster
				addrBookFilePath := path.Join(node.Dir, "chaindata", "config", "addrbook.json")
				if _, err := os.Stat(addrBookFilePath); !os.IsNotExist(err) {
					os.Remove(addrBookFilePath)
				}
				nodes = append(nodes, node)
			}

			node.GenerateNodeAddresses(nodes)
			node.SetNodePeers(nodes)

			for _, node := range nodes {
				if err := node.UpdateTMConfig(); err != nil {
					return err
				}
				if err := node.UpdateLoomConfig(""); err != nil {
					return err
				}

				if err := ioutil.WriteFile(
					path.Join(node.Dir, "node_rpc_addr"),
					[]byte(fmt.Sprintf("127.0.0.1:%d", node.ProxyAppPort)),
					0644,
				); err != nil {
					return err
				}

				if privValHeight >= 0 {
					pv, err := privval.LoadFilePV(path.Join(node.Dir, "chaindata", "config", "priv_validator.json"))
					if err != nil {
						return err
					}

					pv.LastHeight = int64(privValHeight)
					pv.LastRound = 0
					pv.LastStep = 0
					pv.LastSignature = nil
					pv.LastSignBytes = nil
					pv.Save()
				}

				conf.Nodes[fmt.Sprintf("%d", node.ID)] = node
				conf.NodeAddressList = append(conf.NodeAddressList, node.Address)
				conf.NodePubKeyList = append(conf.NodePubKeyList, node.PubKey)
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

	cmdFlags := cmd.Flags()
	cmdFlags.IntSliceVar(&nodeIDs, "nodes", []int{}, "Comma separated list of node IDs")
	cmdFlags.StringVar(&clusterName, "name", "default", "Cluster name")
	cmdFlags.StringVar(&baseDir, "base-dir", "", "Base directory")
	cmdFlags.StringVar(&loomPath, "loom-path", "loom", "Loom binary path")
	cmdFlags.IntVarP(&numAccts, "account", "k", 1, "Number of account to be created")
	cmdFlags.IntVar(&privValHeight, "priv-val-height", -1, "Reset last_* fields in priv_validator.json")

	cmd.MarkFlagRequired("nodes")
	cmd.MarkFlagRequired("base-dir")
	return cmd
}
