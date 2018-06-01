package rpc

import (
	"encoding/hex"
	"errors"
	"strings"

	"encoding/json"

	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/log"
	lcp "github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/loomnetwork/loomchain/store"
	lvm "github.com/loomnetwork/loomchain/vm"
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
	ChainID       string
	Loader        lcp.Loader
	Subscriptions *loomchain.SubscriptionSet
}

var _ QueryService = &QueryServer{}

// Query returns data of given contract from the application states
// The contract parameter should be a hex-encoded local address prefixed by 0x
func (s *QueryServer) Query(contract string, query []byte, vmType vm.VMType) ([]byte, error) {
	if vmType == lvm.VMType_PLUGIN {
		return s.QueryPlugin(contract, query)
	} else {
		return s.QueryEvm(contract, query)
	}
}

func (s *QueryServer) QueryPlugin(contract string, query []byte) ([]byte, error) {
	vm := &lcp.PluginVM{
		Loader: s.Loader,
		State:  s.StateProvider.ReadOnlyState(),
	}
	req := &plugin.Request{
		ContentType: plugin.EncodingType_PROTOBUF3,
		Accept:      plugin.EncodingType_PROTOBUF3,
		Body:        query,
	}
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}
	var caller loom.Address
	localContractAddr, err := decodeHexAddress(contract)
	if err != nil {
		return nil, err
	}
	contractAddr := loom.Address{
		ChainID: s.ChainID,
		Local:   localContractAddr,
	}
	respBytes, err := vm.StaticCall(caller, contractAddr, reqBytes)
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

func (s *QueryServer) QueryEvm(contract string, query []byte) ([]byte, error) {

	vm := lvm.NewLoomVm(s.StateProvider.ReadOnlyState(), nil)
	reqBytes := query

	var caller loom.Address
	localContractAddr, err := decodeHexAddress(contract)
	if err != nil {
		return nil, err
	}
	contractAddr := loom.Address{
		ChainID: s.ChainID,
		Local:   localContractAddr,
	}
	return vm.StaticCall(caller, contractAddr, reqBytes)
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
	registry := &registry.StateRegistry{
		State: s.StateProvider.ReadOnlyState(),
	}

	addr, err := registry.Resolve(name)
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

func (s *QueryServer) Subscribe(wsCtx rpctypes.WSRPCContext, contract string) (*WSEmptyResult, error) {
	evChan, exists := s.Subscriptions.Add(wsCtx.GetRemoteAddr(), contract)
	if exists {
		return &WSEmptyResult{}, nil
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("Caught: WSEvent handler routine panic", "error", r)
				err := fmt.Errorf("Caught: WSEvent handler routine panic")
				wsCtx.WriteRPCResponse(rpctypes.RPCInternalError("Internal server error", err))
				s.Subscriptions.Remove(wsCtx.GetRemoteAddr())
			}
		}()
		for event := range evChan {
			jsonMsg, err := json.Marshal(event)
			if err != nil {
				log.Default.Error("Unable to marshal to JSON", "event", event)
			}
			resp := rpctypes.RPCResponse{
				JSONRPC: "2.0",
				ID:      "0",
			}
			if err != nil {
				resp.Error = &rpctypes.RPCError{
					Code:    -1,
					Message: "Unable to marshal event JSON",
				}
			} else {
				resp.Result = jsonMsg
			}
			wsCtx.TryWriteRPCResponse(resp)
		}
	}()
	return &WSEmptyResult{}, nil
}

func (s *QueryServer) UnSubscribe(wsCtx rpctypes.WSRPCContext) (*WSEmptyResult, error) {
	s.Subscriptions.Remove(wsCtx.GetRemoteAddr())
	return &WSEmptyResult{}, nil
}

func (s *QueryServer) TxReceipt(txHash []byte) ([]byte, error) {
	receiptState := store.PrefixKVStore(lvm.ReceiptPrefix, s.StateProvider.ReadOnlyState())
	return receiptState.Get(txHash), nil
}
