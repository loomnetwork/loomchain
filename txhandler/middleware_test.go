package txhandler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/common"

	appstate "github.com/loomnetwork/loomchain/state"
)

type appHandler struct {
	t *testing.T
}

var appTag = common.KVPair{Key: []byte("AppKey"), Value: []byte("AppValue")}

func (a *appHandler) ProcessTx(state appstate.State, txBytes []byte, isCheckTx bool) (TxHandlerResult, error) {
	require.Equal(a.t, txBytes, []byte("AppData"))
	return TxHandlerResult{
		Tags: []common.KVPair{appTag},
	}, nil
}

// Test that middleware is applied in the correct order, and that tags are set correctly.
func TestMiddlewareTxHandler(t *testing.T) {
	allBytes := []byte("FirstMW/SecondMW/AppData")
	mw1Tag := common.KVPair{Key: []byte("MW1Key"), Value: []byte("MW1Value")}
	mw1Func := TxMiddlewareFunc(
		func(state appstate.State, txBytes []byte, next TxHandlerFunc, isCheckTx bool) (TxHandlerResult, error) {
			require.Equal(t, txBytes, allBytes)
			r, err := next(state, txBytes[len("FirstMW/"):], false)
			if err != nil {
				return r, err
			}
			r.Tags = append(r.Tags, mw1Tag)
			return r, err
		},
	)

	mw2Tag := common.KVPair{Key: []byte("MW2Key"), Value: []byte("MW2Value")}
	mw2Func := TxMiddlewareFunc(
		func(state appstate.State, txBytes []byte, next TxHandlerFunc, isCheckTx bool) (TxHandlerResult, error) {
			require.Equal(t, txBytes, []byte("SecondMW/AppData"))
			r, err := next(state, txBytes[len("SecondMW/"):], false)
			if err != nil {
				return r, err
			}
			r.Tags = append(r.Tags, mw2Tag)
			return r, err
		},
	)

	mwHandler := MiddlewareTxHandler(
		[]TxMiddleware{
			mw1Func,
			mw2Func,
		},
		&appHandler{t: t},
		[]PostCommitMiddleware{},
	)
	r, _ := mwHandler.ProcessTx(nil, allBytes, false)
	require.Equal(t, r.Tags, []common.KVPair{appTag, mw2Tag, mw1Tag})
}
