package rpc

import (
	"encoding/hex"
	"fmt"
	"github.com/gorilla/websocket"
	"strconv"
	"strings"

	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/pkg/errors"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/eth/polls"
	"github.com/loomnetwork/loomchain/eth/query"
	"github.com/loomnetwork/loomchain/eth/subs"
	levm "github.com/loomnetwork/loomchain/evm"
	"github.com/loomnetwork/loomchain/log"
	lcp "github.com/loomnetwork/loomchain/plugin"
	registry "github.com/loomnetwork/loomchain/registry/factory"
	lvm "github.com/loomnetwork/loomchain/vm"
	"github.com/phonkee/go-pubsub"
	"github.com/tendermint/tendermint/rpc/lib/types"
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
	ChainID                     string
	Loader                      lcp.Loader
	Subscriptions               *loomchain.SubscriptionSet
	EthSubscriptions            *subs.EthSubscriptionSet
	EthDepreciatedSubscriptions *subs.EthDepreciatedSubscriptionSet
	EthPolls                    polls.EthSubscriptions
	CreateRegistry              registry.RegistryFactoryFunc
	// If this is nil the EVM won't have access to any account balances.
	NewABMFactory               lcp.NewAccountBalanceManagerFactoryFunc
	ReceiptHandler              loomchain.ReadReceiptHandler
	RPCListenAddress            string
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
		return s.QueryPlugin(callerAddr, contractAddr, query)
	} else {
		return s.QueryEvm(callerAddr, contractAddr, query)
	}
}

func (s *QueryServer) QueryPlugin(caller, contract loom.Address, query []byte) ([]byte, error) {
	vm := lcp.NewPluginVM(
		s.Loader,
		s.StateProvider.ReadOnlyState(),
		s.CreateRegistry(s.StateProvider.ReadOnlyState()),
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

	respBytes, err := vm.StaticCall(caller, contract, reqBytes)
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

func (s *QueryServer) QueryEvm(caller, contract loom.Address, query []byte) ([]byte, error) {
	var createABM levm.AccountBalanceManagerFactoryFunc
	var err error
	if s.NewABMFactory != nil {
		pvm := lcp.NewPluginVM(
			s.Loader,
			s.StateProvider.ReadOnlyState(),
			s.CreateRegistry(s.StateProvider.ReadOnlyState()),
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
	vm := levm.NewLoomVm(s.StateProvider.ReadOnlyState(), nil, nil, createABM, false)
	return vm.StaticCall(caller, contract, query)
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_call
func (s QueryServer) EthCall(query eth.JsonTxCallObject, block eth.BlockHeight) (resp eth.Data, err error) {
	var caller loom.Address
	if len(query.From) > 0 {
		caller, err = eth.DecDataToAddress(s.ChainID, query.From)
		if err != nil {
			return resp, err
		}
	}
	contract, err := eth.DecDataToAddress(s.ChainID, query.To)
	if err != nil {
		return resp, err
	}
	data, err := eth.DecDataToBytes(query.Data)
	if err != nil {
		return resp, err
	}
	bytes, err := s.QueryEvm(caller, contract, data)
	return eth.EncBytes(bytes), err
}

// GetCode returns the runtime byte-code of a contract running on a DAppChain's EVM.
// Gives an error for non-EVM contracts.
// contract - address of the contract in the form of a string. (Use loom.Address.String() to convert)
// return []byte - runtime bytecode of the contract.
func (s *QueryServer) GetEvmCode(contract string) ([]byte, error) {
	contractAddr, err := loom.ParseAddress(contract)
	if err != nil {
		return nil, err
	}
	vm := levm.NewLoomVm(s.StateProvider.ReadOnlyState(), nil, nil, nil, false)
	return vm.GetCode(contractAddr)
}

func (s *QueryServer) EthGetCode(address eth.Data, block eth.BlockHeight) (eth.Data, error) {
	addr, err := eth.DecDataToAddress(s.ChainID, address)
	if err != nil {
		return "", errors.Wrapf(err, "decoding input address parameter %v", address)
	}
	evm := levm.NewLoomVm(s.StateProvider.ReadOnlyState(), nil, nil, nil, false)
	code, err := evm.GetCode(addr)
	if err != nil {
		return "", err
	}
	return eth.EncBytes(code), nil
}

// Nonce returns of nonce from the application states
func (s *QueryServer) Nonce(key string) (uint64, error) {
	k, err := hex.DecodeString(key)
	if err != nil {
		return 0, err
	}
	addr := loom.Address{
		ChainID: s.ChainID,
		Local:   loom.LocalAddressFromPublicKey(k),
	}
	return auth.Nonce(s.StateProvider.ReadOnlyState(), addr), nil
}

func (s *QueryServer) Resolve(name string) (string, error) {
	reg := s.CreateRegistry(s.StateProvider.ReadOnlyState())
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
				clientCtx.WriteRPCResponse(rpctypes.RPCInternalError("Internal server error", err))
				go subs.Purge(clientCtx.GetRemoteAddr())
			}
		}()
		resp := rpctypes.RPCResponse{
			JSONRPC: "2.0",
			ID:      "0",
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
	s.Subscriptions.AddSubscription(caller, topics)
	return &WSEmptyResult{}, nil
}

func (s *QueryServer) UnSubscribe(wsCtx rpctypes.WSRPCContext, topic string) (*WSEmptyResult, error) {
	s.Subscriptions.Remove(wsCtx.GetRemoteAddr(), topic)
	return &WSEmptyResult{}, nil
}

func ethWriter(ctx rpctypes.WSRPCContext, subs *subs.EthDepreciatedSubscriptionSet) pubsub.SubscriberFunc {
	clientCtx := ctx
	log.Debug("Adding handler", "remote", clientCtx.GetRemoteAddr())
	return func(msg pubsub.Message) {
		log.Debug("Received published message", "msg", msg.Body(), "remote", clientCtx.GetRemoteAddr())
		defer func() {
			if r := recover(); r != nil {
				log.Error("Caught: WSEvent handler routine panic", "error", r)
				err := fmt.Errorf("Caught: WSEvent handler routine panic")
				clientCtx.WriteRPCResponse(rpctypes.RPCInternalError("Internal server error", err))
				go subs.Purge(clientCtx.GetRemoteAddr())
			}
		}()
		ethMsg := types.EthMessage{}
		if err := proto.Unmarshal(msg.Body(), &ethMsg); err != nil {
			return
		}
		resp := rpctypes.RPCResponse{
			JSONRPC: "2.0",
			ID:      ethMsg.Id,
		}
		resp.Result = ethMsg.Body
		clientCtx.TryWriteRPCResponse(resp)
	}
}

func (s *QueryServer) EvmSubscribe(wsCtx rpctypes.WSRPCContext, method, filter string) (string, error) {
	caller := wsCtx.GetRemoteAddr()
	sub, id := s.EthDepreciatedSubscriptions.For(caller)
	sub.Do(ethWriter(wsCtx, s.EthDepreciatedSubscriptions))
	err := s.EthDepreciatedSubscriptions.AddSubscription(id, method, filter)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *QueryServer) EvmUnSubscribe(id string) (bool, error) {
	s.EthDepreciatedSubscriptions.Remove(id)
	return true, nil
}

func (s *QueryServer) EvmTxReceipt(txHash []byte) ([]byte, error) {
	txReciept, err := s.ReceiptHandler.GetReceipt(s.StateProvider.ReadOnlyState(), txHash)
	if err != nil {
		return nil, errors.Wrap(err, "get receipt")
	}
	return proto.Marshal(&txReciept)
}

// Takes a filter and returns a list of data relative to transactions that satisfies the filter
// Used to support eth_getLogs
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getlogs
func (s *QueryServer) GetEvmLogs(filter string) ([]byte, error) {
	state := s.StateProvider.ReadOnlyState()
	return query.DeprecatedQueryChain(filter, state, s.ReceiptHandler)
}

// Sets up new filter for polling
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newfilter
func (s *QueryServer) NewEvmFilter(filter string) (string, error) {
	state := s.StateProvider.ReadOnlyState()
	return s.EthPolls.DepreciatedAddLogPoll(filter, uint64(state.Block().Height))
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newblockfilter
func (s *QueryServer) NewBlockEvmFilter() (string, error) {
	state := s.StateProvider.ReadOnlyState()
	return s.EthPolls.AddBlockPoll(uint64(state.Block().Height)), nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newpendingtransactionfilter
func (s *QueryServer) NewPendingTransactionEvmFilter() (string, error) {
	state := s.StateProvider.ReadOnlyState()
	return s.EthPolls.AddTxPoll(uint64(state.Block().Height)), nil
}

// Get the logs since last poll
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getfilterchanges
func (s *QueryServer) GetEvmFilterChanges(id string) ([]byte, error) {
	state := s.StateProvider.ReadOnlyState()
	return s.EthPolls.DepreciatedPoll(state, id, s.ReceiptHandler)
}

// Forget the filter.
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_uninstallfilter
func (s *QueryServer) UninstallEvmFilter(id string) (bool, error) {
	s.EthPolls.Remove(id)
	return true, nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_blocknumber
func (s *QueryServer) EthBlockNumber() (eth.Quantity, error) {
	state := s.StateProvider.ReadOnlyState()
	return eth.EncInt(state.Block().Height), nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_blocknumber
func (s *QueryServer) GetBlockHeight() (int64, error) {
	state := s.StateProvider.ReadOnlyState()
	return state.Block().Height - 1, nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getblockbynumber
func (s *QueryServer) GetEvmBlockByNumber(number string, full bool) ([]byte, error) {
	state := s.StateProvider.ReadOnlyState()
	switch number {
	case "latest":
		return query.DeprecatedGetBlockByNumber(state, state.Block().Height-1, full, s.ReceiptHandler)
	case "pending":
		return query.DeprecatedGetBlockByNumber(state, state.Block().Height, full, s.ReceiptHandler)
	default:
		height, err := strconv.ParseInt(number, 10, 64)
		if err != nil {
			return nil, err
		}
		return query.DeprecatedGetBlockByNumber(state, int64(height), full, s.ReceiptHandler)
	}
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getblockbyhash
func (s *QueryServer) GetEvmBlockByHash(hash []byte, full bool) ([]byte, error) {
	state := s.StateProvider.ReadOnlyState()
	return query.DeprecatedGetBlockByHash(state, hash, full, s.ReceiptHandler)
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_gettransactionbyhash
func (s QueryServer) GetEvmTransactionByHash(txHash []byte) (resp []byte, err error) {
	state := s.StateProvider.ReadOnlyState()
	return query.DeprecatedGetTxByHash(state, txHash, s.ReceiptHandler)
}

func (s *QueryServer) EthGetBlockByNumber(block eth.BlockHeight, full bool) (resp eth.JsonBlockObject, err error) {
	state := s.StateProvider.ReadOnlyState()
	height, err := eth.DecBlockHeight(state.Block().Height, block)
	if err != nil {
		return resp, err
	}
	return query.GetBlockByNumber(state, int64(height), full, s.ReceiptHandler)
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_gettransactionreceipt
func (s *QueryServer) EthGetTransactionReceipt(hash eth.Data) (resp eth.JsonTxReceipt, err error) {
	state := s.StateProvider.ReadOnlyState()
	txHash, err := eth.DecDataToBytes(hash)
	if err != nil {
		return resp, err
	}
	txReceipt, err := s.ReceiptHandler.GetReceipt(state, txHash)
	if err != nil {
		return resp, err
	}
	return eth.EncTxReceipt(txReceipt), nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getblocktransactioncountbyhash
func (s *QueryServer) EthGetBlockTransactionCountByHash(hash eth.Data) (txCount eth.Quantity, err error) {
	state := s.StateProvider.ReadOnlyState()
	blockHash, err := eth.DecDataToBytes(hash)
	if err != nil {
		return txCount, err
	}
	height, err := query.GetBlockHeightFromHash(state, blockHash)
	if err != nil {
		return txCount, err
	}
	count, err := query.GetNumEvmTxBlock(state, height)
	if err != nil {
		return txCount, err
	}
	return eth.EncUint(count), nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getblocktransactioncountbynumber
func (s *QueryServer) EthGetBlockTransactionCountByNumber(block eth.BlockHeight) (txCount eth.Quantity, err error) {
	state := s.StateProvider.ReadOnlyState()
	height, err := eth.DecBlockHeight(state.Block().Height, block)
	if err != nil {
		return txCount, err
	}
	count, err := query.GetNumEvmTxBlock(state, int64(height))
	if err != nil {
		return txCount, err
	}
	return eth.EncUint(count), nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getblockbyhash
func (s *QueryServer) EthGetBlockByHash(hash eth.Data, full bool) (resp eth.JsonBlockObject, err error) {
	state := s.StateProvider.ReadOnlyState()
	blockHash, err := eth.DecDataToBytes(hash)
	if err != nil {
		return resp, err
	}
	height, err := query.GetBlockHeightFromHash(state, blockHash)
	if err != nil {
		return resp, err
	}
	return query.GetBlockByNumber(state, height, full, s.ReceiptHandler)
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_gettransactionbyhash
func (s QueryServer) EthGetTransactionByHash(hash eth.Data) (resp eth.JsonTxObject, err error) {
	state := s.StateProvider.ReadOnlyState()
	txHash, err := eth.DecDataToBytes(hash)
	if err != nil {
		return resp, err
	}
	return query.GetTxByHash(state, txHash, s.ReceiptHandler)
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_gettransactionbyblockHashAndIndex
func (s QueryServer) EthGetTransactionByBlockHashAndIndex(hash eth.Data, index eth.Quantity) (txObj eth.JsonTxObject, err error) {
	state := s.StateProvider.ReadOnlyState()
	blockHash, err := eth.DecDataToBytes(hash)
	if err != nil {
		return txObj, err
	}
	height, err := query.GetBlockHeightFromHash(state, blockHash)
	if err != nil {
		return txObj, err
	}
	txIndex, err := eth.DecQuantityToUint(index)
	if err != nil {
		return txObj, err
	}
	return query.GetTxByBlockAndIndex(state, uint64(height), txIndex, s.ReceiptHandler)
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_gettransactionbyblocknumberandindex
func (s QueryServer) EthGetTransactionByBlockNumberAndIndex(block eth.BlockHeight, index eth.Quantity) (txObj eth.JsonTxObject, err error) {
	state := s.StateProvider.ReadOnlyState()
	height, err := eth.DecBlockHeight(state.Block().Height, block)
	if err != nil {
		return txObj, err
	}
	txIndex, err := eth.DecQuantityToUint(index)
	if err != nil {
		return txObj, err
	}
	return query.GetTxByBlockAndIndex(state, height, txIndex, s.ReceiptHandler)
}

/// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getlogs
func (s QueryServer) EthGetLogs(filter eth.JsonFilter) (resp []eth.JsonLog, err error) {
	state := s.StateProvider.ReadOnlyState()
	ethFilter, err := eth.DecLogFilter(filter)
	if err != nil {
		return resp, err
	}
	logs, err := query.QueryChain(state, ethFilter, s.ReceiptHandler)
	if err != nil {
		return resp, err
	}
	return eth.EncLogs(logs), err
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newblockfilter
func (s QueryServer) EthNewBlockFilter() (eth.Quantity, error) {
	state := s.StateProvider.ReadOnlyState()
	return eth.Quantity(s.EthPolls.AddBlockPoll(uint64(state.Block().Height))), nil
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newpendingtransactionfilter
func (s QueryServer) EthNewPendingTransactionFilter() (eth.Quantity, error) {
	state := s.StateProvider.ReadOnlyState()
	return eth.Quantity(s.EthPolls.AddTxPoll(uint64(state.Block().Height))), nil
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
	state := s.StateProvider.ReadOnlyState()
	return s.EthPolls.Poll(state, string(id), s.ReceiptHandler)
}

// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_getfilterlogs
func (s *QueryServer) EthGetFilterLogs(id eth.Quantity) (interface{}, error) {
	state := s.StateProvider.ReadOnlyState()
	if filter, err := s.EthSubscriptions.GetFilter(string(id)); filter != nil || err != nil {
		logs, err := query.QueryChain(state, *filter, s.ReceiptHandler)
		if err != nil {
			return nil, err
		}
		return eth.EncLogs(logs), err
	} else {
		return s.EthPolls.AllLogs(state, string(id), s.ReceiptHandler)
	}
}

// Sets up new filter for polling
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_newfilter
func (s *QueryServer) EthNewFilter(filter eth.JsonFilter) (eth.Quantity, error) {
	state := s.StateProvider.ReadOnlyState()
	ethFilter, err := eth.DecLogFilter(filter)
	if err != nil {
		return "", errors.Wrap(err, "could decode log filter")
	}
	id, err := s.EthPolls.AddLogPoll(ethFilter, uint64(state.Block().Height))
	return eth.Quantity(id), err
}

func (s *QueryServer) EthSubscribe(conn websocket.Conn, method eth.Data, filter eth.JsonFilter) (eth.Data, error) {
	f, err := eth.DecLogFilter(filter)
	if err != nil {
		return "", errors.Wrapf(err, "decode filter")
	}
	id, err := s.EthSubscriptions.AddSubscription(string(method), f, conn)
	return eth.EncBytes([]byte(id)), nil
}

func (s *QueryServer) EthUnsubscribe(id eth.Quantity) (unsubscribed bool, err error) {
	s.EthSubscriptions.Remove(string(id))
	return true, nil
}
