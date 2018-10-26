package rpc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"

	"github.com/loomnetwork/loomchain/log"
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

type JsonRpcErrorResponse struct {
	JsonRpc string `json:"jsonrpc"`
	Id      int64  `json:"id"`
	Error   Error  `jsont:"error"`
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

	return &LoomApiMethod{
		method:    reflect.ValueOf(method),
		signature: signature,
	}
}

func (m LoomApiMethod) call(input JsonRpcRequest) (resp JsonRpcResponse, jsonErr *Error) {
	// All eth json rpc parameters are arrays. Add object handling for more general support
	paramsBytes := []json.RawMessage{}
	if len(input.Params) > 0 {
		if err := json.Unmarshal(input.Params, &paramsBytes); err != nil {
			return resp, NewError(EcParseError, "Parse params", err.Error())
		}
	}
	if len(paramsBytes) != len(m.signature) {
		return resp, NewErrorf(EcInvalidParams, "Parse params", "argument count mismatch, expected %v got %v", len(m.signature), len(paramsBytes))
	}

	var inValues []reflect.Value
	for i := 0; i < len(m.signature); i++ {
		paramValue := reflect.New(m.signature[i])
		if err := json.Unmarshal(paramsBytes[i], paramValue.Interface()); err != nil {
			return resp, NewErrorf(EcParseError, "Parse params", "unmarshal input parameter position %v", i)
		}
		inValues = append(inValues, paramValue.Elem())
	}

	outValues := m.method.Call(inValues)

	if outValues[1].Interface() != nil {
		return resp, NewErrorf(EcServer, "Server error", "%v", outValues[1].Interface())
	}

	value := outValues[0].Interface()
	outBytes, err := json.Marshal(value)
	if err != nil {
		return resp, NewErrorf(EcServer, "Parse response", "json marshall return value %v", value)
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
			WriteResponse(writer, JsonRpcErrorResponse{
				JsonRpc: "2.0",
				Error:   *NewError(EcInternal, "Http error", "error reading message body"),
			})
			return
		}

		//todo write list of endpints if len(body) == 0??????

		var input JsonRpcRequest
		if err := json.Unmarshal(body, &input); err != nil {
			WriteResponse(writer, JsonRpcErrorResponse{
				JsonRpc: "2.0",
				Error:   *NewErrorf(EcInvalidRequerst, "Invalied request", "error unmasrhalling message body %v", body),
			})
			return
		}

		if input.Id == 0 {
			logger.Debug("Http notification recieved (id=0). Ignoring")
			return
		}

		method, found := funcMap[input.Method]
		if !found {
			logger.Debug("Http notification recieved (id=0). Ignoring")
			WriteResponse(writer, JsonRpcErrorResponse{
				JsonRpc: "2.0",
				Id:      input.Id,
				Error:   *NewErrorf(EcMethodNotFound, "Method not found", "could not find method %v", input.Method),
			})
			return
		}

		output, jsonErr := method.call(input)

		if jsonErr != nil {
			WriteResponse(writer, JsonRpcErrorResponse{
				JsonRpc: "2.0",
				Id:      input.Id,
				Error:   *jsonErr,
			})
		} else {
			WriteResponse(writer, output)
		}
	})
}

func WriteResponse(writer http.ResponseWriter, output interface{}) {
	outBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusOk)
	writer.Write(outBytes)
}

// Json2 compliant error object
// https://www.jsonrpc.org/specification#error_object
type ErrorCode int

const (
	EcParseError      ErrorCode = -32700 // Invalid JSON was received by the server. An error occurred on the server while parsing the JSON text.
	EcInvalidRequerst ErrorCode = -32600 // The JSON sent is not a valid Request object.
	EcMethodNotFound  ErrorCode = -32601 // The method does not exist / is not available.
	EcInvalidParams   ErrorCode = -32602 // Invalid method parameter(s).
	EcInternal        ErrorCode = -32603 // Internal JSON-RPC error.
	EcServer          ErrorCode = -32000 // Reserved for implementation-defined server-errors.
)

type Error struct {
	Code    ErrorCode   `json:"code"`    // A Number that indicates the error type that occurred.
	Message string      `json:"message"` // A String providing a short description of the error. The message SHOULD be limited to a concise single sentence.
	Data    interface{} `json:"data"`    // A Primitive or Structured value that contains additional information about the error.
}

func NewError(code ErrorCode, message, data string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

func NewErrorf(code ErrorCode, message, format string, args ...interface{}) *Error {
	return NewError(code, message, fmt.Sprintf(format, args...))
}

func (e *Error) Error() string {
	return e.Message
}
