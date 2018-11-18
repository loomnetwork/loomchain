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
)

// global port generators
var (
	portGen *portGenerator
)

func init() {
	portGen = &portGenerator{}
}

func GenerateNodeAddresses(nodes []*Node) {
	for _, node := range nodes {
		node.QueryServerHost = fmt.Sprintf("tcp://127.0.0.1:%d", portGen.Next())
		node.RPCPort = portGen.Next()
		node.P2PPort = portGen.Next()
		node.ProxyAppPort = portGen.Next()
		node.ProxyAppAddress = fmt.Sprintf("http://127.0.0.1:%d", node.ProxyAppPort)
		node.RPCAddress = fmt.Sprintf("http://127.0.0.1:%d", node.RPCPort)
	}
}

func SetNodePeers(nodes []*Node) {
	for _, node := range nodes {
		var peers []string
		var persistentPeers []string
		for _, n := range nodes {
			if node.ID != n.ID {
				peers = append(peers, fmt.Sprintf("tcp://%s@127.0.0.1:%d", n.NodeKey, n.P2PPort))
				persistentPeers = append(persistentPeers, fmt.Sprintf("tcp://%s@127.0.0.1:%d", n.NodeKey, n.P2PPort))
			}
		}
		node.Peers = strings.Join(peers, ",")
		node.PersistentPeers = strings.Join(persistentPeers, ",")
	}
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

	GenerateNodeAddresses(nodes)
	SetNodePeers(nodes)

	idToValidator := make(map[int64]*types.Validator)
	for _, node := range nodes {
		node.UpdateTMConfig()
		node.UpdateLoomConfig("default:" + account[0].Address)

		if err := ioutil.WriteFile(
			path.Join(node.Dir, "node_rpc_addr"),
			[]byte(fmt.Sprintf("127.0.0.1:%d", node.ProxyAppPort)),
			0644,
		); err != nil {
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
