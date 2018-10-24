package rpc

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"

	"github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
)

const (
	statusOk = 200
)

type JsonRpcRequest struct {
	JsonRpc string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	Id      int64           `json:"id"`
}

type JsonRpcResponse struct {
	Result  json.RawMessage `json:"result"`
	JsonRpc string          `json:"jsonrpc"`
	Id      int64           `json:"id"`
}

type LoomApiMethod struct {
	method    reflect.Value
	signature []reflect.Type
}

func newLoomApiMethod(method interface{}, paramNamesString string) *LoomApiMethod {
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
	kind := rMethod.Out(0).Kind()
	strT := rMethod.Out(0).String()
	name := rMethod.Out(0).Name()
	kind = kind
	strT = strT
	name = name
	return &LoomApiMethod{
		method:    reflect.ValueOf(method),
		signature: signature,
	}
}

func (m LoomApiMethod) call(input JsonRpcRequest) (JsonRpcResponse, error) {
	//paramsBytes := make(map[string]json.RawMessage)
	// All json parameters are arrays. Add object handling for more general support
	paramsBytes := []json.RawMessage{}
	if len(input.Params) > 0 {
		if err := json.Unmarshal(input.Params, &paramsBytes); err != nil {
			return JsonRpcResponse{}, errors.Wrap(err, "unexpected JSON type, expected map")
		}
	}
	if len(paramsBytes) != len(m.signature) {
		return JsonRpcResponse{}, errors.Errorf("argument count mismatch, expected %v got %v", len(m.signature), len(paramsBytes))
	}

	var inValues []reflect.Value
	for i := 0; i < len(m.signature); i++ {
		paramValue := reflect.New(m.signature[i])
		if err := json.Unmarshal(paramsBytes[i], paramValue.Interface()); err != nil {
			return JsonRpcResponse{}, errors.Wrapf(err, "unmarshal input parameter position %v", i)
		}
		inValues = append(inValues, paramValue.Elem())
	}

	outValues := m.method.Call(inValues)

	if outValues[1].Interface() != nil {
		return JsonRpcResponse{}, errors.Errorf("%v", outValues[1].Interface())
	}

	value := outValues[0].Interface()
	outBytes, err := json.Marshal(value)
	if err != nil {
		return JsonRpcResponse{}, errors.Wrap(err, "json marshall return value")
	}

	return JsonRpcResponse{
		Result:  json.RawMessage(outBytes),
		JsonRpc: "2.0",
		Id:      input.Id,
	}, nil
}

func RegisterJsonFunc(mux *http.ServeMux, funcMap map[string]*LoomApiMethod, logger log.TMLogger) {
	mux.HandleFunc("/", func(writer http.ResponseWriter, reader *http.Request) {
		body, err := ioutil.ReadAll(reader.Body)
		if err != nil {
			return
		}

		var input JsonRpcRequest
		if err := json.Unmarshal(body, &input); err != nil {
			return
		}
		method, found := funcMap[input.Method]
		if !found {
			return
		}

		output, err := method.call(input)

		outBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(statusOk)
		writer.Write(outBytes)
	})
}
