package eth

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/gorilla/websocket"
)

type WSPRCFunc struct {
	RPCFunc
	upgrader websocket.Upgrader
}

func NewWSRPCFunc(method interface{}, paramNamesString string) *WSPRCFunc {
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
		RPCFunc: RPCFunc{
			method:    reflect.ValueOf(method),
			signature: signature,
		},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

func (w WSPRCFunc) unmarshalParmsAndCall(input JsonRpcRequest, writer http.ResponseWriter, reader *http.Request) (resp JsonRpcResponse, jsonErr *Error) {
	inValues, jsonErr := w.getInputValues(input)
	if jsonErr != nil {
		return resp, jsonErr
	}

	conn, err := w.upgrader.Upgrade(writer, reader, nil)
	if err != nil {
		return resp, NewErrorf(EcServer, "Upgraded connection", "error upgrading to websocket connection %v", err)
	}
	inValues = append([]reflect.Value{reflect.ValueOf(conn)}, inValues...)

	return w.call(inValues, input.ID)
}
