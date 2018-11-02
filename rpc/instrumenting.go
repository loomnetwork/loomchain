package rpc

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/tendermint/tendermint/rpc/lib/types"
)

// InstrumentingMiddleware implements QuerySerice interface
type InstrumentingMiddleware struct {
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
	next           QueryService
}

// NewInstrumentingMiddleWare return a new pointer to the struct
func NewInstrumentingMiddleWare(reqCount metrics.Counter, reqLatency metrics.Histogram, next QueryService) *InstrumentingMiddleware {
	return &InstrumentingMiddleware{
		requestCount:   reqCount,
		requestLatency: reqLatency,
		next:           next,
	}
}

// Query calls service Query and captures metrics
func (m InstrumentingMiddleware) Query(caller, contract string, query []byte, vmType vm.VMType) (resp []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "Query", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.Query(caller, contract, query, vmType)
	return
}

// Nonce call service Nonce method and captures metrics
func (m InstrumentingMiddleware) Nonce(key string) (resp uint64, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "Nonce", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.Nonce(key)
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

func (m InstrumentingMiddleware) EvmTxReceipt(txHash []byte) (resp []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EvmTxReceipt", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EvmTxReceipt(txHash)
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

func (m InstrumentingMiddleware) EthGetBlockByNumber(number eth.BlockHeight, full bool) (resp eth.JsonBlockObject, err error) {
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

func (m InstrumentingMiddleware) EthGetBlockTransactionCountByNumber(block eth.BlockHeight) (resp eth.Quantity, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthGetBlockTransactionCountByNumber", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthGetBlockTransactionCountByNumber(block)
	return
}

func (m InstrumentingMiddleware) EthGetTransactionByBlockHashAndIndex(hash eth.Data, index eth.Quantity) (resp eth.JsonTxObject, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "EthGetTransactionByBlockHashAndIndex", "error", fmt.Sprint(err != nil)}
		m.requestCount.With(lvs...).Add(1)
		m.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	resp, err = m.next.EthGetTransactionByBlockHashAndIndex(hash, index)
	return
}

func (m InstrumentingMiddleware) EthGetTransactionByBlockNumberAndIndex(block eth.BlockHeight, index eth.Quantity) (resp eth.JsonTxObject, err error) {
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

