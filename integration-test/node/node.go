package node

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/pkg/errors"
)

type Node struct {
	ID              int64
	Dir             string
	LoomPath        string
	ContractDir     string
	NodeKey         string
	PubKey          string
	Power           int64
	QueryServerHost string
	Address         string
	Local           string
}

func NewNode(ID int64, baseDir, loomPath, contractDir string) *Node {
	return &Node{
		ID:              ID,
		ContractDir:     contractDir,
		LoomPath:        loomPath,
		Dir:             path.Join(baseDir, fmt.Sprintf("%d", ID)),
		QueryServerHost: fmt.Sprintf("tcp://0.0.0.0:%d", 9000+ID),
	}
}

func (n *Node) Init() error {
	if err := os.MkdirAll(n.Dir, 0744); err != nil {
		return err
	}

	// linux copy smart contract: TODO to change to OS independent
	cp := exec.Command("cp", "-r", n.ContractDir, n.Dir)
	if err := cp.Run(); err != nil {
		return errors.Wrapf(err, "copy contract error")
	}

	// run init DPOS
	init := &exec.Cmd{
		Dir:  n.Dir,
		Path: n.LoomPath,
		Args: []string{n.LoomPath, "init", "-f"},
	}
	if err := init.Run(); err != nil {
		return errors.Wrapf(err, "init error")
	}
	// run nodekey
	nodekey := &exec.Cmd{
		Dir:  n.Dir,
		Path: n.LoomPath,
		Args: []string{n.LoomPath, "nodekey"},
	}
	out, err := nodekey.Output()
	if err != nil {
		return errors.Wrapf(err, "fail to run nodekey")
	}

	// update node key
	n.NodeKey = strings.TrimSpace(string(out))
	fmt.Printf("running loom init in directory: %s\n", n.Dir)
	return nil
}

// Run runs node forever
func (n *Node) Run(ctx context.Context) error {
	log, err := os.Create(path.Join(n.Dir, "log.log"))
	if err != nil {
		return err
	}
	defer log.Close()
	cmd := &exec.Cmd{
		Dir:    n.Dir,
		Path:   n.LoomPath,
		Args:   []string{n.LoomPath, "run"},
		Stdout: log,
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	fmt.Printf("starting loom node %d\n", n.ID)

	errC := make(chan error)
	go func() {
		select {
		case errC <- cmd.Wait():
		}
	}()

	select {
	case err := <-errC:
		fmt.Printf("err: %v\n", err)
		return err
	case <-ctx.Done():
		err := cmd.Process.Kill()
		if err != nil {
			fmt.Printf("kill process error: %v", err)
		}
		fmt.Printf("stopping loom node %d\n", n.ID)
	}

	return nil
}
