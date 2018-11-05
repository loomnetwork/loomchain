package node

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv2"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/pkg/errors"
)

type Node struct {
	ID                     int64
	Dir                    string
	LoomPath               string
	ContractDir            string
	NodeKey                string
	PubKey                 string
	Power                  int64
	QueryServerHost        string
	Address                string
	Local                  string
	Peers                  string
	PersistentPeers        string
	LogLevel               string
	LogDestination         string
	LogAppDb               bool
	BaseGenesis            string
	BaseYaml               string
	RPCAddress             string
	ProxyAppAddress        string
}

func NewNode(ID int64, baseDir, loomPath, contractDir, genesisFile, yamlFile string) *Node {
	return &Node{
		ID:              ID,
		ContractDir:     contractDir,
		LoomPath:        loomPath,
		Dir:             path.Join(baseDir, fmt.Sprintf("%d", ID)),
		QueryServerHost: fmt.Sprintf("tcp://127.0.0.1:%d", portGen.Next()),
		BaseGenesis:     genesisFile,
		BaseYaml:        yamlFile,
	}
}

func (n *Node) Init() error {
	if err := os.MkdirAll(n.Dir, 0744); err != nil {
		return err
	}

	// linux copy smart contract: TODO to change to OS independent
	if n.ContractDir != "" {
		cp := exec.Command("cp", "-r", n.ContractDir, n.Dir)
		if err := cp.Run(); err != nil {
			return errors.Wrapf(err, "copy contract error")
		}
	}

	// copy base loom.yaml (if there is one) to the node directory so that the node takes it into
	// account when generating the default genesis
	if len(n.BaseYaml) > 0 {
		baseYaml, err := ioutil.ReadFile(n.BaseYaml)
		if err != nil {
			return errors.Wrap(err, "failed to read base loom.yaml file")
		}

		configPath := path.Join(n.Dir, "loom.yaml")
		if err := ioutil.WriteFile(configPath, baseYaml, 0644); err != nil {
			return errors.Wrap(err, "failed to write loom.yaml")
		}
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

	// If there is base genesis, we use the base genesis as a starting point.
	// And then we looking for the autogen genesis from loom to grap settings from it.
	// Finally, we're gonna write a new genesis file using base genesis with the settings
	// from autogen genesis.
	if n.BaseGenesis != "" {
		gens, err := readGenesis(path.Join(n.Dir, "genesis.json"))
		if err != nil {
			return err
		}
		baseGen, err := readGenesis(n.BaseGenesis)
		if err != nil {
			return err
		}
		var newContracts []contractConfig
		for _, contract := range baseGen.Contracts {
			switch contract.Name {
			case "dposV2":
				var init dtypes.DPOSInitRequestV2
				unmarshaler, err := contractpb.UnmarshalerFactory(plugin.EncodingType_JSON)
				if err != nil {
					return err
				}
				buf := bytes.NewBuffer(contract.Init)
				if err := unmarshaler.Unmarshal(buf, &init); err != nil {
					return err
				}

				// copy other settings from generated genesis file
				for _, c := range gens.Contracts {
					switch c.Name {
					case "dposV2":
						var dposinit dtypes.DPOSInitRequestV2
						unmarshaler, err := contractpb.UnmarshalerFactory(plugin.EncodingType_JSON)
						if err != nil {
							return err
						}
						buf := bytes.NewBuffer(c.Init)
						if err := unmarshaler.Unmarshal(buf, &dposinit); err != nil {
							return err
						}
						// set new validators
						init.Validators = dposinit.Validators
					default:
					}
				}

				// set init to contract
				jsonInit, err := marshalInit(&init)
				if err != nil {
					return err
				}
				contract.Init = jsonInit
			}

			newContracts = append(newContracts, contract)
		}

		newGenesis := &genesis{
			Contracts: newContracts,
		}

		err = writeGenesis(newGenesis, path.Join(n.Dir, "genesis.json"))
		if err != nil {
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
	cmd := exec.CommandContext(ctx, n.LoomPath, "run", "--persistent-peers", n.PersistentPeers)
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
