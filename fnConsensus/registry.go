package fnConsensus

import (
	"errors"
	"sync"
)

var ErrFnIDIsTaken = errors.New("FnID is already used by another Fn Object")
var ErrFnObjCantNil = errors.New("FnObj cant be nil")

// Fn object once registered, will be invoked by Reactor at various point in state cycle
// It should contain pluggable business logic to construct/submit message and signature
type Fn interface {
	SubmitMultiSignedMessage(ctx []byte, message []byte, signatures [][]byte)
	GetMessageAndSignature(ctx []byte) ([]byte, []byte, error)
	MapMessage(ctx []byte, key []byte, message []byte) error
}

// FnRegistry acts as a registry which stores multiple Fn objects by their IDs
// And allows reactor to query Fns at time of propose and validation.
type FnRegistry interface {
	Get(fnID string) Fn
	Set(fnID string, fnObj Fn) error
	GetAll() []string
}

// Transient registry, need to be rebuilt upon restart
type InMemoryFnRegistry struct {
	mtx   sync.RWMutex
	fnMap map[string]Fn
}

func NewInMemoryFnRegistry() *InMemoryFnRegistry {
	return &InMemoryFnRegistry{
		fnMap: make(map[string]Fn),
	}
}

func (f *InMemoryFnRegistry) GetAll() []string {
	fnIDs := make([]string, len(f.fnMap))
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	i := 0
	for fnID := range f.fnMap {
		fnIDs[i] = fnID
		i++
	}

	return fnIDs
}

func (f *InMemoryFnRegistry) Get(fnID string) Fn {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.fnMap[fnID]
}

func (f *InMemoryFnRegistry) Set(fnID string, fnObj Fn) error {
	if fnObj == nil {
		return ErrFnObjCantNil
	}

	f.mtx.Lock()
	defer f.mtx.Unlock()

	_, exists := f.fnMap[fnID]
	if exists {
		return ErrFnIDIsTaken
	}

	f.fnMap[fnID] = fnObj
	return nil
}
