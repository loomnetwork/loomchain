package rpc

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
)

const (
	statusOk = 200
)

type JsonRpcRequest struct {
	JsonRpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  []byte `json:"params"`
	Id      int64  `json:"id"`
}

type JsonRpcResponse struct {
	Result  json.RawMessage `json:"result"`
	JsonRpc string          `json:"jsonrpc"`
	Id      int64           `json:"id"`
}

type JsonRpcRequestIdString struct {
	JsonRpc string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	Id      string          `json:"id"`
}

type JsonRpcResponseIdString struct {
	Result  json.RawMessage `json:"result"`
	JsonRpc string          `json:"jsonrpc"`
	Id      string          `json:"id"`
}

type LoomApiMethod struct {
	method     reflect.Value
	signature  map[string]reflect.Type
	returnType reflect.Type
	idString   bool
}

func newLoomApiMethod(method interface{}, paramNamesString string, idString bool) *LoomApiMethod {
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
	signature := make(map[string]reflect.Type)
	for p := 0; p < rMethod.NumIn(); p++ {
		signature[paramNames[p]] = rMethod.In(p)
	}

	return &LoomApiMethod{
		method:    reflect.ValueOf(method),
		signature: signature,
		idString:  idString,
	}
}

func (m LoomApiMethod) call(input JsonRpcRequest) (JsonRpcResponse, error) {
	paramsBytes := make(map[string]json.RawMessage)
	if len(input.Params) > 0 {
		if err := json.Unmarshal(input.Params, &paramsBytes); err != nil {
			return JsonRpcResponse{}, errors.Wrap(err, "unexpected JSON type, expected map")
		}
	}

	var inValues []reflect.Value
	for name, paramType := range m.signature {
		paramBytes, found := paramsBytes[name]
		paramValue := reflect.New(paramType)
		if !found || len(paramBytes) == 0 {
			paramValue = reflect.Zero(paramType)
		} else {
			if err := json.Unmarshal(paramBytes, paramValue.Interface()); err != nil {
				return JsonRpcResponse{}, errors.Wrapf(err, "unmarshal input parameter %s", name)
			}
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

		// Handle both cases; where id is a string and an integer.
		var input JsonRpcRequest
		if err := json.Unmarshal(body, &input); err != nil {
			inputIdString := JsonRpcRequestIdString{}
			if err := json.Unmarshal(body, &inputIdString); err != nil {
				return
			}
			input.Params = inputIdString.Params
			input.JsonRpc = inputIdString.JsonRpc
			input.Method = inputIdString.Method
			id, err := strconv.ParseInt(inputIdString.Id, 10, 64)
			if err != nil {
				return
			}
			input.Id = id
		}
		method, found := funcMap[input.Method]
		if !found {
			return
		}
		output, err := method.call(input)

		var outBytes []byte
		if method.idString {
			outputIdString := JsonRpcResponseIdString{
				Result:  output.Result,
				JsonRpc: output.JsonRpc,
				Id:      strconv.FormatInt(output.Id, 10),
			}
			outBytes, err = json.MarshalIndent(outputIdString, "", "  ")
		} else {
			outBytes, err = json.MarshalIndent(output, "", "  ")
		}
		if err != nil {
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(statusOk)
		writer.Write(outBytes)
	})
}
