// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"

	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/rpc/eth"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10000 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump(funcMap map[string]eth.RPCFunc, logger log.TMLogger) {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { _ = c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			logger.Error("websocket client read message error", "err", err)
			websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure)
			break
		}

		logger.Debug("JSON-RPC2 websocket request", "message", string(message))

		outBytes, ethError := handleMessage(message, funcMap, c.conn)

		if ethError != nil {
			resp := eth.JsonRpcErrorResponse{
				Version: "2.0",
				Error:   *ethError,
			}
			outBytes, err = json.MarshalIndent(resp, "", "  ")
			if err != nil {
				continue
			}
		}
		logger.Debug("JSON-RPC2 websocket request", "result", string(outBytes))
		if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
			logger.Error("error %v set write deadline", "err", err)
		}
		if err = c.conn.WriteMessage(websocket.TextMessage, outBytes); err != nil {
			logger.Error("error %v writing to websocket", "err", err)
		}
	}
}
