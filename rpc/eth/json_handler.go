package eth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/loomnetwork/loomchain/log"
)

type JsonRpcRequest struct {
	Version string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      int64           `json:"id"`
}

type JsonRpcResponse struct {
	Result  json.RawMessage `json:"result"`
	Version string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
}

type JsonRpcErrorResponse struct {
	Version string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Error   Error  `json:"error"`
}

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

type RPCFunc interface {
	unmarshalParamsAndCall(JsonRpcRequest, http.ResponseWriter, *http.Request, *websocket.Conn) (json.RawMessage, *Error)
	getResponse(json.RawMessage, int64, *websocket.Conn, bool) (*JsonRpcResponse, *Error)
}

func RegisterRPCFuncs(mux *http.ServeMux, funcMap map[string]RPCFunc, logger log.TMLogger) {
	mux.HandleFunc("/", func(writer http.ResponseWriter, reader *http.Request) {
		var isWsReq bool
		var conn *websocket.Conn
		body, err := ioutil.ReadAll(reader.Body)
		if err != nil {
			isWsReq = false
			WriteResponse(writer, JsonRpcErrorResponse{
				Version: "2.0",
				Error:   *NewErrorf(EcInternal, "Http error", "error reading message body %v", err),
			})
			return
		}

		if len(body) == 0 {
			var err error
			conn, err = upgrader.Upgrade(writer, reader, nil)
			if err != nil {
				logger.Debug("message with no body received")
				return
			}
			isWsReq = true
			var msgType int
			msgType, body, err = conn.ReadMessage()
			if err != nil {
				logger.Error("%v reading websocket message", err)
				return
			}
			if len(body) == 0 {
				logger.Error("websocket message with no data recived")
				return
			}
			logger.Debug("message type %v, message %v", msgType, body)
		}

		requestList, isBatchRequest, reqListErr := getRequests(body)

		if reqListErr != nil {
			WriteResponse(writer, JsonRpcErrorResponse{
				Version: "2.0",
				Error:   *reqListErr,
			})
			return
		}

		var outputList []interface{}

		for _, jsonRequest := range requestList {
			method, jsonErr := getRequest(jsonRequest, funcMap)
			if jsonErr != nil {
				outputList = append(outputList, JsonRpcErrorResponse{
					Version: "2.0",
					ID:      jsonRequest.ID,
					Error:   *jsonErr,
				})
				continue
			}

			rawResult, jsonErr := method.unmarshalParamsAndCall(jsonRequest, writer, reader, conn)

			if jsonErr != nil {
				outputList = append(outputList, JsonRpcErrorResponse{
					Version: "2.0",
					ID:      jsonRequest.ID,
					Error:   *jsonErr,
				})
				continue
			}

			resp, jsonErr := method.getResponse(rawResult, jsonRequest.ID, conn, isWsReq)
			if jsonErr != nil {
				outputList = append(outputList, JsonRpcErrorResponse{
					Version: "2.0",
					ID:      jsonRequest.ID,
					Error:   *jsonErr,
				})
				continue
			}

			outputList = append(outputList, resp)
		}

		if len(outputList) > 0 && isBatchRequest {
			WriteResponse(writer, outputList)
			return
		}

		if len(outputList) == 1 && !isBatchRequest {
			WriteResponse(writer, outputList[0])
		}
	})
}

func getRequests(message []byte) ([]JsonRpcRequest, bool, *Error) {
	var isBatchRequest bool = true
	var inputList []JsonRpcRequest
	if err := json.Unmarshal(message, &inputList); err != nil {
		var singleInput JsonRpcRequest
		if err := json.Unmarshal(message, &singleInput); err != nil {
			return nil, false, NewErrorf(EcInvalidRequest, "Invalid request", "error  unmarshalling message body %v", err)
		} else {
			isBatchRequest = false
			inputList = append(inputList, singleInput)
		}
	}

	return inputList, isBatchRequest, nil
}

func getRequest(input JsonRpcRequest, funcMap map[string]RPCFunc) (RPCFunc, *Error) {
	method, found := funcMap[input.Method]
	if !found {
		msg := fmt.Sprintf("Method %s not found", input.Method)
		return nil, NewErrorf(EcMethodNotFound, msg, "could not find method %v", input.Method)
	}

	return method, nil
}

func WriteResponse(writer http.ResponseWriter, output interface{}) {
	outBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(outBytes)
}

// Json2 compliant error object
// https://www.jsonrpc.org/specification#error_object
type ErrorCode int

const (
	EcParseError     ErrorCode = -32700 // Invalid JSON was received by the server. An error occurred on the server while parsing the JSON text.
	EcInvalidRequest ErrorCode = -32600 // The JSON sent is not a valid Request object.
	EcMethodNotFound ErrorCode = -32601 // The method does not exist / is not available.
	EcInvalidParams  ErrorCode = -32602 // Invalid method parameter(s).
	EcInternal       ErrorCode = -32603 // Internal JSON-RPC error.
	EcServer         ErrorCode = -32000 // Reserved for implementation-defined server-errors.
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
