package node

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv2"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/pkg/errors"
	tmtypes "github.com/tendermint/tendermint/types"
	"gopkg.in/yaml.v2"
)

// global port generators
var (
	portGen *portGenerator
)

func init() {
	portGen = &portGenerator{}
}

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

	var genesisTime time.Time
	for i, node := range nodes {
		genFile := path.Join(node.Dir, "chaindata", "config", "genesis.json")
		genDoc, err := tmtypes.GenesisDocFromFile(genFile)
		if err != nil {
			return err
		}
		// timestamp in TM genesis file must match across all nodes in the cluster
		if i == 0 {
			genesisTime = genDoc.GenesisTime
		} else {
			genDoc.GenesisTime = genesisTime
		}
		genDoc.Validators = genValidators
		err = genDoc.SaveAs(genFile)
		if err != nil {
			return err
		}
	}

	idToP2P := make(map[int64]string)
	idToRPCPort := make(map[int64]int)
	idToProxyPort := make(map[int64]int)
	for _, node := range nodes {
		// HACK: change rpc and p2p listen address so we can run it locally
		configPath := path.Join(node.Dir, "chaindata", "config", "config.toml")
		data, err := ioutil.ReadFile(configPath)
		if err != nil {
			return err
		}
		str := string(data)
		rpcPort := portGen.Next()
		p2pPort := portGen.Next()
		proxyAppPort := portGen.Next()
		rpcLaddr := fmt.Sprintf("tcp://127.0.0.1:%d", rpcPort)
		p2pLaddr := fmt.Sprintf("127.0.0.1:%d", p2pPort)
		proxyAppPortAddr := fmt.Sprintf("tcp://127.0.0.1:%d", proxyAppPort)
		// replace config
		str = strings.Replace(str, "tcp://0.0.0.0:46657", rpcLaddr, -1)
		str = strings.Replace(str, "tcp://0.0.0.0:46656", p2pLaddr, -1)
		str = strings.Replace(str, "tcp://0.0.0.0:26657", rpcLaddr, -1) //Temp here cause now tendermint is 2xx range
		str = strings.Replace(str, "tcp://0.0.0.0:26656", p2pLaddr, -1) //Temp here cause now tendermint is 2xx range
		str = strings.Replace(str, "tcp://127.0.0.1:46658", proxyAppPortAddr, -1)
		str = strings.Replace(str, "tcp://127.0.0.1:26658", proxyAppPortAddr, -1) //Temp here cause now tendermint i
		err = ioutil.WriteFile(configPath, []byte(str), 0644)
		if err != nil {
			return err
		}
		if err := ioutil.WriteFile(
			path.Join(node.Dir, "node_rpc_addr"),
			[]byte(fmt.Sprintf("127.0.0.1:%d", proxyAppPort)),
			0644,
		); err != nil {
			return err
		}

		idToP2P[node.ID] = p2pLaddr
		idToRPCPort[node.ID] = rpcPort
		idToProxyPort[node.ID] = proxyAppPort
		node.ProxyAppAddress = fmt.Sprintf("http://127.0.0.1:%d", proxyAppPort)
		node.RPCAddress = fmt.Sprintf("http://127.0.0.1:%d", rpcPort)
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

		rpcPort := idToRPCPort[node.ID]
		proxyAppPort := idToProxyPort[node.ID]
		var config = struct {
			QueryServerHost    string
			Peers              string
			PersistentPeers    string
			RPCProxyPort       int32
			RPCPort            int32
			BlockchainLogLevel string
			LogAppDb           bool
			LogDestination     string
			RPCListenAddress   string
			RPCBindAddress     string
			Oracle             string
		}{
			QueryServerHost:    fmt.Sprintf("tcp://127.0.0.1:%d", portGen.Next()),
			Peers:              strings.Join(peers, ","),
			PersistentPeers:    strings.Join(persistentPeers, ","),
			RPCProxyPort:       int32(proxyAppPort),
			RPCPort:            int32(rpcPort),
			BlockchainLogLevel: node.LogLevel,
			LogDestination:     node.LogDestination,
			LogAppDb:           node.LogAppDb,
			RPCListenAddress:   fmt.Sprintf("tcp://127.0.0.1:%d", rpcPort),
			RPCBindAddress:     fmt.Sprintf("tcp://127.0.0.1:%d", proxyAppPort),
			Oracle:             "default:" + account[0].Address,
		}

		buf := new(bytes.Buffer)
		if err := yaml.NewEncoder(buf).Encode(config); err != nil {
			return err
		}

		if len(node.BaseYaml) > 0 {
			baseYaml, err := ioutil.ReadFile(node.BaseYaml)
			if err != nil {
				return errors.Wrap(err, "reading base yaml file")
			}

			_, err = buf.Write(baseYaml)
			if err != nil {
				return errors.Wrap(err, "concatenating yaml file")
			}
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
			if contract.Name == "dposV2" {
				var init dtypes.DPOSInitRequestV2
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
			case "karma":
				jsonInit, err := modifyKarmaInit(contract.Init, account)
				if err != nil {
					return err
				}
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

func modifyKarmaInit(contractInit json.RawMessage, accounts []*Account) (json.RawMessage, error) {
	// Start karma off with an oracle
	var init ktypes.KarmaInitRequest
	unmarshaler, err := contractpb.UnmarshalerFactory(plugin.EncodingType_JSON)
	if err != nil {
		return []byte{}, err
	}
	buf := bytes.NewBuffer(contractInit)
	if err := unmarshaler.Unmarshal(buf, &init); err != nil {
		return []byte{}, err
	}

	if len(accounts) < 2 {
		return []byte{}, errors.New("karma: not enough accounts")
	}

	localOracle, err := loom.LocalAddressFromHexString(accounts[0].Address)
	if err != nil {
		return []byte{}, errors.Wrap(err, "karma: getting oracle address")
	}
	init.Oracle = &types.Address{
		ChainId: "default",
		Local:   localOracle,
	}
	return marshalInit(&init)
}
