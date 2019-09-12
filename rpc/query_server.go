package rpc

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv3"
	sha3 "github.com/miguelmota/go-solidity-sha3"
	"github.com/phonkee/go-pubsub"
	"github.com/pkg/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	rpctypes "github.com/tendermint/tendermint/rpc/lib/types"

	"github.com/loomnetwork/go-loom"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv3"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/builtin/plugins/ethcoin"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/eth/polls"
	"github.com/loomnetwork/loomchain/eth/query"
	"github.com/loomnetwork/loomchain/eth/subs"
	"github.com/loomnetwork/loomchain/eth/utils"
	levm "github.com/loomnetwork/loomchain/evm"
	"github.com/loomnetwork/loomchain/log"
	lcp "github.com/loomnetwork/loomchain/plugin"
	hsmpv "github.com/loomnetwork/loomchain/privval/hsm"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/registry"
	registryFac "github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/rpc/blockatlas"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/store"
	blockindex "github.com/loomnetwork/loomchain/store/block_index"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
	lvm "github.com/loomnetwork/loomchain/vm"
)

type (
	CandidateStatistic = dtypes.CandidateStatistic
	ListCandidates     = dtypes.ListCandidatesResponse
)

var ErrNotFound = errors.New("not found")

const (
	/**
	 * contract GoContract {}
	 */
	goGetCode = "0x608060405260043610603f576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff168063f6b4dfb4146044575b600080fd5b348015604f57600080fd5b5060566098565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b73e288d6eec7150d6a22fde33f0aa2d81e06591c4d815600a165627a7a72305820b8b6992011e1a3286b9546ca427bf9cb05db8bd25addbee7a9894131d9db12500029"

	StatusTxSuccess = int32(1)
	StatusTxFail    = int32(0)
)

// StateProvider interface is used by QueryServer to access the read-only application state
type StateProvider interface {
	ReadOnlyState() loomchain.State
}

// QueryServer provides the ability to query the current state of the DAppChain via RPC.
//
// Contract state can be queried via:
// - POST request of a JSON-RPC 2.0 object to "/" endpoint:
//   {
//     "jsonrpc": "2.0",
//     "method": "query",
//     "params": {
//       "contract": "0x000000000000000000",
//       "query": { /* query params */ }
//     },
//     "id": "123456789"
//   }
// - POST request to "/query" endpoint with form-encoded contract & query params.
//
// Contract query requests must contain two parameters:
// - contract: the address of the contract to be queried (hex encoded string), and
// - query: a JSON object containing the query parameters, the Loom SDK makes no assumptions about
//          the structure of the object, it is entirely up to the contract author to define the
//          query interface.
//
// The JSON-RPC 2.0 response object will contain the query result as a JSON object:
// {
//   "jsonrpc": "2.0",
//   "result": { /* query result */ },
//   "id": "123456789"
// }
//
// On error the JSON-RPC 2.0 response object will look similar to this:
// {
//   "jsonrpc": "2.0",
//   "error": {
//	   "code": -32603,
//	   "message": "Internal error",
//	   "data": "invalid query"
//   },
//   "id": "123456789"
// }
//
// The nonce associated with a particular signer can be obtained via:
// - GET request to /nonce?key="<hex-encoded-public-key-of-signer>"
// - POST request of a JSON-RPC 2.0 object to "/" endpoint:
//   {
//     "jsonrpc": "2.0",
//     "method": "nonce",
//     "params": {
//       "key": "hex-encoded-public-key-of-signer",
//     },
//     "id": "123456789"
//   }
// - POST request to "/nonce" endpoint with form-encoded key param.
type QueryServer struct {
	StateProvider
	ChainID                string
	Loader                 lcp.Loader
	Subscriptions          *loomchain.SubscriptionSet
	EthSubscriptions       *subs.EthSubscriptionSet
	EthLegacySubscriptions *subs.LegacyEthSubscriptionSet
	EthPolls               polls.EthSubscriptions
	CreateRegistry         registryFac.RegistryFactoryFunc
	// If this is nil the EVM won't have access to any account balances.
	NewABMFactory lcp.NewAccountBalanceManagerFactoryFunc
	loomchain.ReceiptHandlerProvider
	RPCListenAddress string
	store.BlockStore
	*evmaux.EvmAuxStore
	blockindex.BlockIndexStore
	EventStore store.EventStore
	AuthCfg    *auth.Config
}

var _ QueryService = &QueryServer{}

// Query returns data of given contract from the application states
// The contract parameter should be a hex-encoded local address prefixed by 0x
func (s *QueryServer) Query(caller, contract string, query []byte, vmType vm.VMType) ([]byte, error) {
	var callerAddr loom.Address
	var err error
	if len(caller) == 0 {
		callerAddr = loom.RootAddress(s.ChainID)
	} else {
		callerAddr, err = loom.ParseAddress(caller)
		if err != nil {
			return nil, err
		}
	}

	localContractAddr, err := decodeHexAddress(contract)
	if err != nil {
		return nil, err
	}
	contractAddr := loom.Address{
		ChainID: s.ChainID,
		Local:   localContractAddr,
	}

	if vmType == lvm.VMType_PLUGIN {
		return s.queryPlugin(callerAddr, contractAddr, query)
	} else {
		return s.queryEvm(callerAddr, contractAddr, query)
	}
}

func (s *QueryServer) QueryEnv() (*config.EnvInfo, error) {
	cfg, err := config.ParseConfig()
	if err != nil {
		return nil, err
	}

	gen, err := config.ReadGenesis(cfg.GenesisPath())
	if err != nil {
		return nil, err
	}

	envir := config.Env{
		Version:         loomchain.FullVersion(),
		Build:           loomchain.Build,
		BuildVariant:    loomchain.BuildVariant,
		GitSha:          loomchain.GitSHA,
		GoLoom:          loomchain.GoLoomGitSHA,
		TransferGateway: loomchain.TransferGatewaySHA,
		GoEthereum:      loomchain.EthGitSHA,
		GoPlugin:        loomchain.HashicorpGitSHA,
		Btcd:            loomchain.BtcdGitSHA,
		PluginPath:      cfg.PluginsPath(),
		Peers:           cfg.Peers,
	}

	// scrub the HSM config just in case
	cfg.HsmConfig = &hsmpv.HsmConfig{
		HsmEnabled: cfg.HsmConfig.HsmEnabled,
		HsmDevType: cfg.HsmConfig.HsmDevType,
	}

	envInfo := config.EnvInfo{
		Env:         envir,
		LoomGenesis: *gen,
		LoomConfig:  *cfg,
	}

	return &envInfo, err
}

func (s *QueryServer) queryPlugin(caller, contract loom.Address, query []byte) ([]byte, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	callerAddr, err := auth.ResolveAccountAddress(caller, snapshot, s.AuthCfg, s.createAddressMapperCtx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve account address")
	}

	vm := lcp.NewPluginVM(
		s.Loader,
		snapshot,
		s.CreateRegistry(snapshot),
		nil,
		log.Default,
		s.NewABMFactory,
		nil,
		nil,
	)
	req := &plugin.Request{
		ContentType: plugin.EncodingType_PROTOBUF3,
		Accept:      plugin.EncodingType_PROTOBUF3,
		Body:        query,
	}
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}

	respBytes, err := vm.StaticCall(callerAddr, contract, reqBytes)
	if err != nil {
		return nil, err
	}
	resp := &plugin.Response{}
	err = proto.Unmarshal(respBytes, resp)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (s *QueryServer) queryEvm(caller, contract loom.Address, query []byte) ([]byte, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	callerAddr, err := auth.ResolveAccountAddress(caller, snapshot, s.AuthCfg, s.createAddressMapperCtx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve account address")
	}

	var createABM levm.AccountBalanceManagerFactoryFunc
	if s.NewABMFactory != nil {
		pvm := lcp.NewPluginVM(
			s.Loader,
			snapshot,
			s.CreateRegistry(snapshot),
			nil,
			log.Default,
			s.NewABMFactory,
			nil,
			nil,
		)
		createABM, err = s.NewABMFactory(pvm)
		if err != nil {
			return nil, err
		}
	}
	vm := levm.NewLoomVm(snapshot, nil, nil, createABM, false)
	return vm.StaticCall(callerAddr, contract, query)
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_call
func (s *QueryServer) EthCall(query eth.JsonTxCallObject, block eth.BlockHeight) (resp eth.Data, err error) {
	var caller loom.Address
	if len(query.From) > 0 {
		caller, err = eth.DecDataToAddress(s.ChainID, query.From)
		if err != nil {
			return resp, err
		}
	} else {
		caller = loom.RootAddress(s.ChainID)
	}
	contract, err := eth.DecDataToAddress(s.ChainID, query.To)
	if err != nil {
		return resp, err
	}
	data, err := eth.DecDataToBytes(query.Data)
	if err != nil {
		return resp, err
	}
	bytes, err := s.queryEvm(caller, contract, data)
	return eth.EncBytes(bytes), err
}

// GetCode returns the runtime byte-code of a contract running on a DAppChain's EVM.
// Gives an error for non-EVM contracts.
// contract - address of the contract in the form of a string. (Use loom.Address.String() to convert)
// return []byte - runtime bytecode of the contract.
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getcode
func (s *QueryServer) GetEvmCode(contract string) ([]byte, error) {
	contractAddr, err := loom.ParseAddress(contract)
	if err != nil {
		return nil, err
	}

	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	vm := levm.NewLoomVm(snapshot, nil, nil, nil, false)
	return vm.GetCode(contractAddr)
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getcode
func (s *QueryServer) EthGetCode(address eth.Data, block eth.BlockHeight) (eth.Data, error) {
	addr, err := eth.DecDataToAddress(s.ChainID, address)
	if err != nil {
		return "", errors.Wrapf(err, "decoding input address parameter %v", address)
	}

	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	evm := levm.NewLoomVm(snapshot, nil, nil, nil, false)
	code, err := evm.GetCode(addr)
	if err != nil {
		return "", errors.Wrapf(err, "getting evm code for %v", address)
	}
	if code == nil {
		reg := s.CreateRegistry(snapshot)
		_, err := reg.GetRecord(addr)
		if err != nil {
			return eth.ZeroedData, nil
		}
		return eth.Data(goGetCode), nil
	}
	return eth.EncBytes(code), nil
}

// Attempts to construct the context of the Address Mapper contract.
func (s *QueryServer) createAddressMapperCtx(state loomchain.State) (contractpb.StaticContext, error) {
	return s.createStaticContractCtx(state, "addressmapper")
}

func (s *QueryServer) createStaticContractCtx(state loomchain.State, name string) (contractpb.StaticContext, error) {
	ctx, err := lcp.NewInternalContractContext(
		name,
		lcp.NewPluginVM(
			s.Loader,
			state,
			s.CreateRegistry(state),
			nil, // event handler
			log.Default,
			s.NewABMFactory,
			nil, // receipt writer
			nil, // receipt reader
		),
		true,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create %s context", name)
	}
	return ctx, nil
}

// Nonce returns the nonce of the last committed tx sent by the given account.
// NOTE: Either the key or the account must be provided. The account (if not empty) is used in
//       preference to the key.
func (s *QueryServer) Nonce(key, account string) (uint64, error) {
	var addr loom.Address

	if key != "" && account == "" {
		k, err := hex.DecodeString(key)
		if err != nil {
			return 0, err
		}
		addr = loom.Address{
			ChainID: s.ChainID,
			Local:   loom.LocalAddressFromPublicKey(k),
		}
	} else if account != "" {
		var err error
		addr, err = loom.ParseAddress(account)
		if err != nil {
			return 0, err
		}
	} else {
		return 0, errors.New("no key or account specified")
	}

	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	resolvedAddr, err := auth.ResolveAccountAddress(addr, snapshot, s.AuthCfg, s.createAddressMapperCtx)
	if err != nil {
		return 0, errors.Wrap(err, "failed to resolve account address")
	}

	return auth.Nonce(snapshot, resolvedAddr), nil
}

func (s *QueryServer) Resolve(name string) (string, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	reg := s.CreateRegistry(snapshot)
	addr, err := reg.Resolve(name)
	if err != nil {
		return "", err
	}
	return addr.String(), nil
}

func decodeHexAddress(s string) ([]byte, error) {
	if !strings.HasPrefix(s, "0x") {
		return nil, errors.New("string has no hex prefix")
	}

	return hex.DecodeString(s[2:])
}

type WSEmptyResult struct{}

func writer(ctx rpctypes.WSRPCContext, subs *loomchain.SubscriptionSet) pubsub.SubscriberFunc {
	clientCtx := ctx
	log.Debug("Adding handler", "remote", clientCtx.GetRemoteAddr())
	return func(msg pubsub.Message) {
		log.Debug("Received published message", "msg", msg.Body(), "remote", clientCtx.GetRemoteAddr())
		defer func() {
			if r := recover(); r != nil {
				log.Error("Caught: WSEvent handler routine panic", "error", r)
				err := fmt.Errorf("Caught: WSEvent handler routine panic")
				clientCtx.WriteRPCResponse(rpctypes.RPCInternalError(rpctypes.JSONRPCStringID(""), err))
				go subs.Purge(clientCtx.GetRemoteAddr())
			}
		}()
		resp := rpctypes.RPCResponse{
			JSONRPC: "2.0",
			ID:      rpctypes.JSONRPCStringID("0"),
		}
		resp.Result = msg.Body()
		clientCtx.TryWriteRPCResponse(resp)
	}
}

func (s *QueryServer) Subscribe(wsCtx rpctypes.WSRPCContext, topics []string) (*WSEmptyResult, error) {
	if len(topics) == 0 {
		topics = append(topics, "contract")
	}
	caller := wsCtx.GetRemoteAddr()
	sub, existed := s.Subscriptions.For(caller)

	if !existed {
		sub.Do(writer(wsCtx, s.Subscriptions))
	}
	return &WSEmptyResult{}, s.Subscriptions.AddSubscription(caller, topics)
}

func (s *QueryServer) UnSubscribe(wsCtx rpctypes.WSRPCContext, topic string) (*WSEmptyResult, error) {
	return &WSEmptyResult{}, s.Subscriptions.Remove(wsCtx.GetRemoteAddr(), topic)
}

func ethWriter(ctx rpctypes.WSRPCContext, subs *subs.LegacyEthSubscriptionSet) pubsub.SubscriberFunc {
	clientCtx := ctx
	log.Debug("Adding handler", "remote", clientCtx.GetRemoteAddr())
	return func(msg pubsub.Message) {
		log.Debug("Received published message", "msg", msg.Body(), "remote", clientCtx.GetRemoteAddr())
		defer func() {
			if r := recover(); r != nil {
				log.Error("Caught: WSEvent handler routine panic", "error", r)
				err := fmt.Errorf("Caught: WSEvent handler routine panic")
				clientCtx.WriteRPCResponse(rpctypes.RPCInternalError(rpctypes.JSONRPCStringID(""), err))
				go subs.Purge(clientCtx.GetRemoteAddr())
			}
		}()
		ethMsg := types.EthMessage{}
		if err := proto.Unmarshal(msg.Body(), &ethMsg); err != nil {
			return
		}
		resp := rpctypes.RPCResponse{
			JSONRPC: "2.0",
			ID:      rpctypes.JSONRPCStringID(ethMsg.Id),
		}
		resp.Result = ethMsg.Body
		clientCtx.TryWriteRPCResponse(resp)
	}
}

func (s *QueryServer) EvmSubscribe(wsCtx rpctypes.WSRPCContext, method, filter string) (string, error) {
	caller := wsCtx.GetRemoteAddr()
	sub, id := s.EthLegacySubscriptions.For(caller)
	sub.Do(ethWriter(wsCtx, s.EthLegacySubscriptions))
	err := s.EthLegacySubscriptions.AddSubscription(id, method, filter)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *QueryServer) EvmUnSubscribe(id string) (bool, error) {
	return true, s.EthLegacySubscriptions.Remove(id)
}

func (s *QueryServer) EvmTxReceipt(txHash []byte) ([]byte, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	r := s.ReceiptHandlerProvider.Reader()
	txReceipt, err := r.GetReceipt(txHash)
	if errors.Cause(err) == common.ErrTxReceiptNotFound {
		return proto.Marshal(&types.EvmTxReceipt{})
	} else if err != nil {
		return nil, errors.Wrap(err, "get receipt")
	}

	if len(txReceipt.Logs) > 0 {
		height := int64(txReceipt.BlockNumber)
		var blockResult *ctypes.ResultBlock
		blockResult, err := s.BlockStore.GetBlockByHeight(&height)
		if err != nil {
			return nil, errors.Wrapf(err, "get block %d", height)
		}
		timestamp := blockResult.Block.Header.Time.Unix()

		for i := 0; i < len(txReceipt.Logs); i++ {
			txReceipt.Logs[i].BlockTime = timestamp
		}
	}
	return proto.Marshal(&txReceipt)
}

func (s *QueryServer) ContractEvents(
	fromBlock uint64, toBlock uint64, contractName string,
) (*types.ContractEventsResult, error) {
	if s.EventStore == nil {
		return nil, errors.New("event store is not available")
	}

	if fromBlock == 0 {
		return nil, fmt.Errorf("fromBlock not specified")
	}

	if toBlock == 0 {
		toBlock = fromBlock
	}

	if toBlock < fromBlock {
		return nil, fmt.Errorf("toBlock must be equal or greater than")
	}

	// default to max 20 blocks for now.
	maxRange := uint64(20)

	if toBlock-fromBlock > maxRange {
		return nil, fmt.Errorf("range exceeded, maximum range: %v", maxRange)
	}

	filter := store.EventFilter{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Contract:  contractName,
	}
	events, err := s.EventStore.FilterEvents(filter)
	if err != nil {
		return nil, err
	}

	return &types.ContractEventsResult{
		Events:    events,
		FromBlock: fromBlock,
		ToBlock:   toBlock,
	}, nil
}

func (s *QueryServer) GetContractRecord(contractAddrStr string) (*types.ContractRecordResponse, error) {
	contractAddr, err := loom.ParseAddress(contractAddrStr)
	if err != nil {
		return nil, err
	}
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	reg := s.CreateRegistry(snapshot)
	rec, err := reg.GetRecord(contractAddr)
	if err != nil {
		return nil, errors.Wrapf(err, "no contract exists at %s", contractAddr.String())
	}
	k := &types.ContractRecordResponse{
		ContractName:    rec.Name,
		ContractAddress: rec.Address,
		CreatorAddress:  rec.Owner,
	}
	return k, nil
}

// Takes a filter and returns a list of data relative to transactions that satisfies the filter
// Used to support eth_getLogs
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getlogs
func (s *QueryServer) GetEvmLogs(filter string) ([]byte, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	return query.DeprecatedQueryChain(
		filter, s.BlockStore, snapshot, s.ReceiptHandlerProvider.Reader(), s.EvmAuxStore,
	)
}

// Sets up new filter for polling
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newfilter
func (s *QueryServer) NewEvmFilter(filter string) (string, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	return s.EthPolls.LegacyAddLogPoll(filter, uint64(snapshot.Block().Height))
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newblockfilter
func (s *QueryServer) NewBlockEvmFilter() (string, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	return s.EthPolls.AddBlockPoll(uint64(snapshot.Block().Height)), nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newpendingtransactionfilter
func (s *QueryServer) NewPendingTransactionEvmFilter() (string, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	return s.EthPolls.AddTxPoll(uint64(snapshot.Block().Height)), nil
}

// Get the logs since last poll
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getfilterchanges
func (s *QueryServer) GetEvmFilterChanges(id string) ([]byte, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	// TODO: Reading from the TM block store could take a while, might be more efficient to release
	//       the current snapshot and get a new one after pulling out whatever we need from the TM
	//       block store.
	return s.EthPolls.LegacyPoll(snapshot, id, s.ReceiptHandlerProvider.Reader())
}

// Forget the filter.
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_uninstallfilter
func (s *QueryServer) UninstallEvmFilter(id string) (bool, error) {
	s.EthPolls.Remove(id)
	return true, nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_blocknumber
func (s *QueryServer) EthBlockNumber() (eth.Quantity, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	return eth.EncInt(snapshot.Block().Height), nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_blocknumber
func (s *QueryServer) GetBlockHeight() (int64, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	return snapshot.Block().Height - 1, nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getblockbynumber
func (s *QueryServer) GetEvmBlockByNumber(number string, full bool) ([]byte, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	r := s.ReceiptHandlerProvider.Reader()
	switch number {
	case "latest":
		return query.DeprecatedGetBlockByNumber(s.BlockStore, snapshot, snapshot.Block().Height, full, r, s.EvmAuxStore)
	case "pending":
		return query.DeprecatedGetBlockByNumber(s.BlockStore, snapshot, snapshot.Block().Height, full, r, s.EvmAuxStore)
	default:
		height, err := strconv.ParseInt(number, 10, 64)
		if err != nil {
			return nil, err
		}
		return query.DeprecatedGetBlockByNumber(s.BlockStore, snapshot, int64(height), full, r, s.EvmAuxStore)
	}
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getblockbyhash
func (s *QueryServer) GetEvmBlockByHash(hash []byte, full bool) ([]byte, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	return query.DeprecatedGetBlockByHash(
		s.BlockStore, snapshot, hash, full, s.ReceiptHandlerProvider.Reader(), s.EvmAuxStore,
	)
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_gettransactionbyhash
func (s *QueryServer) GetEvmTransactionByHash(txHash []byte) (resp []byte, err error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	return query.DeprecatedGetTxByHash(snapshot, txHash, s.ReceiptHandlerProvider.Reader())
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getblockbynumber
func (s *QueryServer) EthGetBlockByNumber(block eth.BlockHeight, full bool) (resp eth.JsonBlockObject, err error) {
	if block == "0x0" {
		return eth.GetBlockZero(), nil
	}

	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	height, err := eth.DecBlockHeight(snapshot.Block().Height, block)
	if err != nil {
		return resp, err
	}

	// TODO: Reading from the TM block store could take a while, might be more efficient to release
	//       the current snapshot and get a new one after pulling out whatever we need from the TM
	//       block store.
	blockResult, err := query.GetBlockByNumber(s.BlockStore, snapshot, int64(height), full, s.EvmAuxStore)
	if err != nil {
		return resp, err
	}

	if block == "0x1" && blockResult.ParentHash == "0x0" {
		blockResult.ParentHash = "0x0000000000000000000000000000000000000000000000000000000000000001"
	}

	return blockResult, err
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_gettransactionreceipt
func (s *QueryServer) EthGetTransactionReceipt(hash eth.Data) (*eth.JsonTxReceipt, error) {
	txHash, err := eth.DecDataToBytes(hash)
	if err != nil {
		return nil, err
	}

	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	r := s.ReceiptHandlerProvider.Reader()
	txReceipt, err := r.GetReceipt(txHash)
	if err != nil {
		resp, err := getReceiptByTendermintHash(snapshot, s.BlockStore, r, txHash)
		if err != nil {
			if strings.Contains(errors.Cause(err).Error(), "not found") {
				// return nil response if cannot find hash
				return nil, nil
			}
			return nil, err
		}
		return resp, nil
	}
	snapshot.Release()

	height := int64(txReceipt.BlockNumber)
	blockResult, err := s.BlockStore.GetBlockByHeight(&height)
	if err != nil {
		return nil, err
	}
	if int32(len(blockResult.Block.Data.Txs)) <= txReceipt.TransactionIndex {
		return nil, errors.Errorf(
			"Transaction index %v out of bounds for transactions in block %v",
			txReceipt.TransactionIndex, len(blockResult.Block.Data.Txs),
		)
	}
	txResults, err := s.BlockStore.GetTxResult(blockResult.Block.Data.Txs[txReceipt.TransactionIndex].Hash())
	if err != nil {
		if strings.Contains(errors.Cause(err).Error(), "not found") {
			// return nil response if cannot find hash
			return nil, nil
		}
		return nil, err
	}
	return completeReceipt(txResults, blockResult, &txReceipt), nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getblocktransactioncountbyhash
func (s *QueryServer) EthGetBlockTransactionCountByHash(hash eth.Data) (txCount eth.Quantity, err error) {
	blockHash, err := eth.DecDataToBytes(hash)
	if err != nil {
		return txCount, err
	}

	height, err := s.getBlockHeightFromHash(blockHash)
	if err != nil {
		return txCount, err
	}

	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()
	count, err := query.GetNumTxBlock(s.BlockStore, snapshot, int64(height))
	if err != nil {
		return txCount, err
	}
	return eth.EncUint(count), nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getblocktransactioncountbynumber
func (s *QueryServer) EthGetBlockTransactionCountByNumber(block eth.BlockHeight) (txCount eth.Quantity, err error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	height, err := eth.DecBlockHeight(snapshot.Block().Height, block)
	if err != nil {
		return txCount, err
	}
	count, err := query.GetNumTxBlock(s.BlockStore, snapshot, int64(height))
	if err != nil {
		return txCount, err
	}
	return eth.EncUint(count), nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getblockbyhash
func (s *QueryServer) EthGetBlockByHash(hash eth.Data, full bool) (resp eth.JsonBlockObject, err error) {
	blockHash, err := eth.DecDataToBytes(hash)
	if err != nil {
		return resp, err
	}

	height, err := s.getBlockHeightFromHash(blockHash)
	if err != nil {
		return resp, err
	}

	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	return query.GetBlockByNumber(s.BlockStore, snapshot, int64(height), full, s.EvmAuxStore)
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_gettransactionbyhash
func (s *QueryServer) EthGetTransactionByHash(hash eth.Data) (resp eth.JsonTxObject, err error) {
	txHash, err := eth.DecDataToBytes(hash)
	if err != nil {
		return resp, err
	}

	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	r := s.ReceiptHandlerProvider.Reader()
	txObj, err := query.GetTxByHash(snapshot, s.BlockStore, txHash, r)
	if err != nil {
		if errors.Cause(err) != common.ErrTxReceiptNotFound {
			return resp, err
		}

		txObj, err = getTxByTendermintHash(s.BlockStore, txHash)
		if err != nil {
			return resp, errors.Wrapf(err, "failed to find tx with hash %v", txHash)
		}
	}
	return txObj, nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_gettransactionbyblockHashAndIndex
func (s *QueryServer) EthGetTransactionByBlockHashAndIndex(
	hash eth.Data, index eth.Quantity,
) (txObj eth.JsonTxObject, err error) {
	blockHash, err := eth.DecDataToBytes(hash)
	if err != nil {
		return txObj, err
	}

	height, err := s.getBlockHeightFromHash(blockHash)
	if err != nil {
		return txObj, err
	}

	txIndex, err := eth.DecQuantityToUint(index)
	if err != nil {
		return txObj, err
	}

	return query.GetTxByBlockAndIndex(s.BlockStore, uint64(height), txIndex)
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_gettransactionbyblocknumberandindex
func (s *QueryServer) EthGetTransactionByBlockNumberAndIndex(
	block eth.BlockHeight, index eth.Quantity,
) (txObj eth.JsonTxObject, err error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	height, err := eth.DecBlockHeight(snapshot.Block().Height, block)
	if err != nil {
		return txObj, err
	}
	snapshot.Release() // don't need to hold on to it any longer

	txIndex, err := eth.DecQuantityToUint(index)
	if err != nil {
		return txObj, err
	}
	return query.GetTxByBlockAndIndex(s.BlockStore, height, txIndex)
}

/// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getlogs
func (s *QueryServer) EthGetLogs(filter eth.JsonFilter) (resp []eth.JsonLog, err error) {
	ethFilter, err := eth.DecLogFilter(filter)
	if err != nil {
		return resp, err
	}

	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	// TODO: Reading from the TM block store could take a while, might be more efficient to release
	//       the current snapshot and get a new one after pulling out whatever we need from the TM
	//       block store.
	logs, err := query.QueryChain(
		s.BlockStore, snapshot, ethFilter, s.ReceiptHandlerProvider.Reader(), s.EvmAuxStore,
	)
	if err != nil {
		return resp, err
	}
	return eth.EncLogs(logs), err
}

// todo add EthNewBlockFilter EthNewPendingTransactionFilter EthUninstallFilter EthGetFilterChanges and EthGetFilterLogs
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newblockfilter
func (s *QueryServer) EthNewBlockFilter() (eth.Quantity, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()
	return eth.Quantity(s.EthPolls.AddBlockPoll(uint64(snapshot.Block().Height))), nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newpendingtransactionfilter
func (s *QueryServer) EthNewPendingTransactionFilter() (eth.Quantity, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()
	return eth.Quantity(s.EthPolls.AddTxPoll(uint64(snapshot.Block().Height))), nil
}

// Forget the filter.
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_uninstallfilter
func (s *QueryServer) EthUninstallFilter(id eth.Quantity) (bool, error) {
	s.EthPolls.Remove(string(id))
	return true, nil
}

// Get the logs since last poll
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getfilterchanges
func (s *QueryServer) EthGetFilterChanges(id eth.Quantity) (interface{}, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	return s.EthPolls.Poll(snapshot, string(id), s.ReceiptHandlerProvider.Reader())
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getfilterlogs
func (s *QueryServer) EthGetFilterLogs(id eth.Quantity) (interface{}, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()
	return s.EthPolls.AllLogs(snapshot, string(id), s.ReceiptHandlerProvider.Reader())
}

// Sets up new filter for polling
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newfilter
func (s *QueryServer) EthNewFilter(filter eth.JsonFilter) (eth.Quantity, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()
	ethFilter, err := eth.DecLogFilter(filter)
	if err != nil {
		return "", errors.Wrap(err, "could decode log filter")
	}
	id, err := s.EthPolls.AddLogPoll(ethFilter, uint64(snapshot.Block().Height))
	return eth.Quantity(id), err
}

func (s *QueryServer) EthSubscribe(conn *websocket.Conn, method eth.Data, filter eth.JsonFilter) (eth.Data, error) {
	f, err := eth.DecLogFilter(filter)
	if err != nil {
		return "", errors.Wrapf(err, "decode filter")
	}
	id, err := s.EthSubscriptions.AddSubscription(string(method), f, conn)
	if err != nil {
		return "", errors.Wrapf(err, "add subscription")
	}
	return eth.Data(id), nil
}

func (s *QueryServer) EthUnsubscribe(id eth.Quantity) (unsubscribed bool, err error) {
	s.EthSubscriptions.Remove(string(id))
	return true, nil
}

func (s *QueryServer) EthGetTransactionCount(local eth.Data, block eth.BlockHeight) (eth.Quantity, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	height, err := eth.DecBlockHeight(snapshot.Block().Height, block)
	if err != nil {
		return eth.Quantity("0x0"), err
	}

	if height != uint64(snapshot.Block().Height) {
		return eth.Quantity("0x0"), errors.New("transaction count only available for the latest block")
	}
	address, err := eth.DecDataToAddress(s.ChainID, local)
	if err != nil {
		return eth.Quantity("0x0"), err
	}
	nonce, err := s.Nonce("", address.String())
	if err != nil {
		return eth.Quantity("0x0"), errors.Wrap(err, "requesting transaction count")
	}

	return eth.EncUint(nonce), nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getbalance
// uses ethcoin contract to return the balance corresponding to the address
func (s *QueryServer) EthGetBalance(address eth.Data, block eth.BlockHeight) (eth.Quantity, error) {
	owner, err := eth.DecDataToAddress(s.ChainID, address)
	if err != nil {
		return "", errors.Wrapf(err, "decoding input address parameter %v", address)
	}

	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()
	height, err := eth.DecBlockHeight(snapshot.Block().Height, block)
	if err != nil {
		return "", errors.Wrapf(err, "invalid block height %s", block)
	}
	if int64(height) != snapshot.Block().Height {
		return "", errors.Errorf("height %s not latest", block)
	}

	ctx, err := s.createStaticContractCtx(snapshot, "ethcoin")
	if err != nil {
		if errors.Cause(err) == registry.ErrNotFound {
			return eth.Quantity("0x0"), nil
		}
		return eth.Quantity("0x0"), err
	}
	amount, err := ethcoin.BalanceOf(ctx, owner)
	if err != nil {
		return eth.Quantity("0x0"), err
	}
	if amount == nil {
		return eth.Quantity("0x0"), errors.Errorf("No amount returned for address %s", address)
	}

	return eth.EncBigInt(*amount.Int), nil
}

func (s *QueryServer) EthEstimateGas(query eth.JsonTxCallObject) (eth.Quantity, error) {
	return eth.Quantity("0x0"), nil
}

func (s *QueryServer) EthGasPrice() (eth.Quantity, error) {
	return eth.Quantity("0x0"), nil
}

func (s *QueryServer) EthNetVersion() (string, error) {
	hash := sha3.SoliditySHA3(sha3.String(s.ChainID))
	versionBigInt := new(big.Int)
	versionBigInt.SetString(hex.EncodeToString(hash)[0:13], 16)
	return versionBigInt.String(), nil
}

func (s *QueryServer) EthAccounts() ([]eth.Data, error) {
	return []eth.Data{}, nil
}

func (s *QueryServer) GetValidators() (*blockatlas.JsonGetValidators, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	ctx, err := s.createDPOS3Ctx(snapshot)
	if err != nil {
		return nil, err
	}

	validators, err := dposv3.ValidatorList(ctx)
	if err != nil {
		return nil, err
	}

	candidates, err := dposv3.LoadCandidateList(ctx)
	if err != nil {
		return nil, err
	}

	getValidatorResponse := make([]blockatlas.Validator, 0)
	chainID := ctx.Block().ChainID

	validatorList := make(map[string]loom.Address, len(validators))
	for _, validator := range validators {
		address := loom.LocalAddressFromPublicKey(validator.PubKey)
		validatorList[address.String()] = loom.Address{ChainID: chainID, Local: address}
	}

	for _, candidate := range candidates {
		candidateAddr := loom.UnmarshalAddressPB(candidate.Address)
		if validatorList[candidate.Address.Local.String()].Compare(candidateAddr) != 0 {
			continue
		}
		statistic, err := getStatistic(ctx, candidateAddr)
		if err != nil && err != ErrNotFound {
			return nil, err
		}

		getValidatorResponse = append(getValidatorResponse, blockatlas.Validator{
			Address:         prefixLoom(statistic.GetAddress().Local.String()),
			Jailed:          statistic.Jailed,
			Name:            candidate.Name,
			Description:     candidate.Description,
			DelegationTotal: statistic.GetDelegationTotal().Value.String(),
			Website:         candidate.Website,
			Fee:             strconv.FormatUint(candidate.Fee, 10),
		})
	}

	return &blockatlas.JsonGetValidators{
		Validators: getValidatorResponse,
	}, nil
}

func getStatistic(ctx contractpb.StaticContext, address loom.Address) (*dtypes.ValidatorStatistic, error) {
	addressBytes, err := address.Local.Marshal()
	if err != nil {
		return nil, err
	}
	return getStatisticByAddressBytes(ctx, addressBytes)
}

func getStatisticByAddressBytes(ctx contractpb.StaticContext, addressBytes []byte) (*dtypes.ValidatorStatistic, error) {
	var statistic dtypes.ValidatorStatistic
	err := ctx.Get(append([]byte("statistic"), addressBytes...), &statistic)
	if err != nil {
		return nil, err
	}
	return &statistic, nil
}

func (s *QueryServer) createDPOS3Ctx(state loomchain.State) (contractpb.StaticContext, error) {
	return s.createStaticContractCtx(state, "dposV3")
}

func (s *QueryServer) getBlockHeightFromHash(hash []byte) (uint64, error) {
	if nil != s.BlockIndexStore {
		return s.BlockIndexStore.GetBlockHeightByHash(hash)
	} else {
		snapshot := s.StateProvider.ReadOnlyState()
		defer snapshot.Release()
		height, err := query.GetBlockHeightFromHash(s.BlockStore, snapshot, hash)
		return uint64(height), err
	}
}

func getReceiptByTendermintHash(state loomchain.State, blockStore store.BlockStore, rh loomchain.ReadReceiptHandler, hash []byte) (*eth.JsonTxReceipt, error) {
	txResults, err := blockStore.GetTxResult(hash)
	if err != nil {
		return nil, err
	}
	blockResult, err := blockStore.GetBlockByHeight(&txResults.Height)
	if err != nil {
		return nil, err
	}
	txObj, contractAddr, err := query.GetTxObjectFromBlockResult(blockResult, txResults.TxResult.Data, int64(txResults.Index))
	if err != nil {
		return nil, err
	}
	txHash, err := eth.DecDataToBytes(txObj.Hash)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid loom transaction hash %x", txObj.Hash)
	}
	txReceipt, err := rh.GetReceipt(txHash)
	if err != nil {
		jsonReceipt := eth.TxObjToReceipt(txObj, contractAddr)
		if txResults.TxResult.Code == abci.CodeTypeOK {
			jsonReceipt.Status = eth.EncInt(int64(StatusTxSuccess))
		} else {
			jsonReceipt.Status = eth.EncInt(int64(StatusTxFail))
		}
		if txResults.TxResult.Info == utils.CallEVM || txResults.TxResult.Info == utils.CallPlugin {
			if jsonReceipt.To == nil || len(*jsonReceipt.To) == 0 {
				jsonReceipt.To = jsonReceipt.ContractAddress
			}
			jsonReceipt.ContractAddress = nil
		}

		return &jsonReceipt, nil
	}
	return completeReceipt(txResults, blockResult, &txReceipt), nil
}

func completeReceipt(txResults *ctypes.ResultTx, blockResult *ctypes.ResultBlock, txReceipt *types.EvmTxReceipt) *eth.JsonTxReceipt {
	if len(txReceipt.Logs) > 0 {
		timestamp := blockResult.Block.Header.Time.Unix()
		for i := 0; i < len(txReceipt.Logs); i++ {
			txReceipt.Logs[i].BlockTime = timestamp
		}
	}
	if txResults.TxResult.Code == abci.CodeTypeOK {
		txReceipt.Status = StatusTxSuccess
	} else {
		txReceipt.Status = StatusTxFail
	}
	jsonReceipt := eth.EncTxReceipt(*txReceipt)
	if txResults.TxResult.Info == utils.CallEVM && (jsonReceipt.To == nil || len(*jsonReceipt.To) == 0) {
		jsonReceipt.To = jsonReceipt.ContractAddress
		jsonReceipt.ContractAddress = nil
	}
	return &jsonReceipt
}

func getTxByTendermintHash(blockStore store.BlockStore, hash []byte) (eth.JsonTxObject, error) {
	txResults, err := blockStore.GetTxResult(hash)
	if err != nil {
		return eth.JsonTxObject{}, err
	}
	blockResult, err := blockStore.GetBlockByHeight(&txResults.Height)
	if err != nil {
		return eth.JsonTxObject{}, err
	}
	txObj, _, err := query.GetTxObjectFromBlockResult(blockResult, txResults.TxResult.Data, int64(txResults.Index))
	return txObj, err
}

func prefixLoom(address string) string {
	if address[:2] != "0x" {
		return address
	}
	return "loom" + address[2:]
}
