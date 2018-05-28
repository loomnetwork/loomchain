package loomchain

import (
	"context"
	"testing"

	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/log"
	"github.com/stretchr/testify/require"
	common "github.com/tendermint/tmlibs/common"
)

type appHandler struct {
	t               *testing.T
	CalledProcessTx int
}

var appTag = common.KVPair{Key: []byte("AppKey"), Value: []byte("AppValue")}

func (a *appHandler) ProcessTx(state State, txBytes []byte) (TxHandlerResult, error) {
	a.CalledProcessTx++
	require.Equal(a.t, txBytes, []byte("AppData"))
	return TxHandlerResult{
		Tags: []common.KVPair{appTag},
	}, nil
}

// Test that middleware is applied in the correct order, and that tags are set correctly.
func TestMiddlewareTxHandler(t *testing.T) {
	log.Setup("debug", "")
	allBytes := []byte("FirstMW/SecondMW/AppData")
	mw1Tag := common.KVPair{Key: []byte("MW1Key"), Value: []byte("MW1Value")}
	mw1Func := TxMiddlewareFunc(func(state State, txBytes []byte, next TxHandlerFunc) (TxHandlerResult, error) {
		require.Equal(t, txBytes, allBytes)
		r, err := next(state, txBytes[len("FirstMW/"):])
		if err != nil {
			return r, err
		}
		r.Tags = append(r.Tags, mw1Tag)
		return r, err
	})

	mw2Tag := common.KVPair{Key: []byte("MW2Key"), Value: []byte("MW2Value")}
	mw2Func := TxMiddlewareFunc(func(state State, txBytes []byte, next TxHandlerFunc) (TxHandlerResult, error) {
		require.Equal(t, txBytes, []byte("SecondMW/AppData"))
		r, err := next(state, txBytes[len("SecondMW/"):])
		if err != nil {
			return r, err
		}
		r.Tags = append(r.Tags, mw2Tag)
		return r, err
	})

	handler := appHandler{t: t}
	mwHandler := MiddlewareTxHandler(
		[]TxMiddleware{
			mw1Func,
			mw2Func,
		},
		&handler,
		[]PostCommitMiddleware{},
	)
	deliverTxState := StoreState{
		ctx: context.Background(),
		block: types.BlockHeader{
			Height: 1,
		},
	}

	r, _ := mwHandler.ProcessTx(&deliverTxState, allBytes)
	require.Equal(t, []common.KVPair{appTag, mw2Tag, mw1Tag}, r.Tags)

	checkTxState := StoreState{
		ctx: context.WithValue(context.Background(), "checkTx", true),
		block: types.BlockHeader{
			Height: 1,
		},
	}

	mwHandler.ProcessTx(&checkTxState, allBytes)
	// appHandler ProcessTx should be called only once
	require.Equal(t, 1, handler.CalledProcessTx, "contract handler called for checkTX")
}
