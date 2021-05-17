package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv2"
	d3types "github.com/loomnetwork/go-loom/builtin/types/dposv3"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain/config"
)

type Node struct {
	ID              int64
	Dir             string
	LoomPath        string
	ContractDir     string
	NodeKey         string
	PubKey          string
	PrivKeyPath     string
	Power           int64
	Address         string
	Local           string
	Peers           string
	PersistentPeers string
	LogLevel        string
	LogDestination  string
	LogAppDb        bool
	BaseGenesis     string
	BaseYaml        string
	RPCAddress      string
	ProxyAppAddress string
	Config          config.Config
}

func NewNode(ID int64, baseDir, loomPath, contractDir, genesisFile, yamlFile string) *Node {
	return &Node{
		ID:          ID,
		ContractDir: contractDir,
		LoomPath:    loomPath,
		Dir:         path.Join(baseDir, fmt.Sprintf("%d", ID)),
		BaseGenesis: genesisFile,
		BaseYaml:    yamlFile,
		Config:      *config.DefaultConfig(),
	}
}

func (n *Node) Init(accounts []*Account) error {
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
	if err := n.SetConfigFromYaml(accounts); err != nil {
		return errors.Wrapf(err, "reading loom yaml file %s", n.BaseYaml)
	}
	loomYamlPath := path.Join(n.Dir, "loom.yaml")
	if err := n.Config.WriteToFile(loomYamlPath); err != nil {
		return errors.Wrapf(err, "write config to %s", loomYamlPath)
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
			case "dposV3":
				var init d3types.DPOSInitRequest
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
					case "dposV3":
						var dposinit d3types.DPOSInitRequest
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
			Config:    baseGen.Config,
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

	// create private key file
	nodeKeyPath := path.Join(n.Dir, "/chaindata/config/priv_validator.json")
	nodeKeyData, err := ioutil.ReadFile(nodeKeyPath)
	if err != nil {
		return errors.Wrapf(err, "fail to read node key Data")
	}
	var objmap map[string]*json.RawMessage
	_ = json.Unmarshal(nodeKeyData, &objmap)
	var objmap2 map[string]*json.RawMessage
	_ = json.Unmarshal(*objmap["priv_key"], &objmap2)

	configPath := path.Join(n.Dir, "node_privkey")
	if err := ioutil.WriteFile(configPath, (*objmap2["value"])[1:(len(*objmap2["value"])-1)], 0644); err != nil {
		return errors.Wrap(err, "failed to write node_privekey")
	}

	return nil
}

// Run runs node forever
func (n *Node) Run(ctx context.Context, eventC chan *Event) error {
	//TODO it seems like we want to either dynamically generate the ports, or
	//have both the client and server give the previous test a few seconds to
	//start you can't simply put a sleep here cause the client to the
	//integration test needs to wait also
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
		errC <- cmd.Run()
	}()

	for {
		select {
		case event := <-eventC:
			delay := event.Delay.Duration
			time.Sleep(delay)
			switch event.Action {
			case ActionStop:
				if event.Node != int(n.ID) {
					eventC <- event
					continue
				}

				err := cmd.Process.Kill()
				if err != nil {
					fmt.Printf("error kill process: %v", err)
				}

				dur := event.Duration.Duration
				// consume error when killing process
				e := <-errC
				if e != nil {
					// check error
				}
				fmt.Printf("stopped node %d for %v\n", n.ID, dur)

				// restart
				time.Sleep(dur)
				cmd = exec.CommandContext(ctx, n.LoomPath, "run", "--persistent-peers", n.PersistentPeers)
				cmd.Dir = n.Dir
				cmd.Env = append(os.Environ(),
					"CONTRACT_LOG_DESTINATION=file://contract.log",
					"CONTRACT_LOG_LEVEL=debug",
				)
				cmd.Stderr = os.Stderr
				cmd.Stdout = os.Stdout
				go func() {
					fmt.Printf("starting node %d after %v\n", n.ID, dur)
					errC <- cmd.Run()
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

func (n *Node) SetConfigFromYaml(accounts []*Account) error {
	if len(n.BaseYaml) > 0 {
		conf, err := config.ParseConfigFrom(strings.TrimSuffix(n.BaseYaml, filepath.Ext(n.BaseYaml)))
		if err != nil {
			return err
		}

		addAccounts(accounts, conf.GoContractDeployerWhitelist.DeployerAddressList)
		n.Config = *conf
	}
	return nil
}

func addAccounts(accounts []*Account, list []string) {
	for i, strAddr := range list {
		acctId, err := strconv.ParseUint(strAddr, 10, 64)
		if err != nil || acctId >= uint64(len(accounts)) {
			continue
		}
		list[i] = accounts[acctId].Address
	}
}
