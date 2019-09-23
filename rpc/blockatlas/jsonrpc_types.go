package blockatlas

import (
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

type RPCFunc interface {
	UnmarshalParamsAndCall(JsonRpcRequest, *websocket.Conn) (json.RawMessage, *Error)
	GetResponse(result json.RawMessage, ID *json.RawMessage) (*JsonRpcResponse, *Error)
}

type JsonRpcRequest struct {
	Version string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	// The request ID can be a string, number, or may be missing entirely.
	// We just pass it through as is in the response/error without unmarshalling.
	// Related: https://github.com/ethereum/go-ethereum/issues/295
	ID *json.RawMessage `json:"id"`
}

type JsonRpcResponse struct {
	Result  json.RawMessage  `json:"result"`
	Version string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
}

type JsonRpcErrorResponse struct {
	Version string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Error   Error            `json:"error"`
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
