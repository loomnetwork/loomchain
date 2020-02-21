package rpc

import (
	"sync"

	"github.com/gorilla/websocket"
	rpctypes "github.com/tendermint/tendermint/rpc/lib/types"

	"github.com/loomnetwork/go-loom/plugin/types"

	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/vm"
)

type MockQueryService struct {
	mutex         sync.RWMutex
	MethodsCalled []string
}

func (m *MockQueryService) Query(caller, contract string, query []byte, vmType vm.VMType) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"Query"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) Resolve(name string) (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"Resolve"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) Nonce(key, account string) (uint64, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"Nonce"}, m.MethodsCalled...)
	return 0, nil
}

func (m *MockQueryService) Subscribe(wsCtx rpctypes.WSRPCContext, topics []string) (*WSEmptyResult, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"Subscribe"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) UnSubscribe(wsCtx rpctypes.WSRPCContext, topics string) (*WSEmptyResult, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"UnSubscribe"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) QueryEnv() (*config.EnvInfo, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"QueryEnv"}, m.MethodsCalled...)
	return nil, nil
}

// New JSON web3 methods
func (m *MockQueryService) EthBlockNumber() (eth.Quantity, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthBlockNumber"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) EthGetBlockByNumber(block eth.BlockHeight, full bool) (*eth.JsonBlockObject, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthGetBlockByNumber"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) EthGetBlockByHash(hash eth.Data, full bool) (eth.JsonBlockObject, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthGetBlockByHash"}, m.MethodsCalled...)
	return eth.JsonBlockObject{}, nil
}

func (m *MockQueryService) EthGetTransactionReceipt(hash eth.Data) (*eth.JsonTxReceipt, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthGetTransactionReceipt"}, m.MethodsCalled...)
	return &eth.JsonTxReceipt{}, nil
}

func (m *MockQueryService) EthGetTransactionByHash(hash eth.Data) (eth.JsonTxObject, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthGetTransactionByHash"}, m.MethodsCalled...)
	return eth.JsonTxObject{}, nil
}

func (m *MockQueryService) EthGetCode(address eth.Data, block eth.BlockHeight) (eth.Data, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthGetCode"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) EthGetStorageAt(address eth.Data, position string, block eth.BlockHeight) (eth.Data, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthGetStorageAt"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) EthCall(query eth.JsonTxCallObject, block eth.BlockHeight) (eth.Data, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthCall"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) EthGetLogs(filter eth.JsonFilter) ([]eth.JsonLog, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthGetLogs"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) EthGetBlockTransactionCountByHash(hash eth.Data) (eth.Quantity, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthGetBlockTransactionCountByHash"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) EthGetBlockTransactionCountByNumber(block eth.BlockHeight) (eth.Quantity, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthGetBlockTransactionCountByNumber"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) EthGetTransactionByBlockHashAndIndex(
	hash eth.Data, index eth.Quantity,
) (eth.JsonTxObject, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthGetTransactionByBlockHashAndIndex"}, m.MethodsCalled...)
	return eth.JsonTxObject{}, nil
}

func (m *MockQueryService) EthGetTransactionByBlockNumberAndIndex(
	block eth.BlockHeight, index eth.Quantity,
) (eth.JsonTxObject, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthGetTransactionByBlockNumberAndIndex"}, m.MethodsCalled...)
	return eth.JsonTxObject{}, nil
}

func (m *MockQueryService) EthNewBlockFilter() (eth.Quantity, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthNewBlockFilter"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) EthNewPendingTransactionFilter() (eth.Quantity, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthNewPendingTransactionFilter"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) EthUninstallFilter(id eth.Quantity) (bool, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthUninstallFilter"}, m.MethodsCalled...)
	return true, nil
}

func (m *MockQueryService) EthGetFilterChanges(id eth.Quantity) (interface{}, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthGetFilterChanges"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) EthGetFilterLogs(id eth.Quantity) (interface{}, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthGetFilterLogs"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) EthNewFilter(filter eth.JsonFilter) (eth.Quantity, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthNewFilter"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) EthSubscribe(
	conn *websocket.Conn, method eth.Data, filter eth.JsonFilter,
) (id eth.Data, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthSubscribe"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) EthUnsubscribe(id eth.Quantity) (unsubscribed bool, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthUnsubscribe"}, m.MethodsCalled...)
	return true, nil
}

func (m *MockQueryService) EthGetBalance(address eth.Data, block eth.BlockHeight) (eth.Quantity, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthGetBalance"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) EthEstimateGas(query eth.JsonTxCallObject) (eth.Quantity, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthEstimateGas"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) EthGasPrice() (eth.Quantity, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthGasPrice"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) EthNetVersion() (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthNetVersion"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) EthGetTransactionCount(address eth.Data, block eth.BlockHeight) (eth.Quantity, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthGetTransactionCount"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) EthAccounts() ([]eth.Data, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EthAccounts"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) ContractEvents(
	fromBlock uint64, toBlock uint64, contract string,
) (*types.ContractEventsResult, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"ContractEvents"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) GetContractRecord(addr string) (*types.ContractRecordResponse, error) {
	m.MethodsCalled = append([]string{"GetcontractRecord"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) DPOSTotalStaked() (*DPOSTotalStakedResponse, error) {
	m.MethodsCalled = append([]string{"DposTotalStaked"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) GetCanonicalTxHash(block, txIndex uint64, evmTxHash eth.Data) (eth.Data, error) {
	m.MethodsCalled = append([]string{"GetCanonicalTxHash"}, m.MethodsCalled...)
	return "", nil
}

// deprecated function
func (m *MockQueryService) EvmTxReceipt(txHash []byte) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EvmTxReceipt"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) GetEvmCode(contract string) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"GetEvmCode"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) GetEvmLogs(filter string) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"GetEvmLogs"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) NewEvmFilter(filter string) (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"NewEvmFilter"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) NewBlockEvmFilter() (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"NewBlockEvmFilter"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) NewPendingTransactionEvmFilter() (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"NewPendingTransactionEvmFilter"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) GetEvmFilterChanges(id string) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"GetEvmFilterChanges"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) UninstallEvmFilter(id string) (bool, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"UninstallEvmFilter"}, m.MethodsCalled...)
	return true, nil
}

func (m *MockQueryService) GetBlockHeight() (int64, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"GetBlockHeight"}, m.MethodsCalled...)
	return 0, nil
}

func (m *MockQueryService) GetEvmBlockByNumber(number string, full bool) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"GetEvmBlockByNumber"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) GetEvmBlockByHash(hash []byte, full bool) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"GetEvmBlockByHash"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) GetEvmTransactionByHash(txHash []byte) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"GetEvmTransactionByHash"}, m.MethodsCalled...)
	return nil, nil
}

func (m *MockQueryService) EvmSubscribe(wsCtx rpctypes.WSRPCContext, method, filter string) (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EvmSubscribe"}, m.MethodsCalled...)
	return "", nil
}

func (m *MockQueryService) EvmUnSubscribe(id string) (bool, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MethodsCalled = append([]string{"EvmUnSubscribe"}, m.MethodsCalled...)
	return true, nil
}
