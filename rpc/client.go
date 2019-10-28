// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/json"
	"io"
	"net/http"
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

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 16384
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

// readPump pumps messages from the websocket connection.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump(funcMap map[string]eth.RPCFunc, logger log.TMLogger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("WebSocket read panicked", "err", r)
		}
		c.hub.unregister <- c
		if err := c.conn.Close(); err != nil {
			logger.Error("Failed to close WebSocket (read pump)", "err", err)
		}
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { _ = c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
				websocket.CloseNoStatusReceived,
			) {
				logger.Error("Failed to read from closed WebSocket", "err", err)
			} else {
				logger.Debug("Failed to read WebSocket", "err", err)
			}
			return
		}

		outBytes, ethError := handleMessage(message, funcMap, c.conn)

		if ethError != nil {
			logger.Error("Failed to handle WebSocket message (read pump)", "err", ethError.Error())
			resp := eth.JsonRpcErrorResponse{
				Version: "2.0",
				Error:   *ethError,
			}
			outBytes, err = json.MarshalIndent(resp, "", "  ")
			if err != nil {
				continue
			}
		}

		c.send <- outBytes
	}
}

// writePump pumps messages to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump(logger log.TMLogger) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		if r := recover(); r != nil {
			logger.Error("WebSocket write panicked", "err", r)
		}
		ticker.Stop()
		if err := c.conn.Close(); err != nil {
			logger.Error("Failed to close WebSocket (write pump)", "err", err)
		}

	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					if err != websocket.ErrCloseSent {
						logger.Error("Failed to write close message to WebSocket", "err", err)
					}
				}
				return
			}
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				logger.Error("Failed to set write deadline on WebSocket", "err", err)
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				if err != io.ErrClosedPipe {
					logger.Error("Failed to write message to WebSocket", "err", err)
				} else {
					logger.Debug("Failed to write message to closed WebSocket", "err", err)
				}
				return
			}

			n := len(c.send)
			for i := 0; i < n; i++ {
				msg := <-c.send
				if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					if err != io.ErrClosedPipe {
						logger.Error("Failed to write message to WebSocket", "err", err)
					} else {
						logger.Debug("Failed to write message to closed WebSocket", "err", err)
					}
					return
				}
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Error("Failed to write ping message to WebSocket", "err", err)
				return
			}
		}
	}
}
