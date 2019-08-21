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
	cctypes "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	dwtypes "github.com/loomnetwork/go-loom/builtin/types/deployer_whitelist"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv2"
	d3types "github.com/loomnetwork/go-loom/builtin/types/dposv3"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/fnConsensus"
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

func CreateCluster(nodes []*Node, account []*Account, fnconsensus bool) error {
	// rewrite chaindata/config/genesis.json
	var genValidators []tmtypes.GenesisValidator
	for _, node := range nodes {
		genFile := path.Join(node.Dir, "chaindata", "config", "genesis.json")
		genDoc, err := tmtypes.GenesisDocFromFile(genFile)
		if err != nil {
			return err
		}

		genValidators = append(genValidators, genDoc.Validators...)
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
		nodeKeyFile := path.Join(node.Dir, "node_privkey")
		node.PrivKeyPath = nodeKeyFile
	}

	// Initialize the override validators
	overrideValidators := make([]*fnConsensus.OverrideValidatorParsable, 0, len(genValidators))
	for _, val := range genValidators {
		address := val.Address
		overrideValidators = append(overrideValidators, &fnConsensus.OverrideValidatorParsable{
			Address:     address.String(),
			VotingPower: 100,
		})
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

		//TODO we need a better way to update the configs
		str = strings.Replace(str, "recheck = true", "recheck = false", -1)

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

		if err := node.SetConfigFromYaml(account); err != nil {
			return errors.Wrapf(err, "reading loom yaml file %s", node.BaseYaml)
		}

		if fnconsensus {
			node.Config.FnConsensus.Enabled = true
			node.Config.FnConsensus.Reactor.FnVoteSigningThreshold = "All"
			node.Config.FnConsensus.Reactor.OverrideValidators = overrideValidators
		}

		node.Config.Peers = strings.Join(peers, ",")
		node.Config.PersistentPeers = strings.Join(persistentPeers, ",")
		node.Config.RPCProxyPort = int32(proxyAppPort)
		node.Config.BlockchainLogLevel = node.LogLevel
		node.Config.LogDestination = node.LogDestination
		node.Config.RPCListenAddress = fmt.Sprintf("tcp://127.0.0.1:%d", rpcPort)
		node.Config.RPCBindAddress = fmt.Sprintf("tcp://127.0.0.1:%d", proxyAppPort)
		if len(account) > 0 {
			node.Config.Oracle = "default:" + account[0].Address
		}

		configureGateways(&node.Config, proxyAppPort)

		node.Config.ChainConfig.DAppChainReadURI = fmt.Sprintf("http://127.0.0.1:%d/query", proxyAppPort)
		node.Config.ChainConfig.DAppChainWriteURI = fmt.Sprintf("http://127.0.0.1:%d/rpc", proxyAppPort)

		loomYamlPath := path.Join(node.Dir, "loom.yaml")
		if err := node.Config.WriteToFile(loomYamlPath); err != nil {
			return errors.Wrapf(err, "write config to %s", loomYamlPath)
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
			if contract.Name == "dposV3" {
				var init d3types.DPOSInitRequest
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
				// set new validators
				init.Validators = validators
				oracleAddr := loom.LocalAddressFromPublicKey(validators[0].PubKey)
				init.Params.OracleAddress = &types.Address{
					ChainId: "default",
					Local:   oracleAddr,
				}
				init.InitCandidates = false
				jsonInit, err := marshalInit(&init)
				if err != nil {
					return err
				}
				contract.Init = jsonInit
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
				oracleAddr := loom.LocalAddressFromPublicKey(validators[0].PubKey)
				init.Params.OracleAddress = &types.Address{
					ChainId: "default",
					Local:   oracleAddr,
				}
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
							Balance: 100000000,
						}
						init.Accounts = append(init.Accounts, account)
					}

					for _, validator := range validators {
						addr := &types.Address{
							ChainId: "default",
							Local:   loom.LocalAddressFromPublicKey(validator.PubKey),
						}
						account := &ctypes.InitialAccount{
							Owner:   addr,
							Balance: 100000000,
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
			case "chainconfig":
				var init cctypes.InitRequest
				unmarshaler, err := contractpb.UnmarshalerFactory(plugin.EncodingType_JSON)
				if err != nil {
					return err
				}
				buf := bytes.NewBuffer(contract.Init)
				if err := unmarshaler.Unmarshal(buf, &init); err != nil {
					return err
				}
				// set contract owner
				ownerAddr := loom.LocalAddressFromPublicKey(validators[0].PubKey)
				init.Owner = &types.Address{
					ChainId: "default",
					Local:   ownerAddr,
				}
				jsonInit, err := marshalInit(&init)
				if err != nil {
					return err
				}
				contract.Init = jsonInit
			case "deployerwhitelist":
				var init dwtypes.InitRequest
				unmarshaler, err := contractpb.UnmarshalerFactory(plugin.EncodingType_JSON)
				if err != nil {
					return err
				}
				buf := bytes.NewBuffer(contract.Init)
				if err := unmarshaler.Unmarshal(buf, &init); err != nil {
					return err
				}
				// set contract owner
				ownerAddr := loom.LocalAddressFromPublicKey(validators[0].PubKey)
				init.Owner = &types.Address{
					ChainId: "default",
					Local:   ownerAddr,
				}
				jsonInit, err := marshalInit(&init)
				if err != nil {
					return err
				}
				contract.Init = jsonInit
			case "binance-gateway":
				var init tgtypes.TransferGatewayInitRequest
				unmarshaler, err := contractpb.UnmarshalerFactory(plugin.EncodingType_JSON)
				if err != nil {
					return err
				}
				buf := bytes.NewBuffer(contract.Init)
				if err := unmarshaler.Unmarshal(buf, &init); err != nil {
					return err
				}
				// set contract owner
				ownerAddr := loom.LocalAddressFromPublicKey(validators[0].PubKey)
				init.Owner = &types.Address{
					ChainId: "default",
					Local:   ownerAddr,
				}
				jsonInit, err := marshalInit(&init)
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

	newContracts := make([]contractConfig, 0, len(gens.Contracts))
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
