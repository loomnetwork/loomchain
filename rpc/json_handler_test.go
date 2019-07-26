package rpc

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/posener/wstest"
	"github.com/stretchr/testify/require"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/events"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/store"
)

var (
	tests = []struct {
		method string
		target string
		params string
	}{
		{"eth_blockNumber", "EthBlockNumber", ``},
		{"eth_getBlockByNumber", "EthGetBlockByNumber", ``},
		{"eth_getBlockByHash", "EthGetBlockByHash", ``},
		{"eth_getTransactionReceipt", "EthGetTransactionReceipt", ``},
		{"eth_getTransactionByHash", "EthGetTransactionByHash", ``},
		{"eth_getCode", "EthGetCode", ``},
		{"eth_call", "EthCall", ``},
		{"eth_getLogs", "EthGetLogs", ``},
		{"eth_getBlockTransactionCountByNumber", "EthGetBlockTransactionCountByNumber", ``},
		{"eth_getBlockTransactionCountByHash", "EthGetBlockTransactionCountByHash", ``},
		{"eth_getTransactionByBlockHashAndIndex", "EthGetTransactionByBlockHashAndIndex", ``},
		{"eth_getTransactionByBlockNumberAndIndex", "EthGetTransactionByBlockNumberAndIndex", ``},
		{"eth_newBlockFilter", "EthNewBlockFilter", ``},
		{"eth_newPendingTransactionFilter", "EthNewPendingTransactionFilter", ``},
		{"eth_uninstallFilter", "EthUninstallFilter", ``},
		{"eth_getFilterChanges", "EthGetFilterChanges", ``},
		{"eth_getFilterLogs", "EthGetFilterLogs", ``},
		{"eth_newFilter", "EthNewFilter", ``},
		{"eth_unsubscribe", "EthUnsubscribe", ``},
		{"eth_getBalance", "EthGetBalance", ``},
		{"eth_estimateGas", "EthEstimateGas", ``},
		{"eth_gasPrice", "EthGasPrice", ``},
		{"net_version", "EthNetVersion", ``},
		{"eth_getTransactionCount", "EthGetTransactionCount", ``},
		{"eth_accounts", "EthAccounts", ``},
	}
)

func TestJsonRpcHandler(t *testing.T) {
	log.Setup("debug", "file://-")
	testlog = log.Root.With("module", "query-server")

	t.Run("Http JSON-RPC", testHttpJsonHandler)
	t.Run("Http JSON-RPC batch", testBatchHttpJsonHandler)
	t.Run("Multi Websocket JSON-RPC", testMultipleWebsocketConnections)
	t.Run("Single Websocket JSON-RPC", testSingleWebsocketConnections)
	t.Run("test eth_subscribe and eth_unsubscribe", testEthSubscribeEthUnSubscribe)
}

func testHttpJsonHandler(t *testing.T) {
	qs := &MockQueryService{}
	handler := MakeEthQueryServiceHandler(qs, testlog, nil)

	for _, test := range tests {
		payload := `{"jsonrpc":"2.0","method":"` + test.method + `","params":[` + test.params + `],"id":99}`
		req := httptest.NewRequest("POST", "http://localhost/eth", strings.NewReader(payload))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, 200, rec.Result().StatusCode)
		require.Equal(t, test.target, qs.MethodsCalled[0])
	}
}

func testBatchHttpJsonHandler(t *testing.T) {
	qs := &MockQueryService{}
	handler := MakeEthQueryServiceHandler(qs, testlog, nil)

	blockPayload := "["
	first := true
	for _, test := range tests {
		if !first {
			blockPayload += ","
		}
		blockPayload += `{"jsonrpc":"2.0","method":"` + test.method + `","params":[` + test.params + `],"id":99}`
		first = false
	}
	blockPayload += "]"
	req := httptest.NewRequest("POST", "http://localhost/eth", strings.NewReader(blockPayload))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Result().StatusCode)
	for i, test := range tests {
		require.Equal(t, test.target, qs.MethodsCalled[len(qs.MethodsCalled)-1-i])
	}
}

func testEthSubscribeEthUnSubscribe(t *testing.T) {
	hub := newHub()
	go hub.run()
	loader := &queryableContractLoader{TMLogger: log.Root.With("module", "contract")}
	eventDispatcher := events.NewLogEventDispatcher()
	eventHandler := loomchain.NewDefaultEventHandler(eventDispatcher)
	createRegistry, err := factory.NewRegistryFactory(factory.LatestRegistryVersion)
	require.NoError(t, err)
	var qs QueryService = &QueryServer{
		StateProvider:    &stateProvider{},
		Loader:           loader,
		CreateRegistry:   createRegistry,
		BlockStore:       store.NewMockBlockStore(),
		AuthCfg:          auth.DefaultConfig(),
		EthSubscriptions: eventHandler.EthSubscriptionSet(),
	}
	handler := MakeEthQueryServiceHandler(qs, testlog, hub)

	dialer := wstest.NewDialer(handler)
	conn, _, err := dialer.Dial("ws://localhost/eth", nil)
	require.NoError(t, err)

	payloadSubscribe := `{"jsonrpc":"2.0","method":"eth_subscribe","params":["logs", {"address": "0x8320fe7702b96808f7bbc0d4a888ed1468216cfd", "topics": ["0xd78a0cb8bb633d06981248b816e7bd33c2a35a6089241d099fa519e361cab902"]}],"id":99}`
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(payloadSubscribe)))
	var resp eth.JsonRpcResponse
	require.NoError(t, conn.ReadJSON(&resp))

	payloadUnsubscribe := `{"id": 1, "method": "eth_unsubscribe", "params": [` + string(resp.Result) + `]}`
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(payloadUnsubscribe)))

	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(payloadSubscribe)))
	require.NoError(t, conn.ReadJSON(&resp))
	require.True(t, len(resp.Result) > 0)

	require.NoError(t, conn.Close())

	require.Error(t, conn.WriteMessage(websocket.TextMessage, []byte(payloadSubscribe)))
}

func testMultipleWebsocketConnections(t *testing.T) {
	hub := newHub()
	go hub.run()
	qs := &MockQueryService{}
	handler := MakeEthQueryServiceHandler(qs, testlog, hub)

	for _, test := range tests {
		dialer := wstest.NewDialer(handler)
		conn, _, err := dialer.Dial("ws://localhost/eth", nil)
		require.NoError(t, err)

		payload := `{"jsonrpc":"2.0","method":"` + test.method + `","params":[` + test.params + `],"id":99}`
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(payload)))
		require.NoError(t, conn.Close())
	}
	time.Sleep(5 * time.Second)
	require.Equal(t, len(tests), len(qs.MethodsCalled))
	for _, test := range tests {
		found := false
		for _, method := range qs.MethodsCalled {
			if test.target == method {
				found = true
				break
			}
		}
		require.True(t, found)
	}
}

func testSingleWebsocketConnections(t *testing.T) {
	hub := newHub()
	go hub.run()
	qs := &MockQueryService{}
	handler := MakeEthQueryServiceHandler(qs, testlog, hub)
	dialer := wstest.NewDialer(handler)
	conn, _, err := dialer.Dial("ws://localhost/eth", nil)
	writeMutex := &sync.Mutex{}
	var wg sync.WaitGroup
	for _, test := range tests {
		require.NoError(t, err)
		payload := `{"jsonrpc":"2.0","method":"` + test.method + `","params":[` + test.params + `],"id":99}`

		wg.Add(1)
		go func() {
			defer wg.Done()
			writeMutex.Lock()
			require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(payload)))
			writeMutex.Unlock()
		}()
	}
	wg.Wait()
	time.Sleep(time.Second)

	require.Equal(t, len(tests), len(qs.MethodsCalled))
	for _, test := range tests {
		found := false
		for _, method := range qs.MethodsCalled {
			if test.target == method {
				found = true
				break
			}
		}
		require.True(t, found)
	}
	require.NoError(t, conn.Close())
}
