package rpc

import (
	"net/http"
)

type responseWriterWithStatus struct {
	http.ResponseWriter
	statusCode int
	lastWrite  []byte
}

func NewResponseWriterWithStatus(w http.ResponseWriter) *responseWriterWithStatus {
	return &responseWriterWithStatus{w, http.StatusOK, []byte{}}
}

func (rwws *responseWriterWithStatus) WriteHeader(code int) {
	rwws.statusCode = code
	rwws.ResponseWriter.WriteHeader(code)
}

func (rwws *responseWriterWithStatus) Write(data []byte) (int, error) {
	rwws.lastWrite = data
	return rwws.ResponseWriter.Write(data)
}
