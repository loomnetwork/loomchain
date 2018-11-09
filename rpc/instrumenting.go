package rpc

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/gorilla/websocket"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/vm"
	rpctypes "github.com/tendermint/tendermint/rpc/lib/types"
)

// InstrumentingMiddleware implements QuerySerice interface
type InstrumentingMiddleware struct {
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
	next           QueryService
}

// NewInstrumentingMiddleWare return a new pointer to the struct
func NewInstrumentingMiddleWare(
	reqCount metrics.Counter, reqLatency metrics.Histogram, next QueryService,
) *InstrumentingMiddleware {
	return &InstrumentingMiddleware{
		requestCount:   reqCount,
		requestLatency: reqLatency,
		next:           next,
	}
}

// Query calls service Query and captures metrics
func (m InstrumentingMiddleware) Query(
	caller, contract string, query []byte, vmType vm.VMType,
) (resp []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "Query", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.Query(caller, contract, query, vmType)
	return
}

func (m InstrumentingMiddleware) QueryEnv() (resp *config.EnvInfo, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "QueryEnv", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.QueryEnv()

	return resp, err
}

// Nonce call service Nonce method and captures metrics
func (m InstrumentingMiddleware) Nonce(key, account string) (resp uint64, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "Nonce", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.Nonce(key, account)
	return
}

func (m InstrumentingMiddleware) Subscribe(wsCtx rpctypes.WSRPCContext, contracts []string) (*WSEmptyResult, error) {
	return m.next.Subscribe(wsCtx, contracts)
}

func (m InstrumentingMiddleware) UnSubscribe(wsCtx rpctypes.WSRPCContext, topic string) (*WSEmptyResult, error) {
	return m.next.UnSubscribe(wsCtx, topic)
}

func (m InstrumentingMiddleware) Resolve(name string) (resp string, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "Resolve", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.Resolve(name)
	return
}

func (m InstrumentingMiddleware) GetContractRecord(contractAddr string) (resp *registry.Record, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "GetContractRecord", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.GetContractRecord(contractAddr)
	return
}

func (m InstrumentingMiddleware) EvmTxReceipt(txHash []byte) (resp []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EvmTxReceipt", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EvmTxReceipt(txHash)
	return
}

func (m InstrumentingMiddleware) ContractEvents(
	fromBlock uint64, toBlock uint64, contractName string,
) (result *types.ContractEventsResult, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "ContractEvents", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	result, err = m.next.ContractEvents(fromBlock, toBlock, contractName)
	return
}

func (m InstrumentingMiddleware) GetEvmCode(contract string) (resp []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "GetEvmCode", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.GetEvmCode(contract)
	return
}

func (m InstrumentingMiddleware) EthGetCode(address eth.Data, block eth.BlockHeight) (resp eth.Data, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthGetCode", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthGetCode(address, block)
	return
}

func (m InstrumentingMiddleware) GetEvmLogs(filter string) (resp []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "GetEvmLogs", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.GetEvmLogs(filter)
	return
}

func (m InstrumentingMiddleware) NewEvmFilter(filter string) (resp string, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "NewEvmFilter", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.NewEvmFilter(filter)
	return
}

func (m InstrumentingMiddleware) NewBlockEvmFilter() (resp string, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "NewBlockEvmFilter", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.NewBlockEvmFilter()
	return
}

func (m InstrumentingMiddleware) NewPendingTransactionEvmFilter() (resp string, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "NewPendingTransactionEvmFilter", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.NewPendingTransactionEvmFilter()
	return
}

func (m InstrumentingMiddleware) GetEvmFilterChanges(id string) (resp []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "GetEvmFilterChanges", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.GetEvmFilterChanges(id)
	return
}

func (m InstrumentingMiddleware) UninstallEvmFilter(id string) (resp bool, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "UninstallEvmFilter", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.UninstallEvmFilter(id)
	return
}

func (m InstrumentingMiddleware) EthBlockNumber() (resp eth.Quantity, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthBlockNumber", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthBlockNumber()
	return
}

func (m InstrumentingMiddleware) GetBlockHeight() (resp int64, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "GetBlockHeight", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.GetBlockHeight()
	return
}

func (m InstrumentingMiddleware) EthGetBlockByNumber(
	number eth.BlockHeight, full bool,
) (resp eth.JsonBlockObject, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthGetBlockByNumber", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthGetBlockByNumber(number, full)
	return
} //EthGetBlockByHash

func (m InstrumentingMiddleware) GetEvmBlockByNumber(number string, full bool) (resp []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "GetEvmBlockByNumber", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.GetEvmBlockByNumber(number, full)
	return
}

func (m InstrumentingMiddleware) EthGetBlockByHash(hash eth.Data, full bool) (resp eth.JsonBlockObject, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthGetBlockByHash", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthGetBlockByHash(hash, full)
	return
}

func (m InstrumentingMiddleware) GetEvmBlockByHash(hash []byte, full bool) (resp []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "GetEvmBlockByHash", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.GetEvmBlockByHash(hash, full)
	return
}

func (m InstrumentingMiddleware) EvmSubscribe(wsCtx rpctypes.WSRPCContext, method, filter string) (string, error) {
	return m.next.EvmSubscribe(wsCtx, method, filter)
}

func (m InstrumentingMiddleware) EvmUnSubscribe(id string) (resp bool, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EvmUnSubscribe", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EvmUnSubscribe(id)
	return
}

func (m InstrumentingMiddleware) GetEvmTransactionByHash(txHash []byte) (resp []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "GetEvmTransactionByHash", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.GetEvmTransactionByHash(txHash)
	return
}

func (m InstrumentingMiddleware) EthGetTransactionReceipt(hash eth.Data) (resp eth.JsonTxReceipt, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthGetTransactionReceipt", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthGetTransactionReceipt(hash)
	return
}

func (m InstrumentingMiddleware) EthGetTransactionByHash(hash eth.Data) (resp eth.JsonTxObject, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthGetTransactionByHash", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthGetTransactionByHash(hash)
	return
}

func (m InstrumentingMiddleware) EthGetBlockTransactionCountByHash(hash eth.Data) (resp eth.Quantity, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthGetBlockTransactionCountByHash", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthGetBlockTransactionCountByHash(hash)
	return
}

func (m InstrumentingMiddleware) EthGetBlockTransactionCountByNumber(
	block eth.BlockHeight,
) (resp eth.Quantity, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthGetBlockTransactionCountByNumber", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthGetBlockTransactionCountByNumber(block)
	return
}

func (m InstrumentingMiddleware) EthGetTransactionByBlockHashAndIndex(
	hash eth.Data, index eth.Quantity,
) (resp eth.JsonTxObject, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthGetTransactionByBlockHashAndIndex", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthGetTransactionByBlockHashAndIndex(hash, index)
	return
}

func (m InstrumentingMiddleware) EthGetTransactionByBlockNumberAndIndex(
	block eth.BlockHeight, index eth.Quantity,
) (resp eth.JsonTxObject, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthGetTransactionByBlockNumberAndIndex", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthGetTransactionByBlockNumberAndIndex(block, index)
	return
}

func (m InstrumentingMiddleware) EthCall(query eth.JsonTxCallObject, block eth.BlockHeight) (resp eth.Data, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthCall", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthCall(query, block)
	return
}

func (m InstrumentingMiddleware) EthGetLogs(filter eth.JsonFilter) (resp []eth.JsonLog, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthGetLogs", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthGetLogs(filter)
	return
}

func (m InstrumentingMiddleware) EthNewBlockFilter() (resp eth.Quantity, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthNewBlockFilter", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthNewBlockFilter()
	return
}

func (m InstrumentingMiddleware) EthNewPendingTransactionFilter() (resp eth.Quantity, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthNewPendingTransactionFilter", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthNewPendingTransactionFilter()
	return
}

func (m InstrumentingMiddleware) EthUninstallFilter(id eth.Quantity) (resp bool, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthUninstallFilter", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthUninstallFilter(id)
	return
}

func (m InstrumentingMiddleware) EthGetFilterChanges(id eth.Quantity) (resp interface{}, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthGetFilterChanges", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthGetFilterChanges(id)
	return
}

func (m InstrumentingMiddleware) EthGetFilterLogs(id eth.Quantity) (resp interface{}, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthGetFilterLogs", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthGetFilterLogs(id)
	return
}

func (m InstrumentingMiddleware) EthNewFilter(filter eth.JsonFilter) (resp eth.Quantity, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthNewFilter", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthNewFilter(filter)
	return
}

func (m InstrumentingMiddleware) EthSubscribe(
	conn *websocket.Conn, method eth.Data, filter eth.JsonFilter,
) (resp eth.Data, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthSubscribe", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthSubscribe(conn, method, filter)
	return
}

func (m InstrumentingMiddleware) EthUnsubscribe(id eth.Quantity) (resp bool, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthUnsubscribe", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthUnsubscribe(id)
	return
}

func (m InstrumentingMiddleware) EthGetBalance(address eth.Data, block eth.BlockHeight) (resp eth.Quantity, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthGetBalance", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthGetBalance(address, block)
	return
}

func (m InstrumentingMiddleware) EthEstimateGas(query eth.JsonTxCallObject) (resp eth.Quantity, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthEstimateGas", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthEstimateGas(query)
	return
}

func (m InstrumentingMiddleware) EthGasPrice() (resp eth.Quantity, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthGasPrice", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthGasPrice()
	return
}

func (m InstrumentingMiddleware) EthNetVersion() (resp string, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthNetVersion", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthNetVersion()
	return
}

func (m InstrumentingMiddleware) EthAccounts() (resp []eth.Data, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthAccounts", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthAccounts()
	return
}

func (m InstrumentingMiddleware) EthGetTransactionCount(
	local eth.Data, block eth.BlockHeight,
) (resp eth.Quantity, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthGetTransactionCount", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthGetTransactionCount(local, block)
	return
}
