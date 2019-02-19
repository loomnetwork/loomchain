package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	//abci "github.com/tendermint/tendermint/abci/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/rpc/lib/types"
)

const (
	CodeTypeFail = 1 //abci.CodeTypeOK
)

func TestLimitVisits(t *testing.T) {
	handler := limitVisits(newNextHandler(t, CodeTypeFail))
	require.Equal(t, 0, len(visitors))
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	ip := getRealAddr(req)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, 200, w.Result().StatusCode)
	require.Equal(t, 1, len(visitors))
	limiter, err := visitors[ip].limiter.Peek(context.TODO(), keyVisitors)
	require.NoError(t, err)
	require.Equal(t, int64(9), limiter.Remaining)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, 200, w.Result().StatusCode)
	limiter, err = visitors[ip].limiter.Peek(context.TODO(), keyVisitors)
	require.NoError(t, err)
	require.Equal(t, int64(8), limiter.Remaining)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, 200, w.Result().StatusCode)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, 200, w.Result().StatusCode)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, 200, w.Result().StatusCode)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, 200, w.Result().StatusCode)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, 200, w.Result().StatusCode)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, 200, w.Result().StatusCode)
	limiter, err = visitors[ip].limiter.Peek(context.TODO(), keyVisitors)
	require.NoError(t, err)
	require.Equal(t, int64(2), limiter.Remaining)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, 200, w.Result().StatusCode)
	limiter, err = visitors[ip].limiter.Peek(context.TODO(), keyVisitors)
	require.NoError(t, err)
	require.Equal(t, int64(1), limiter.Remaining)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, 200, w.Result().StatusCode)
	limiter, err = visitors[ip].limiter.Peek(context.TODO(), keyVisitors)
	require.NoError(t, err)
	require.Equal(t, int64(0), limiter.Remaining)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, 200, w.Result().StatusCode)
	limiter, err = visitors[ip].limiter.Peek(context.TODO(), keyVisitors)
	require.NoError(t, err)
	require.Equal(t, int64(0), limiter.Remaining)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, 429, w.Result().StatusCode)
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
