package loom

import (
	"encoding/json"
	"log"
)

type EventHandler interface {
	PostCommit(state State, txBytes []byte, res TxHandlerResult) error
	EmitBlockTx(height int64) error
}

type EventDispatcher interface {
	Stash(index int64, msg []byte) error
	FetchStash(index int64) ([][]byte, error)
	PurgeStash(index int64) error
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
	return ed.dispatcher.Stash(height, msg)
}

func (ed *DefaultEventHandler) EmitBlockTx(height int64) error {
	msgs, err := ed.dispatcher.FetchStash(height)
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		if err := ed.dispatcher.Send(height, msg); err != nil {
			log.Printf("Error sending event: height: %d; msg: %+v\n", height, msg)
		}
	}
	if err := ed.dispatcher.PurgeStash(height); err != nil {
		log.Printf("Error purging stash for height %d: %+v", height, err)
		return err
	}
	return nil
}
