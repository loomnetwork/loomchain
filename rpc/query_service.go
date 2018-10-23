package rpc

import (
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/subs"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/rpc/lib/server"

	"net/http"

	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/rpc/lib/types"
	"golang.org/x/net/context"
)

// QueryService provides neccesary methods for the client to query appication states
type QueryService interface {
	Query(caller, contract string, query []byte, vmType vm.VMType) ([]byte, error) // Similar to Eth_call for EVM
	Resolve(name string) (string, error)
	Nonce(key string) (uint64, error)
	Subscribe(wsCtx rpctypes.WSRPCContext, topics []string) (*WSEmptyResult, error)
	UnSubscribe(wsCtx rpctypes.WSRPCContext, topics string) (*WSEmptyResult, error)

	// New JSON web3 methods
	EthBlockNumber() (height string, err error)

	/*
		//EthSubscribe(req string) (rsp string, err error))
		//EthUnsubscribe(req string) (rsp string, err error)
		EthGetBlockTransactionCountByHash(req string) (rsp string, err error)
		EthGetBlockTransactionCountByNumber(req string) (rsp string, err error)
		EthGetCode(req string) (rsp string, err error)
		EthCall(req string) (rsp string, err error) // similar to Query for EVM
		EthGetBlockByHash(req string) (rsp string, err error)
		EthGetBlockByNumber(req string) (rsp string, err error)
		EthGetTransactionByHash(req string) (rsp string, err error)
		EthGetTransactionByBlockHashAndIndex(req string) (rsp string, err error)
		EthGetTransactionByBlockNumberAndIndex(req string) (rsp string, err error)
		EthGetTransactionReceipt(req string) (rsp string, err error)
		EthNewFilter(req string) (rsp string, err error)
		EthNewBlockFilter(req string) (rsp string, err error)
		EthNewPendingTransactionFilter(req string) (rsp string, err error)
		EthUninstallFilter(req string) (rsp string, err error)
		EthGetFilterChanges(req string) (rsp string, err error)
		EthGetFilterLogs(req string) (rsp string, err error)
		EthGetLogs(req string) (rsp string, err error)
	*/

	// deprecated protbuf function
	EvmSubscribe(wsCtx rpctypes.WSRPCContext, method, filter string) (string, error)
	EvmUnSubscribe(id string) (bool, error)
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
}

type QueryEventBus struct {
	Subs    loomchain.SubscriptionSet
	EthSubs subs.EthSubscriptionSet
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
	routes["nonce"] = rpcserver.NewRPCFunc(svc.Nonce, "key")
	routes["resolve"] = rpcserver.NewRPCFunc(svc.Resolve, "name")
	routes["subevents"] = rpcserver.NewWSRPCFunc(svc.Subscribe, "topics")
	routes["unsubevents"] = rpcserver.NewWSRPCFunc(svc.UnSubscribe, "topic")

	// deprecated protbuf function
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
	rpcserver.RegisterRPCFuncs(wsmux, routes, codec, logger)
	wm := rpcserver.NewWebsocketManager(routes, codec, rpcserver.EventSubscriber(bus))
	wsmux.HandleFunc("/queryws", wm.WebsocketHandler)

	// set up json route
	//muxJson:= http.NewServeMux()
	/*
		routesJson := map[string]*LoomApiMethod{}
		routesJson["query"] = newLoomApiMethod(svc.Query, "caller,contract,query,vmType", true)
		routesJson["nonce"] = newLoomApiMethod(svc.Nonce, "key", true)
		routesJson["resolve"] = newLoomApiMethod(svc.Resolve, "name", true)
		routesJson["eth_blockNumber"] = newLoomApiMethod(svc.EthBlockNumber, "", false)
		RegisterJsonFunc(wsmux, routesJson, logger)
	*/
	// setup default route
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
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

// makeQueryServiceHandler returns a http handler mapping to query service
func MakeJsonoQueryServiceHandler(svc QueryService, logger log.TMLogger) http.Handler {
	wsmux := http.NewServeMux()
	routesJson := map[string]*LoomApiMethod{}
	routesJson["query"] = newLoomApiMethod(svc.Query, "caller,contract,query,vmType", true)
	routesJson["nonce"] = newLoomApiMethod(svc.Nonce, "key", true)
	routesJson["resolve"] = newLoomApiMethod(svc.Resolve, "name", true)
	routesJson["eth_blockNumber"] = newLoomApiMethod(svc.EthBlockNumber, "", false)
	RegisterJsonFunc(wsmux, routesJson, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
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
