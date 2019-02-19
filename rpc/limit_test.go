package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/mempool"

	abci "github.com/tendermint/tendermint/abci/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/rpc/lib/types"
)

const (
	CodeTypeFail = 1
)

func TestLimitVisits(t *testing.T) {
	confirmLimited(t, newNextHandler(t, CodeTypeFail), true)
	confirmLimited(t, newNextHandlerError(t, rpctypes.RPCError{
		Code:    -32603,
		Message: "Internal error",
		Data:    mempool.ErrTxInCache.Error(),
	}), true)

	confirmLimited(t, newNextHandler(t, abci.CodeTypeOK), false)
	confirmLimited(t, newNextHandlerError(t, rpctypes.RPCError{
		Code:    -32603,
		Message: "Internal error",
		Data:    "Some random error",
	}), false)
}

func confirmLimited(t *testing.T, next http.Handler, resultLimited bool) {
	handler := limitVisits(next)
	vvv := visitors
	lll := len(visitors)
	vvv = vvv
	lll = lll
	require.Equal(t, 0, len(visitors))
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	ip := getRealAddr(req)
	for i := 1; i <= limiterCount; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		require.Equal(t, 200, w.Result().StatusCode)
		_, exits := visitors[ip]
		if exits != resultLimited {
			require.Equal(t, exits, resultLimited)
		}
		require.Equal(t, exits, resultLimited)
		if resultLimited {
			limiter, err := visitors[ip].limiter.Peek(context.TODO(), keyVisitors)
			require.NoError(t, err)
			require.Equal(t, int64(limiterCount-i), limiter.Remaining)
		}

	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, 200, w.Result().StatusCode)
	_, exits := visitors[ip]
	require.Equal(t, exits, resultLimited)
	if resultLimited {
		limiter, err := visitors[ip].limiter.Peek(context.TODO(), keyVisitors)
		require.NoError(t, err)
		require.Equal(t, int64(0), limiter.Remaining)
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if resultLimited {
		require.Equal(t, 429, w.Result().StatusCode)
	} else {
		require.Equal(t, 200, w.Result().StatusCode)
	}
	emptyVisitors()
}

type nextHandlerError struct {
	http.Handler
	t        *testing.T
	rpcError rpctypes.RPCError
}

func newNextHandlerError(t *testing.T, err rpctypes.RPCError) http.Handler {
	return nextHandlerError{
		t:        t,
		rpcError: err,
	}
}

func (h nextHandlerError) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	resp := rpctypes.RPCResponse{JSONRPC: "2.0", Error: &h.rpcError}
	jsonBytes, err := json.MarshalIndent(resp, "", "  ")
	require.NoError(h.t, err)
	_, err = w.Write(jsonBytes)
	require.NoError(h.t, err)
}

type nextHandler struct {
	http.Handler
	t             *testing.T
	checkTxResult uint32
}

func newNextHandler(t *testing.T, checkTxResult uint32) http.Handler {
	return nextHandler{
		t:             t,
		checkTxResult: checkTxResult,
	}
}

func (h nextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	result := ctypes.ResultBroadcastTx{
		Code: h.checkTxResult,
	}
	var js []byte
	js, err := cdc.MarshalJSON(result)
	require.NoError(h.t, err)
	rawMsg := json.RawMessage(js)
	resp := rpctypes.RPCResponse{JSONRPC: "2.0", Result: rawMsg}

	jsonBytes, err := json.MarshalIndent(resp, "", "  ")
	require.NoError(h.t, err)
	_, err = w.Write(jsonBytes)
	require.NoError(h.t, err)
}

func emptyVisitors() {
	mtx.Lock()
	defer mtx.Unlock()
	for ip := range visitors {
		delete(visitors, ip)
	}
}
