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

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
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

		outBytes, ethError := handleMessage(message, funcMap, c.conn)

		if ethError != nil {
			logger.Error("error handling message", "err", ethError.Error())
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
			logger.Error("Websocket write panicked", "err", r)
		}
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					logger.Error("error writing close message to websocket", "err", err)
				}
				return
			}

			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				logger.Error("error setting write deadline", "err", err)
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Error("error writing message to websocket", "err", err)
				return
			}

			n := len(c.send)
			for i := 0; i < n; i++ {
				msg := <-c.send
				if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					logger.Error("error writing message to websocket", "err", err)
					return
				}
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
