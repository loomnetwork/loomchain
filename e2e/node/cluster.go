package node

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	loom "github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dpos"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	tmtypes "github.com/tendermint/tendermint/types"
	yaml "gopkg.in/yaml.v2"
)

func CreateCluster(nodes []*Node, account []*Account) error {
	// rewrite chaindata/config/genesis.json
	var genValidators []tmtypes.GenesisValidator
	for _, node := range nodes {
		genFile := path.Join(node.Dir, "chaindata", "config", "genesis.json")
		genDoc, err := tmtypes.GenesisDocFromFile(genFile)
		if err != nil {
			return err
		}

		for _, val := range genDoc.Validators {
			genValidators = append(genValidators, val)
		}
	}
	for _, node := range nodes {
		genFile := path.Join(node.Dir, "chaindata", "config", "genesis.json")
		genDoc, err := tmtypes.GenesisDocFromFile(genFile)
		if err != nil {
			return err
		}
		genDoc.Validators = genValidators
		err = genDoc.SaveAs(genFile)
		if err != nil {
			return err
		}
	}

	idToP2P := make(map[int64]string)
	for _, node := range nodes {
		// HACK: change rpc and p2p listen address so we can run it locally
		configPath := path.Join(node.Dir, "chaindata", "config", "config.toml")
		data, err := ioutil.ReadFile(configPath)
		if err != nil {
			return err
		}
		str := string(data)
		// running port number 46x57 e.g. 46657, 46757, 46857, ...
		rpcLaddr := fmt.Sprintf("tcp://127.0.0.1:%d", 46657+(node.ID*100))
		// running port number 46x56 e.g. 46656, 46756, 46856, ...
		p2pLaddr := fmt.Sprintf("127.0.0.1:%d", 46656+(node.ID*100))
		// replace config
		str = strings.Replace(str, "tcp://0.0.0.0:46657", rpcLaddr, -1)
		str = strings.Replace(str, "tcp://0.0.0.0:46656", p2pLaddr, -1)

		err = ioutil.WriteFile(configPath, []byte(str), 0644)
		if err != nil {
			return err
		}
		idToP2P[node.ID] = p2pLaddr
		node.RPCAddress = fmt.Sprintf("http://127.0.0.1:%d", 46658+(node.ID*100))
		node.ABCIAddress = fmt.Sprintf("http://127.0.0.1:%d", 46657+(node.ID*100))
	}

	idToValidator := make(map[int64]*types.Validator)
	for _, node := range nodes {
		var peers []string
		var persistentPeers []string
		for _, n := range nodes {
			if node.ID != n.ID {
				peers = append(peers, fmt.Sprintf("tcp://%s@%s", n.NodeKey, idToP2P[n.ID]))
				persistentPeers = append(persistentPeers, fmt.Sprintf("tcp://%s@%s", n.NodeKey, idToP2P[n.ID]))
			}
		}
		node.Peers = strings.Join(peers, ",")
		node.PersistentPeers = strings.Join(persistentPeers, ",")

		var config = struct {
			QueryServerHost    string
			Peers              string
			PersistentPeers    string
			RPCProxyPort       int32
			BlockchainLogLevel string
			LogAppDb           bool
			LogDestination     string
		}{
			QueryServerHost:    fmt.Sprintf("tcp://0.0.0.0:%d", 9000+node.ID),
			Peers:              strings.Join(peers, ","),
			PersistentPeers:    strings.Join(persistentPeers, ","),
			RPCProxyPort:       int32(46658 + (node.ID * 100)),
			BlockchainLogLevel: node.LogLevel,
			LogDestination:     node.LogDestination,
			LogAppDb:           node.LogAppDb,
		}

		buf := new(bytes.Buffer)
		if err := yaml.NewEncoder(buf).Encode(config); err != nil {
			return err
		}

		configPath := path.Join(node.Dir, "loom.yaml")
		if err := ioutil.WriteFile(configPath, buf.Bytes(), 0644); err != nil {
			return err
		}

		genesis, err := readGenesis(path.Join(node.Dir, "genesis.json"))
		if err != nil {
			return err
		}
		for _, contract := range genesis.Contracts {
			if contract.Name == "dpos" {
				var init dtypes.DPOSInitRequest
				unmarshaler, err := contractpb.UnmarshalerFactory(plugin.EncodingType_JSON)
				if err != nil {
					return err
				}
				buf := bytes.NewBuffer(contract.Init)
				if err := unmarshaler.Unmarshal(buf, &init); err != nil {
					return err
				}
				if len(init.Validators) > 0 {
					idToValidator[node.ID] = init.Validators[0]
				}
			}
		}
	}

	var validators []*types.Validator
	encoder := base64.StdEncoding
	for _, node := range nodes {
		validator := idToValidator[node.ID]
		if validator != nil {
			address := loom.LocalAddressFromPublicKey(validator.PubKey)
			node.PubKey = encoder.EncodeToString(validator.PubKey)
			node.Address = address.String()
			node.Power = validator.Power
			node.Local = encoder.EncodeToString(address)
			validators = append(validators, validator)
		}
	}
	// rewrite genesis
	for _, node := range nodes {
		gens, err := readGenesis(path.Join(node.Dir, "genesis.json"))
		if err != nil {
			return err
		}
		var newContracts []contractConfig
		for _, contract := range gens.Contracts {
			switch contract.Name {
			case "dpos":
				var init dtypes.DPOSInitRequest
				unmarshaler, err := contractpb.UnmarshalerFactory(plugin.EncodingType_JSON)
				if err != nil {
					return err
				}
				buf := bytes.NewBuffer(contract.Init)
				if err := unmarshaler.Unmarshal(buf, &init); err != nil {
					return err
				}
				// set new validators
				init.Validators = validators
				// contract.Init = init
				jsonInit, err := marshalInit(&init)
				if err != nil {
					return err
				}
				contract.Init = jsonInit
			case "coin":
				var init ctypes.InitRequest
				unmarshaler, err := contractpb.UnmarshalerFactory(plugin.EncodingType_JSON)
				if err != nil {
					return err
				}
				buf := bytes.NewBuffer(contract.Init)
				if err := unmarshaler.Unmarshal(buf, &init); err != nil {
					return err
				}
				// set initial coint to account node 0
				if len(init.Accounts) == 0 {
					for _, acct := range account {
						address, err := loom.LocalAddressFromHexString(acct.Address)
						if err != nil {
							return err
						}
						addr := &types.Address{
							ChainId: "default",
							Local:   address,
						}
						account := &ctypes.InitialAccount{
							Owner:   addr,
							Balance: 100,
						}
						init.Accounts = append(init.Accounts, account)
					}

					jsonInit, err := marshalInit(&init)
					if err != nil {
						return err
					}
					contract.Init = jsonInit
				}
			case "BluePrint":
				jsonInit := json.RawMessage(nil)
				contract.Init = jsonInit
			// in case we need to define custom setups for a new contract, insert
			// a new case here
			default:
			}

			newContracts = append(newContracts, contract)
		}

		newGenesis := &genesis{
			Contracts: newContracts,
		}

		err = writeGenesis(newGenesis, path.Join(node.Dir, "genesis.json"))
		if err != nil {
			return err
		}
	}

	return nil
}

func GenesisFromTemplate(genfile string, outfile string, account ...*Account) error {
	// create genesis file
	gens, err := readGenesis(genfile)
	if err != nil {
		return err
	}
	var newContracts []contractConfig
	for _, contract := range gens.Contracts {
		switch contract.Name {
		case "coin":
			var init ctypes.InitRequest
			unmarshaler, err := contractpb.UnmarshalerFactory(plugin.EncodingType_JSON)
			if err != nil {
				return err
			}
			buf := bytes.NewBuffer(contract.Init)
			if err := unmarshaler.Unmarshal(buf, &init); err != nil {
				return err
			}
			// set initial coint to account node 0
			for _, acct := range account {
				address, err := loom.LocalAddressFromHexString(acct.Address)
				if err != nil {
					return err
				}
				addr := &types.Address{
					ChainId: "default",
					Local:   address,
				}
				account := &ctypes.InitialAccount{
					Owner:   addr,
					Balance: 100,
				}
				init.Accounts = append(init.Accounts, account)
			}

			jsonInit, err := marshalInit(&init)
			if err != nil {
				return err
			}
			contract.Init = jsonInit
		default:
		}

		newContracts = append(newContracts, contract)
	}

	newGenesis := &genesis{
		Contracts: newContracts,
	}

	err = writeGenesis(newGenesis, outfile)
	return err
}
