package node

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

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
	Peers           string
	PersistentPeers string
	LogLevel        string
	LogDestination  string
	BaseGenesis     string
	RPCAddress      string
}

func NewNode(ID int64, baseDir, loomPath, contractDir, genesisFile string) *Node {
	return &Node{
		ID:              ID,
		ContractDir:     contractDir,
		LoomPath:        loomPath,
		Dir:             path.Join(baseDir, fmt.Sprintf("%d", ID)),
		QueryServerHost: fmt.Sprintf("tcp://0.0.0.0:%d", 9000+ID),
		BaseGenesis:     genesisFile,
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

	// run init
	init := &exec.Cmd{
		Dir:  n.Dir,
		Path: n.LoomPath,
		Args: []string{n.LoomPath, "init", "-f"},
	}
	if err := init.Run(); err != nil {
		return errors.Wrapf(err, "init error")
	}
	// replace with base genesis if given
	if n.BaseGenesis != "" {
		data, err := ioutil.ReadFile(n.BaseGenesis)
		if err != nil {
			return err
		}
		genFile := path.Join(n.Dir, "genesis.json")
		if err := ioutil.WriteFile(genFile, data, 0644); err != nil {
			return err
		}
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
func (n *Node) Run(ctx context.Context, eventC chan *Event) error {
	fmt.Printf("starting loom node %d\n", n.ID)
	cmd := exec.CommandContext(ctx, n.LoomPath, "run", "--peers", n.Peers, "--persistent-peers", n.PersistentPeers)
	cmd.Dir = n.Dir
	cmd.Env = append(os.Environ(),
		"CONTRACT_LOG_DESTINATION=file://contract.log",
		"CONTRACT_LOG_LEVEL=debug",
	)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	errC := make(chan error)
	go func() {
		select {
		case errC <- cmd.Run():
		}
	}()

	for {
		select {
		case event := <-eventC:
			delay := event.Delay.Duration
			time.Sleep(delay)
			switch event.Action {
			case ActionStop:
				err := cmd.Process.Kill()
				if err != nil {
					fmt.Printf("error kill process: %v", err)
				}

				dur := event.Duration.Duration
				// consume error when killing process
				select {
				case e := <-errC:
					if e != nil {
						// check error
					}
					fmt.Printf("stopped node %d for %v\n", n.ID, dur)
				}

				// restart
				time.Sleep(dur)
				cmd = exec.CommandContext(ctx, n.LoomPath, "run")
				cmd.Dir = n.Dir
				go func() {
					fmt.Printf("starting node %d after %v\n", n.ID, dur)
					select {
					case errC <- cmd.Run():
					}
				}()
			}
		case err := <-errC:
			if err != nil {
				fmt.Printf("node %d error %v\n", n.ID, err)
			}
			return err
		case <-ctx.Done():
			fmt.Printf("stopping loom node %d\n", n.ID)
			return nil
		}
	}
}
