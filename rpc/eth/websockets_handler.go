package eth

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/gorilla/websocket"
)

const (
	ReadBufferSize  = 1024
	WriteBufferSize = 1024
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  ReadBufferSize,
		WriteBufferSize: WriteBufferSize,
	}
)

type WsJsonRpcResponse struct {
	Result  json.RawMessage `json:"result"`
	Version string          `json:"jsonrpc"`
	Id      int64           `json:"id"`
}

type WSPRCFunc struct {
	HttpRPCFunc
}

func NewWSRPCFunc(method interface{}, paramNamesString string) RPCFunc {
	var paramNames []string
	if len(paramNamesString) > 0 {
		paramNames = strings.Split(paramNamesString, ",")

	} else {
		paramNames = []string{}
	}

	rMethod := reflect.TypeOf(method)
	if len(paramNames) != rMethod.NumIn() {
		panic("parameter count mismatch making loom api method")
	}
	signature := []reflect.Type{}
	// first parameter is a websocket connection
	for p := 1; p < rMethod.NumIn(); p++ {
		signature = append(signature, rMethod.In(p))
	}

	return &WSPRCFunc{
		HttpRPCFunc: HttpRPCFunc{
			method:    reflect.ValueOf(method),
			signature: signature,
		},
	}
}

func (w *WSPRCFunc) GetResponse(result json.RawMessage, id int64) (*JsonRpcResponse, *Error) {
	return getResponse(result, id)
}

func (w *WSPRCFunc) UnmarshalParamsAndCall(input JsonRpcRequest, conn *websocket.Conn) (resp json.RawMessage, jsonErr *Error) {
	inValues, jsonErr := w.getInputValues(input)
	if jsonErr != nil {
		return resp, jsonErr
	}
/*
	w.conn = conn
	if w.conn == nil {
		var err error
		w.conn, err = upgrader.Upgrade(writer, reader, nil)
		if err != nil {
			return resp, NewErrorf(rpc.EcServer, "Upgraded connection", "error upgrading to websocket connection %v", err)
		}
	}
*/
	inValues = append([]reflect.Value{reflect.ValueOf(conn)}, inValues...)
	return w.call(inValues, input.ID)
}
