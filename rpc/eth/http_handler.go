package eth

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/gorilla/websocket"
)

type HttpRPCFunc struct {
	method    reflect.Value
	signature []reflect.Type
}

func NewRPCFunc(method interface{}, paramNamesString string) RPCFunc {
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
	for p := 0; p < rMethod.NumIn(); p++ {
		signature = append(signature, rMethod.In(p))
	}

	return &HttpRPCFunc{
		method:    reflect.ValueOf(method),
		signature: signature,
	}
}

func (m *HttpRPCFunc) getInputValues(input JsonRpcRequest) (resp []reflect.Value, jsonErr *Error) {
	paramsBytes := []json.RawMessage{}
	if len(input.Params) > 0 {
		if err := json.Unmarshal(input.Params, &paramsBytes); err != nil {
			return resp, NewError(EcParseError, "Parse params", err.Error())
		}
	}
	if len(paramsBytes) > len(m.signature) {
		return resp, NewErrorf(EcInvalidParams, "Parse params", "excess input arguments, expected %v got %v", len(m.signature), len(paramsBytes))
	}

	var inValues []reflect.Value
	for i := 0; i < len(m.signature); i++ {
		paramValue := reflect.New(m.signature[i])
		if i < len(paramsBytes) {
			if err := json.Unmarshal(paramsBytes[i], paramValue.Interface()); err != nil {
				return resp, NewErrorf(EcParseError, "Parse params", "unmarshal input parameter position %v", i)
			}
		}
		inValues = append(inValues, paramValue.Elem())
	}
	return inValues, nil
}

func (m *HttpRPCFunc) GetResponse(result json.RawMessage, ID *json.RawMessage) (*JsonRpcResponse, *Error) {
	return &JsonRpcResponse{
		Result:  result,
		Version: "2.0",
		ID:      ID,
	}, nil
}

func (m *HttpRPCFunc) UnmarshalParamsAndCall(input JsonRpcRequest, _ *websocket.Conn) (resp json.RawMessage, jsonErr *Error) {
	inValues, jsonErr := m.getInputValues(input)
	if jsonErr != nil {
		return resp, jsonErr
	}
	return m.call(inValues)
}

func (m *HttpRPCFunc) call(inValues []reflect.Value) (resp json.RawMessage, jsonErr *Error) {
	outValues := m.method.Call(inValues)

	if outValues[1].Interface() != nil {
		return resp, NewErrorf(EcServer, "Server error", "loom error: %v", outValues[1].Interface())
	}

	value := outValues[0].Interface()
	outBytes, err := json.Marshal(value)
	if err != nil {
		return resp, NewErrorf(EcServer, "Parse response", "json marshall return value %v", value)
	}
	return json.RawMessage(outBytes), nil
}
