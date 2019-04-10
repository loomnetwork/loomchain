package rpc

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQuerySeverJsonHandler(t *testing.T) {
	qs := &MockQueryService{}
	handler := MakeEthQueryServiceHandler(qs, testlog, nil)

	tests := []struct {
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

	for _, test := range tests {
		payload := `{"jsonrpc":"2.0","method":"` + test.method + `","params":[` + test.params + `],"id":99}`
		req := httptest.NewRequest("POST", "http://localhost/eth", strings.NewReader(payload))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, 200, rec.Result().StatusCode)
		require.Equal(t, test.target, qs.MethodsCalled[0])
	}

	blockPayload := "["
	for _, test := range tests {
		blockPayload += `{"jsonrpc":"2.0","method":"` + test.method + `","params":[` + test.params + `],"id":99}`
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
