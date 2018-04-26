package loom

import (
	"encoding/json"
)

type EventHandler interface {
	PostCommit(state State, txBytes []byte, res TxHandlerResult) error
}

type EventDispatcher interface {
	Send(index int64, msg []byte) error
}

type DefaultEventHandler struct {
	dispatcher EventDispatcher
}

func NewDefaultEventHandler(dispatcher EventDispatcher) *DefaultEventHandler {
	return &DefaultEventHandler{
		dispatcher: dispatcher,
	}
}

func (ed *DefaultEventHandler) PostCommit(state State, txBytes []byte, res TxHandlerResult) error {
	queueStruct := struct {
		Event  string
		Tx     []byte
		Result TxHandlerResult
	}{"postcommit", txBytes, res}
	height := state.Block().Height
	msg, err := json.Marshal(queueStruct)
	if err != nil {
		return err
	}
	return ed.dispatcher.Send(height, msg)
}
