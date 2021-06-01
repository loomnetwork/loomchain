package rpc

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/libs/pubsub"
	rpccore "github.com/tendermint/tendermint/rpc/core"
	rpcserver "github.com/tendermint/tendermint/rpc/lib/server"
	rpctypes "github.com/tendermint/tendermint/rpc/lib/types"
	"golang.org/x/net/context"

	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/config"
	"github.com/loomnetwork/loomchain/eth/subs"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/vm"
)

// QueryService provides necessary methods for the client to query application states
type QueryService interface {
	Query(caller, contract string, query []byte, vmType vm.VMType) ([]byte, error)
	Resolve(name string) (string, error)
	Nonce(key, account string) (uint64, error)
	Subscribe(wsCtx rpctypes.WSRPCContext, topics []string) (*WSEmptyResult, error)
	UnSubscribe(wsCtx rpctypes.WSRPCContext, topics string) (*WSEmptyResult, error)
	QueryEnv() (*config.EnvInfo, error)
	// New JSON web3 methods
	EthBlockNumber() (eth.Quantity, error)
	EthGetBlockByNumber(block eth.BlockHeight, full bool) (*eth.JsonBlockObject, error)
	EthGetBlockByHash(hash eth.Data, full bool) (eth.JsonBlockObject, error)
	EthGetTransactionReceipt(hash eth.Data) (*eth.JsonTxReceipt, error)
	EthGetTransactionByHash(hash eth.Data) (eth.JsonTxObject, error)
	EthGetCode(address eth.Data, block eth.BlockHeight) (eth.Data, error)
	EthGetStorageAt(address eth.Data, position string, block eth.BlockHeight) (eth.Data, error)
	EthCall(query eth.JsonTxCallObject, block eth.BlockHeight) (eth.Data, error)
	EthGetLogs(filter eth.JsonFilter) ([]eth.JsonLog, error)
	EthGetBlockTransactionCountByHash(hash eth.Data) (eth.Quantity, error)
	EthGetBlockTransactionCountByNumber(block eth.BlockHeight) (eth.Quantity, error)
	EthGetTransactionByBlockHashAndIndex(hash eth.Data, index eth.Quantity) (eth.JsonTxObject, error)
	EthGetTransactionByBlockNumberAndIndex(block eth.BlockHeight, index eth.Quantity) (eth.JsonTxObject, error)

	EthNewBlockFilter() (eth.Quantity, error)
	EthNewPendingTransactionFilter() (eth.Quantity, error)
	EthUninstallFilter(id eth.Quantity) (bool, error)
	EthGetFilterChanges(id eth.Quantity) (interface{}, error)
	EthGetFilterLogs(id eth.Quantity) (interface{}, error)

	EthNewFilter(filter eth.JsonFilter) (eth.Quantity, error)
	EthSubscribe(conn *websocket.Conn, method eth.Data, filter eth.JsonFilter) (id eth.Data, err error)
	EthUnsubscribe(id eth.Quantity) (unsubscribed bool, err error)

	EthGetBalance(address eth.Data, block eth.BlockHeight) (eth.Quantity, error)
	EthEstimateGas(query eth.JsonTxCallObject, block eth.BlockHeight) (eth.Quantity, error)
	EthGasPrice() (eth.Quantity, error)
	EthNetVersion() (string, error)
	EthGetTransactionCount(local eth.Data, block eth.BlockHeight) (eth.Quantity, error)
	EthAccounts() ([]eth.Data, error)

	ContractEvents(fromBlock uint64, toBlock uint64, contract string) (*types.ContractEventsResult, error)
	GetContractRecord(contractAddr string) (*types.ContractRecordResponse, error)
	DPOSTotalStaked() (*DPOSTotalStakedResponse, error)
	GetCanonicalTxHash(block, txIndex uint64, evmTxHash eth.Data) (eth.Data, error)

	// deprecated function
	EvmTxReceipt(txHash []byte) ([]byte, error)
	GetEvmCode(contract string) ([]byte, error)
	GetEvmLogs(filter string) ([]byte, error)
	NewEvmFilter(filter string) (string, error)
	NewBlockEvmFilter() (string, error)
	NewPendingTransactionEvmFilter() (string, error)
	GetEvmFilterChanges(id string) ([]byte, error)
	UninstallEvmFilter(id string) (bool, error)
	GetBlockHeight() (int64, error)
	GetEvmBlockByNumber(number string, full bool) ([]byte, error)
	GetEvmBlockByHash(hash []byte, full bool) ([]byte, error)
	GetEvmTransactionByHash(txHash []byte) ([]byte, error)
	EvmSubscribe(wsCtx rpctypes.WSRPCContext, method, filter string) (string, error)
	EvmUnSubscribe(id string) (bool, error)
}

type QueryEventBus struct {
	Subs    loomchain.SubscriptionSet
	EthSubs subs.LegacyEthSubscriptionSet
}

func (b *QueryEventBus) Subscribe(ctx context.Context,
	subscriber string, query pubsub.Query, out chan<- interface{}) error {
	return nil
}

func (b *QueryEventBus) Unsubscribe(ctx context.Context, subscriber string, query pubsub.Query) error {
	return nil
}

func (b *QueryEventBus) UnsubscribeAll(ctx context.Context, subscriber string) error {
	log.Debug("Removing WS event subscriber", "address", subscriber)
	b.EthSubs.Purge(subscriber)
	b.Subs.Purge(subscriber)
	return nil
}

// makeQueryServiceHandler returns a http handler mapping to query service
func MakeQueryServiceHandler(svc QueryService, logger log.TMLogger, bus *QueryEventBus) http.Handler {
	// set up websocket route
	codec := amino.NewCodec()
	wsmux := http.NewServeMux()
	routes := map[string]*rpcserver.RPCFunc{}
	routes["query"] = rpcserver.NewRPCFunc(svc.Query, "caller,contract,query,vmType")
	routes["env"] = rpcserver.NewRPCFunc(svc.QueryEnv, "")
	routes["nonce"] = rpcserver.NewRPCFunc(svc.Nonce, "key,account")
	routes["subevents"] = rpcserver.NewWSRPCFunc(svc.Subscribe, "topics")
	routes["unsubevents"] = rpcserver.NewWSRPCFunc(svc.UnSubscribe, "topic")
	routes["resolve"] = rpcserver.NewRPCFunc(svc.Resolve, "name")
	routes["evmtxreceipt"] = rpcserver.NewRPCFunc(svc.EvmTxReceipt, "txHash")
	routes["getevmcode"] = rpcserver.NewRPCFunc(svc.GetEvmCode, "contract")
	routes["getevmlogs"] = rpcserver.NewRPCFunc(svc.GetEvmLogs, "filter")
	routes["newevmfilter"] = rpcserver.NewRPCFunc(svc.NewEvmFilter, "filter")
	routes["newblockevmfilter"] = rpcserver.NewRPCFunc(svc.NewBlockEvmFilter, "")
	routes["newpendingtransactionevmfilter"] = rpcserver.NewRPCFunc(svc.NewPendingTransactionEvmFilter, "")
	routes["getevmfilterchanges"] = rpcserver.NewRPCFunc(svc.GetEvmFilterChanges, "id")
	routes["evmunsubscribe"] = rpcserver.NewRPCFunc(svc.EvmUnSubscribe, "id")
	routes["uninstallevmfilter"] = rpcserver.NewRPCFunc(svc.UninstallEvmFilter, "id")
	routes["getblockheight"] = rpcserver.NewRPCFunc(svc.GetBlockHeight, "")
	routes["getevmblockbynumber"] = rpcserver.NewRPCFunc(svc.GetEvmBlockByNumber, "number,full")
	routes["getevmblockbyhash"] = rpcserver.NewRPCFunc(svc.GetEvmBlockByHash, "hash,full")
	routes["getevmtransactionbyhash"] = rpcserver.NewRPCFunc(svc.GetEvmTransactionByHash, "txHash")
	routes["evmsubscribe"] = rpcserver.NewWSRPCFunc(svc.EvmSubscribe, "method,filter")
	routes["contractevents"] = rpcserver.NewRPCFunc(svc.ContractEvents, "fromBlock,toBlock,contract")
	routes["contractrecord"] = rpcserver.NewRPCFunc(svc.GetContractRecord, "contract")
	routes["dpos_total_staked"] = rpcserver.NewRPCFunc(svc.DPOSTotalStaked, "")
	routes["canonical_tx_hash"] = rpcserver.NewRPCFunc(svc.GetCanonicalTxHash, "block,txIndex,evmTxHash")
	rpcserver.RegisterRPCFuncs(wsmux, routes, codec, logger)
	wm := rpcserver.NewWebsocketManager(routes, codec, rpcserver.EventSubscriber(bus))
	wsmux.HandleFunc("/queryws", wm.WebsocketHandler)

	// setup default route
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		wsmux.ServeHTTP(w, req)
	})

	// setup metrics route
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func createDefaultEthRoutes(svc QueryService, chainID string) map[string]eth.RPCFunc {
	routes := map[string]eth.RPCFunc{}
	routes["eth_blockNumber"] = eth.NewRPCFunc(svc.EthBlockNumber, "")
	routes["eth_getBlockByNumber"] = eth.NewRPCFunc(svc.EthGetBlockByNumber, "block,full")
	routes["eth_getBlockByHash"] = eth.NewRPCFunc(svc.EthGetBlockByHash, "hash,full")
	routes["eth_getTransactionReceipt"] = eth.NewRPCFunc(svc.EthGetTransactionReceipt, "hash")
	routes["eth_getTransactionByHash"] = eth.NewRPCFunc(svc.EthGetTransactionByHash, "hash")
	routes["eth_getCode"] = eth.NewRPCFunc(svc.EthGetCode, "address,block")
	routes["eth_getStorageAt"] = eth.NewRPCFunc(svc.EthGetStorageAt, "address,position,block")
	routes["eth_call"] = eth.NewRPCFunc(svc.EthCall, "query,block")
	routes["eth_getLogs"] = eth.NewRPCFunc(svc.EthGetLogs, "filter")
	routes["eth_getBlockTransactionCountByNumber"] = eth.NewRPCFunc(svc.EthGetBlockTransactionCountByNumber, "block")
	routes["eth_getBlockTransactionCountByHash"] = eth.NewRPCFunc(svc.EthGetBlockTransactionCountByHash, "hash")
	routes["eth_getTransactionByBlockHashAndIndex"] = eth.NewRPCFunc(
		svc.EthGetTransactionByBlockHashAndIndex, "block,index")
	routes["eth_getTransactionByBlockNumberAndIndex"] = eth.NewRPCFunc(
		svc.EthGetTransactionByBlockNumberAndIndex, "hash,index")

	routes["eth_newBlockFilter"] = eth.NewRPCFunc(svc.EthNewBlockFilter, "")
	routes["eth_newPendingTransactionFilter"] = eth.NewRPCFunc(svc.EthNewPendingTransactionFilter, "")
	routes["eth_uninstallFilter"] = eth.NewRPCFunc(svc.EthUninstallFilter, "id")
	routes["eth_getFilterChanges"] = eth.NewRPCFunc(svc.EthGetFilterChanges, "id")
	routes["eth_getFilterLogs"] = eth.NewRPCFunc(svc.EthGetFilterLogs, "id")

	routes["eth_newFilter"] = eth.NewRPCFunc(svc.EthNewFilter, "filter")
	routes["eth_subscribe"] = eth.NewWSRPCFunc(svc.EthSubscribe, "conn,method,filter")
	routes["eth_unsubscribe"] = eth.NewRPCFunc(svc.EthUnsubscribe, "id")

	routes["eth_accounts"] = eth.NewRPCFunc(svc.EthAccounts, "")
	routes["eth_getBalance"] = eth.NewRPCFunc(svc.EthGetBalance, "address,block")
	routes["eth_estimateGas"] = eth.NewRPCFunc(svc.EthEstimateGas, "query,block")
	routes["eth_gasPrice"] = eth.NewRPCFunc(svc.EthGasPrice, "")
	routes["net_version"] = eth.NewRPCFunc(svc.EthNetVersion, "")
	routes["eth_getTransactionCount"] = eth.NewRPCFunc(svc.EthGetTransactionCount, "local,block")
	routes["eth_sendRawTransaction"] = NewSendRawTransactionRPCFunc(chainID, rpccore.BroadcastTxSync)
	return routes
}

// MakeEthQueryServiceHandler returns an http handler mapping to query service
func MakeEthQueryServiceHandler(logger log.TMLogger, hub *Hub, routes map[string]eth.RPCFunc) http.Handler {
	wsmux := http.NewServeMux()
	RegisterRPCFuncs(wsmux, routes, logger, hub)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		wsmux.ServeHTTP(w, req)
	})
	return mux
}

// MakeUnsafeQueryServiceHandler returns a http handler for unsafe RPC routes
func MakeUnsafeQueryServiceHandler(logger log.TMLogger) http.Handler {
	codec := amino.NewCodec()
	mux := http.NewServeMux()
	routes := map[string]*rpcserver.RPCFunc{}
	routes["dial_seeds"] = rpcserver.NewRPCFunc(rpccore.UnsafeDialSeeds, "seeds")
	routes["dial_peers"] = rpcserver.NewRPCFunc(rpccore.UnsafeDialPeers, "peers,persistent")
	routes["unsafe_flush_mempool"] = rpcserver.NewRPCFunc(rpccore.UnsafeFlushMempool, "")

	// profiler API
	routes["unsafe_start_cpu_profiler"] = rpcserver.NewRPCFunc(rpccore.UnsafeStartCPUProfiler, "filename")
	routes["unsafe_stop_cpu_profiler"] = rpcserver.NewRPCFunc(rpccore.UnsafeStopCPUProfiler, "")
	routes["unsafe_write_heap_profile"] = rpcserver.NewRPCFunc(rpccore.UnsafeWriteHeapProfile, "filename")

	rpcserver.RegisterRPCFuncs(mux, routes, codec, logger)
	return mux
}
