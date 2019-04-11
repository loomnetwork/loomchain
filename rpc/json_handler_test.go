package rpc

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/posener/wstest"
	"github.com/stretchr/testify/require"

	"github.com/loomnetwork/loomchain/log"
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
	t.Run("Http JSON-RPC batch", testBartchHttpJsonHandler)
	t.Run("Multi Websocket JSON-RPC", testMultipleWebsocketConnections)
	t.Run("Multi Websocket JSON-RPC", testSingleWebsocketConnections)
}

func testHttpJsonHandler(t *testing.T) {
	qs :=  &MockQueryService{}
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

func testBartchHttpJsonHandler(t *testing.T) {
	qs :=  &MockQueryService{}
	handler := MakeEthQueryServiceHandler(qs, testlog, nil)

	blockPayload := "["
	first := true
	for _, test := range tests {
		if !first {
			blockPayload+=","
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

func testMultipleWebsocketConnections(t *testing.T) {
	hub := newHub()
	go hub.run()
	qs :=  &MockQueryService{}
	handler := MakeEthQueryServiceHandler(qs, testlog, hub)

	for _, test := range tests {
		dialer := wstest.NewDialer(handler)
		conn, _, err := dialer.Dial("ws://localhost/eth", nil)
		require.NoError(t, err)

		payload := `{"jsonrpc":"2.0","method":"` + test.method + `","params":[` + test.params + `],"id":99}`
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(payload)))
	}
	time.Sleep(time.Second)
	require.Equal(t, len(tests), len(qs.MethodsCalled))
	for _, test := range tests {
		found := false
		for _, method := range qs.MethodsCalled {
			if test.target == method {
				found = true
			}
		}
		require.True(t, found)
	}
}

func testSingleWebsocketConnections(t *testing.T) {
	//t.Skip()
	log.Setup("error", "file://-")
	testlog = log.Root.With("module", "query-server")
	hub := newHub()
	go hub.run()
	qs := &MockQueryService{}
	handler := MakeEthQueryServiceHandler(qs, testlog, hub)

	for _, test := range tests {
		dialer := wstest.NewDialer(handler)
		conn, _, err := dialer.Dial("ws://localhost/eth", nil)
		require.NoError(t, err)
		payload := `{"jsonrpc":"2.0","method":"` + test.method + `","params":[` + test.params + `],"id":99}`

		go func() {
			require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(payload)))
		}()
		time.Sleep(100*time.Microsecond)
	}
	time.Sleep(time.Second)
	require.Equal(t, len(tests), len(qs.MethodsCalled))
	for _, test := range tests {
		found := false
		for _, method := range qs.MethodsCalled {
			if test.target == method {
				found = true
			}
		}
		require.True(t, found)
	}
}