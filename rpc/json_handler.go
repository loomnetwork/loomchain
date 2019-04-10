package rpc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"

	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/rpc/eth"
)

func RegisterRPCFuncs(mux *http.ServeMux, funcMap map[string]eth.RPCFunc, logger log.TMLogger, hub *Hub) {
	mux.HandleFunc("/", func(writer http.ResponseWriter, reader *http.Request) {
		if isWebSocketConnection(reader) {
			conn, err := upgrader.Upgrade(writer, reader, nil)
			if err != nil {
				logger.Debug("message with no body received")
				return
			}
			client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
			client.hub.register <- client

			// Allow collection of memory referenced by the caller by doing all work in
			// new goroutines.
			go client.writePump()
			go client.readPump(funcMap)
			return

		}

		body, err := ioutil.ReadAll(reader.Body)
		if err != nil {
			WriteResponse(writer, eth.JsonRpcErrorResponse{
				Version: "2.0",
				Error:   *eth.NewErrorf(eth.EcInternal, "Http error", "error reading message body %v", err),
			})
			return
		}
		logger.Info("JSON-RPC2 http request, input message", string(body))
		outBytes, ethError := handleMessage(body, funcMap, nil)

		if ethError != nil {
			WriteResponse(writer, eth.JsonRpcErrorResponse{
				Version: "2.0",
				Error:   *ethError,
			})
			return
		}


		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		logger.Info("JSON-RPC2 http request, result", string(outBytes))
		_, err = writer.Write(outBytes)
		if err != nil {
			logger.Debug("error writing responce", err)
		}
	})
}

func handleMessage(body []byte, funcMap map[string]eth.RPCFunc, conn *websocket.Conn) ([]byte, *eth.Error) {
	requestList, reqListErr := getRequests(body)

	if reqListErr != nil {
		return nil, reqListErr
	}

	var outputList []interface{}

	for _, jsonRequest := range requestList {
		method, jsonErr := getRequest(jsonRequest, funcMap)
		if jsonErr != nil {
			outputList = append(outputList, eth.JsonRpcErrorResponse{
				Version: "2.0",
				ID:      jsonRequest.ID,
				Error:   *jsonErr,
			})
			continue
		}

		rawResult, jsonErr := method.UnmarshalParamsAndCall(jsonRequest, conn)

		if jsonErr != nil {
			outputList = append(outputList, eth.JsonRpcErrorResponse{
				Version: "2.0",
				ID:      jsonRequest.ID,
				Error:   *jsonErr,
			})
			continue
		}

		resp, jsonErr := method.GetResponse(rawResult, jsonRequest.ID)
		if jsonErr != nil {
			outputList = append(outputList, eth.JsonRpcErrorResponse{
				Version: "2.0",
				ID:      jsonRequest.ID,
				Error:   *jsonErr,
			})
			continue
		}

		outputList = append(outputList, resp)
	}

	outBytes, err := json.MarshalIndent(outputList, "", "  ")
	if err != nil {
		return nil,  eth.NewErrorf(eth.EcServer, "Server error", "error  marshalling result %v", err)
	}
	return outBytes, nil
}

func getRequests(message []byte) ([]eth.JsonRpcRequest, *eth.Error) {
	var inputList []eth.JsonRpcRequest
	if err := json.Unmarshal(message, &inputList); err != nil {
		var singleInput eth.JsonRpcRequest
		if err := json.Unmarshal(message, &singleInput); err != nil {
			return nil, eth.NewErrorf(eth.EcInvalidRequest, "Invalid request", "error  unmarshalling message body %v", err)
		} else {
			inputList = append(inputList, singleInput)
		}
	}
	return inputList, nil
}

func isWebSocketConnection(req *http.Request) bool {
	if strings.ToLower(req.Header.Get(http.CanonicalHeaderKey("Connection"))) != "upgrade" {
		return false
	}

	if strings.ToLower(req.Header.Get(http.CanonicalHeaderKey("Upgrade"))) != "websocket" {
		return false
	}

	if req.Method != "GET" {
		return false
	}
	return true
}

func getRequest(input eth.JsonRpcRequest, funcMap map[string]eth.RPCFunc) (eth.RPCFunc, *eth.Error) {
	method, found := funcMap[input.Method]
	if !found {
		msg := fmt.Sprintf("Method %s not found", input.Method)
		return nil, eth.NewErrorf(eth.EcMethodNotFound, msg, "could not find method %v", input.Method)
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


