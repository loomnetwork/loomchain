package loom

import (
	"encoding/json"
	"net/http"
)

// From tendermint/tmlibs/common/http.go (which doesn't seem to exist anymore)

type ErrorResponse struct {
	Success bool `json:"success,omitempty"`

	// Err is the error message if Success is false
	Err string `json:"error,omitempty"`

	// Code is set if Success is false
	Code int `json:"code,omitempty"`
}

// ErrorWithCode makes an ErrorResponse with the
// provided err's Error() content, and status code.
// It panics if err is nil.
func ErrorWithCode(err error, code int) *ErrorResponse {
	return &ErrorResponse{
		Err:  err.Error(),
		Code: code,
	}
}

// Ensure that ErrorResponse implements error
var _ error = (*ErrorResponse)(nil)

func (er *ErrorResponse) Error() string {
	return er.Err
}

// WriteSuccess JSON marshals the content provided, to an HTTP
// response, setting the provided status code and setting header
// "Content-Type" to "application/json".
func WriteSuccess(w http.ResponseWriter, data interface{}) {
	WriteCode(w, data, 200)
}

// WriteCode JSON marshals content, to an HTTP response,
// setting the provided status code, and setting header
// "Content-Type" to "application/json". If JSON marshalling fails
// with an error, WriteCode instead writes out the error invoking
// WriteError.
func WriteCode(w http.ResponseWriter, out interface{}, code int) {
	blob, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		WriteError(w, err)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		w.Write(blob)
	}
}

// WriteError is a convenience function to write out an
// error to an http.ResponseWriter, to send out an error
// that's structured as JSON i.e the form
//    {"error": sss, "code": ddd}
// If err implements the interface HTTPCode() int,
// it will use that status code otherwise, it will
// set code to be http.StatusBadRequest
func WriteError(w http.ResponseWriter, err error) {
	code := http.StatusBadRequest
	WriteCode(w, ErrorWithCode(err, code), code)
}
