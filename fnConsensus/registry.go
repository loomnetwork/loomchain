package fnConsensus

import (
	"errors"
	"sync"
)

var ErrFnIDIsTaken = errors.New("FnID is already used by another Fn Object")
var ErrFnObjCantNil = errors.New("FnObj cant be nil")

// Fn object once registered, will be invoked by Reactor at various point in state cycle
// It should contain pluggable business logic to construct/submit message and signature
// TODO: Eliminate the unused ctx parameter from all these methods.
type Fn interface {
	// Generates a message and associated signature.
	// The reactor will attempt to reach a consensus on the message, which means that a sufficient
	// number of validators must generate exactly the same message, ergo the message must be obtained
	// somewhat deterministically, otherwise consensus will never be reached.
	// The signature will be broadcast to other validators, but doesn't have to match across validators.
	// Once consensus is reached the signature, along with those obtained from the other validators,
	// may be passed to SubmitMultiSignedMessage.
	GetMessageAndSignature(ctx []byte) ([]byte, []byte, error)
	// Associates the given key with a message so it can be looked up in SubmitMultiSignedMessage.
	MapMessage(ctx []byte, key []byte, message []byte) error
	// Once the reactor reaches the vote threshold for the message identified by the given key
	// it invokes this method with the signatures submitted by the validators that pariticipated in the vote.
	SubmitMultiSignedMessage(ctx []byte, key []byte, signatures [][]byte)
}

// FnRegistry acts as a registry which stores multiple Fn objects by their IDs
// And allows reactor to query Fns at time of propose and validation.
type FnRegistry interface {
	Get(fnID string) Fn
	Set(fnID string, fnObj Fn) error
	GetAll() []string
}

// InMemoryFnRegistry is a transient registry that needs to be rebuilt upon restart.
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
