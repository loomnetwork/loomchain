package loomchain

import (
	"fmt"
	"log"
	"strings"

	"github.com/loomnetwork/loomchain/events"
)

type EventHandler interface {
	Post(state State, txBytes []byte) error
	EmitBlockTx(height int64) error
}

type EventDispatcher interface {
	Send(index int64, msg []byte) error
}

type DefaultEventHandler struct {
	dispatcher EventDispatcher
	stash      *stash
}

func NewDefaultEventHandler(dispatcher EventDispatcher) *DefaultEventHandler {
	return &DefaultEventHandler{
		dispatcher: dispatcher,
		stash:      newStash(),
	}
}

func (ed *DefaultEventHandler) Post(state State, msg []byte) error {
	height := state.Block().Height
	log.Printf("Stashing event for height=%d, msg=%s", height, msg)
	ed.stash.add(height, msg)
	return nil
}

func (ed *DefaultEventHandler) EmitBlockTx(height int64) error {
	log.Printf("Emitting stashed events for height=%d", height)
	msgs, err := ed.stash.fetch(height)
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		if err := ed.dispatcher.Send(height, msg); err != nil {
			log.Printf("Error sending event: height: %d; msg: %+v\n", height, msg)
		}
	}
	ed.stash.purge(height)
	return nil
}

// byteString set implementation
var exists = struct{}{}

type byteStringSet struct {
	m map[string]struct{}
}

func newByteStringSet() *byteStringSet {
	s := &byteStringSet{}
	s.m = make(map[string]struct{})
	return s
}

func (s *byteStringSet) Add(value []byte) {
	s.m[string(value)] = exists
}

func (s *byteStringSet) Values() [][]byte {
	keys := [][]byte{}
	for k := range s.m {
		keys = append(keys, []byte(k))
	}
	return keys
}

////////

// stash is a map of height -> byteStringSet
type stash struct {
	m map[int64]*byteStringSet
}

func newStash() *stash {
	return &stash{
		m: make(map[int64]*byteStringSet),
	}
}

func (s *stash) add(height int64, msg []byte) {
	_, ok := s.m[height]
	if !ok {
		s.m[height] = newByteStringSet()
	}
	s.m[height].Add(msg)
}

func (s *stash) fetch(height int64) ([][]byte, error) {
	set, ok := s.m[height]
	if !ok {
		return nil, fmt.Errorf("stash does not exist")
	}
	return set.Values(), nil
}

func (s *stash) purge(height int64) {
	delete(s.m, height)
}

func NewEventDispatcher(uri string) (EventDispatcher, error) {
	if strings.HasPrefix(uri, "redis") {
		return events.NewRedisEventDispatcher(uri)
	}
	return nil, fmt.Errorf("Cannot handle event dispatcher uri %s", uri)
}
