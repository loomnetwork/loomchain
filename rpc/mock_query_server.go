package rpc

import (
	"github.com/gorilla/websocket"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/vm"
	rpctypes "github.com/tendermint/tendermint/rpc/lib/types"
)

type MockQueryService struct {
	LastMethodCalled string
}

func (m *MockQueryService) Query(caller, contract string, query []byte, vmType vm.VMType) ([]byte, error) {
	m.LastMethodCalled = "Query"
	return nil, nil
}

func (m *MockQueryService) Resolve(name string) (string, error) {
	m.LastMethodCalled = "Resolve"
	return "", nil
}

func (m *MockQueryService) Nonce(key, account string) (uint64, error) {
	m.LastMethodCalled = "Nonce"
	return 0, nil
}

func (m *MockQueryService) Subscribe(wsCtx rpctypes.WSRPCContext, topics []string) (*WSEmptyResult, error) {
	m.LastMethodCalled = "Subscribe"
	return nil, nil
}

func (m *MockQueryService) UnSubscribe(wsCtx rpctypes.WSRPCContext, topics string) (*WSEmptyResult, error) {
	m.LastMethodCalled = "UnSubscribe"
	return nil, nil
}

func (m *MockQueryService) QueryEnv() (*config.EnvInfo, error) {
	m.LastMethodCalled = "QueryEnv"
	return nil, nil
}

// New JSON web3 methods
func (m *MockQueryService) EthBlockNumber() (eth.Quantity, error) {
	m.LastMethodCalled = "EthBlockNumber"
	return "", nil
}

func (m *MockQueryService) EthGetBlockByNumber(block eth.BlockHeight, full bool) (eth.JsonBlockObject, error) {
	m.LastMethodCalled = "EthGetBlockByNumber"
	return eth.JsonBlockObject{}, nil
}

func (m *MockQueryService) EthGetBlockByHash(hash eth.Data, full bool) (eth.JsonBlockObject, error) {
	m.LastMethodCalled = "EthGetBlockByHash"
	return eth.JsonBlockObject{}, nil
}

func (m *MockQueryService) EthGetTransactionReceipt(hash eth.Data) (eth.JsonTxReceipt, error) {
	m.LastMethodCalled = "EthGetTransactionReceipt"
	return eth.JsonTxReceipt{}, nil
}

func (m *MockQueryService) EthGetTransactionByHash(hash eth.Data) (eth.JsonTxObject, error) {
	m.LastMethodCalled = "EthGetTransactionByHash"
	return eth.JsonTxObject{}, nil
}

func (m *MockQueryService) EthGetCode(address eth.Data, block eth.BlockHeight) (eth.Data, error) {
	m.LastMethodCalled = "EthGetCode"
	return "", nil
}

func (m *MockQueryService) EthCall(query eth.JsonTxCallObject, block eth.BlockHeight) (eth.Data, error) {
	m.LastMethodCalled = "EthCall"
	return "", nil
}

func (m *MockQueryService) EthGetLogs(filter eth.JsonFilter) ([]eth.JsonLog, error) {
	m.LastMethodCalled = "EthGetLogs"
	return nil, nil
}

func (m *MockQueryService) EthGetBlockTransactionCountByHash(hash eth.Data) (eth.Quantity, error) {
	m.LastMethodCalled = "EthGetBlockTransactionCountByHash"
	return "", nil
}

func (m *MockQueryService) EthGetBlockTransactionCountByNumber(block eth.BlockHeight) (eth.Quantity, error) {
	m.LastMethodCalled = "EthGetBlockTransactionCountByNumber"
	return "", nil
}

func (m *MockQueryService) EthGetTransactionByBlockHashAndIndex(hash eth.Data, index eth.Quantity) (eth.JsonTxObject, error) {
	m.LastMethodCalled = "EthGetTransactionByBlockHashAndIndex"
	return eth.JsonTxObject{}, nil
}

func (m *MockQueryService) EthGetTransactionByBlockNumberAndIndex(block eth.BlockHeight, index eth.Quantity) (eth.JsonTxObject, error) {
	m.LastMethodCalled = "EthGetTransactionByBlockNumberAndIndex"
	return eth.JsonTxObject{}, nil
}

func (m *MockQueryService) EthNewBlockFilter() (eth.Quantity, error) {
	m.LastMethodCalled = "EthNewBlockFilter"
	return "", nil
}

func (m *MockQueryService) EthNewPendingTransactionFilter() (eth.Quantity, error) {
	m.LastMethodCalled = "EthNewPendingTransactionFilter"
	return "", nil
}

func (m *MockQueryService) EthUninstallFilter(id eth.Quantity) (bool, error) {
	m.LastMethodCalled = "EthUninstallFilter"
	return true, nil
}

func (m *MockQueryService) EthGetFilterChanges(id eth.Quantity) (interface{}, error) {
	m.LastMethodCalled = "EthGetFilterChanges"
	return nil, nil
}

func (m *MockQueryService) EthGetFilterLogs(id eth.Quantity) (interface{}, error) {
	m.LastMethodCalled = "EthGetFilterLogs"
	return nil, nil
}

func (m *MockQueryService) EthNewFilter(filter eth.JsonFilter) (eth.Quantity, error) {
	m.LastMethodCalled = "EthNewFilter"
	return "", nil
}

func (m *MockQueryService) EthSubscribe(conn websocket.Conn, method eth.Data, filter eth.JsonFilter) (id eth.Data, err error) {
	m.LastMethodCalled = "EthSubscribe"
	return "", nil
}

func (m *MockQueryService) EthUnsubscribe(id eth.Quantity) (unsubscribed bool, err error) {
	m.LastMethodCalled = "EthUnsubscribe"
	return true, nil
}

func (m *MockQueryService) EthGetBalance(address eth.Data, block eth.BlockHeight) (eth.Quantity, error) {
	m.LastMethodCalled = "EthGetBalance"
	return "", nil
}

func (m *MockQueryService) EthEstimateGas(query eth.JsonTxCallObject) (eth.Quantity, error) {
	m.LastMethodCalled = "EthEstimateGas"
	return "", nil
}

func (m *MockQueryService) EthGasPrice() (eth.Quantity, error) {
	m.LastMethodCalled = "EthGasPrice"
	return "", nil
}

func (m *MockQueryService) EthNetVersion() (string, error) {
	m.LastMethodCalled = "EthNetVersion"
	return "", nil
}

func (m *MockQueryService) ContractEvents(fromBlock uint64, toBlock uint64, contract string) (*types.ContractEventsResult, error) {
	m.LastMethodCalled = "ContractEvents"
	return nil, nil
}

// deprecated function
func (m *MockQueryService) EvmTxReceipt(txHash []byte) ([]byte, error) {
	m.LastMethodCalled = "EvmTxReceipt"
	return nil, nil
}

func (m *MockQueryService) GetEvmCode(contract string) ([]byte, error) {
	m.LastMethodCalled = "GetEvmCode"
	return nil, nil
}

func (m *MockQueryService) GetEvmLogs(filter string) ([]byte, error) {
	m.LastMethodCalled = "GetEvmLogs"
	return nil, nil
}

func (m *MockQueryService) NewEvmFilter(filter string) (string, error) {
	m.LastMethodCalled = "NewEvmFilter"
	return "", nil
}

func (m *MockQueryService) NewBlockEvmFilter() (string, error) {
	m.LastMethodCalled = "NewBlockEvmFilter"
	return "", nil
}

func (m *MockQueryService) NewPendingTransactionEvmFilter() (string, error) {
	m.LastMethodCalled = "NewPendingTransactionEvmFilter"
	return "", nil
}

func (m *MockQueryService) GetEvmFilterChanges(id string) ([]byte, error) {
	m.LastMethodCalled = "GetEvmFilterChanges"
	return nil, nil
}

func (m *MockQueryService) UninstallEvmFilter(id string) (bool, error) {
	m.LastMethodCalled = "UninstallEvmFilter"
	return true, nil
}

func (m *MockQueryService) GetBlockHeight() (int64, error) {
	m.LastMethodCalled = "GetBlockHeight"
	return 0, nil
}

func (m *MockQueryService) GetEvmBlockByNumber(number string, full bool) ([]byte, error) {
	m.LastMethodCalled = "GetEvmBlockByNumber"
	return nil, nil
}

func (m *MockQueryService) GetEvmBlockByHash(hash []byte, full bool) ([]byte, error) {
	m.LastMethodCalled = "GetEvmBlockByHash"
	return nil, nil
}

func (m *MockQueryService) GetEvmTransactionByHash(txHash []byte) ([]byte, error) {
	m.LastMethodCalled = "GetEvmTransactionByHash"
	return nil, nil
}

func (m *MockQueryService) EvmSubscribe(wsCtx rpctypes.WSRPCContext, method, filter string) (string, error) {
	m.LastMethodCalled = "EvmSubscribe"
	return "", nil
}

func (m *MockQueryService) EvmUnSubscribe(id string) (bool, error) {
	m.LastMethodCalled = "EvmUnSubscribe"
	return true, nil
}
