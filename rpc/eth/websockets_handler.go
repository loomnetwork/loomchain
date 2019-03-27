package eth

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/loomnetwork/loomchain/log"
)

const (
	ReadBufferSize  = 1024
	WriteBufferSize = 1024
)

type WsJsonRpcResponse struct {
	Result  json.RawMessage `json:"result"`
	Version string          `json:"jsonrpc"`
	Id      int64           `json:"id"`
}

type WSPRCFunc struct {
	HttpRPCFunc
	upgrader websocket.Upgrader
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
		upgrader: websocket.Upgrader{
			ReadBufferSize:  ReadBufferSize,
			WriteBufferSize: WriteBufferSize,
		},
	}
}

func (w WSPRCFunc) unmarshalParamsAndCall(input JsonRpcRequest, writer http.ResponseWriter, reader *http.Request, conn *websocket.Conn) (resp *JsonRpcResponse, jsonErr *Error) {
	inValues, jsonErr := w.getInputValues(input)
	if jsonErr != nil {
		return resp, jsonErr
	}
	if conn == nil {
		var err error
		conn, err = w.upgrader.Upgrade(writer, reader, nil)
		if err != nil {
			return resp, NewErrorf(EcServer, "Upgraded connection", "error upgrading to websocket connection %v", err)
		}
	}

	inValues = append([]reflect.Value{reflect.ValueOf(*conn)}, inValues...)
	result, jsonErr := w.call(inValues, input.ID)
	if jsonErr != nil {
		return resp, jsonErr
	}

	wsResp := WsJsonRpcResponse{
		Result:  result,
		Version: "2.0",
		Id:      input.ID,
	}
	jsonBytes, err := json.MarshalIndent(wsResp, "", "  ")
	if err != nil {
		log.Error("error %v marshalling response %v", err, result)
	}
	if err := conn.WriteMessage(websocket.TextMessage, jsonBytes); err != nil {
		log.Error("error %v writing response %v to websocket, id %v", err, jsonBytes, input.ID)
	}

	return nil, nil
}
