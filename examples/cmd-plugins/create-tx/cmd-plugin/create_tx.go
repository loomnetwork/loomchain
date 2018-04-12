package cmdplugins

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/loom"
	"github.com/loomnetwork/loom/auth"
	"github.com/loomnetwork/loom/cli"
	"golang.org/x/crypto/ed25519"
)

const (
	nodeUriFlag = "node"
)

// CreateTxCmdPlugin is a sample admin CLI cmd plugin that creates a new dummy tx & commits it to the DAppChain.
type CreateTxCmdPlugin struct {
	cmdPluginSystem cli.CmdPluginSystem
}

func (c *CreateTxCmdPlugin) Init(sys cli.CmdPluginSystem) error {
	c.cmdPluginSystem = sys
	return nil
}

func (c *CreateTxCmdPlugin) GetCmds() []*cli.Cmd {
	cmd := &cli.Cmd{
		Use:   "create-tx <value>",
		Short: "Create & commit a dummy tx to the DAppChain",
		Args:  cli.ExactArgs(1),
		RunE:  c.runCmd,
	}
	cmd.Flags().StringP(
		nodeUriFlag, "n", "tcp://0.0.0.0:46657",
		"URI of node to administer, in the form tcp://<host>:<port>")
	return []*cli.Cmd{cmd}
}

func (c *CreateTxCmdPlugin) runCmd(cmd *cli.Cmd, args []string) error {
	nodeUri, err := cmd.Flags().GetString(nodeUriFlag)
	if err != nil {
		return err
	}
	client, err := c.cmdPluginSystem.GetClient(nodeUri)
	if err != nil {
		return err
	}
	dummyValue := args[0]
	dummyTx := loom.DummyTx{
		Key: "hello",
		Val: dummyValue,
	}
	txBytes, err := proto.Marshal(&dummyTx)
	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return err
	}
	signer := auth.NewEd25519Signer(privKey)
	client.CommitTx(signer, txBytes)
	return nil
}
