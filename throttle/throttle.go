package throttle

import (
	"github.com/loomnetwork/loomchain/store"
	"sync"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/util"
	"time"
	"encoding/binary"
	"bytes"
)

type Throttle struct {
	maxAccessCount int16
	sessionSize int64
	Store *store.MemStore
	origin loom.Address
}

var (
	instance *Throttle
	once sync.Once
)

func GetThrottle(origin loom.Address) *Throttle {
	once.Do(func() {
		instance = &Throttle{
			maxAccessCount:  100,
			sessionSize:     600,
			Store: store.NewMemStore(),
		}
	})
	instance.origin = origin
	return instance
}

func (t *Throttle) getKeyWithPrefix(prefix string) []byte {
	return util.PrefixKey([]byte(prefix) , []byte(t.origin.String()))
}

func (t *Throttle) getStartTimeKey() []byte {
	return t.getKeyWithPrefix("session-start-time-")
}

func (t *Throttle)  getAccessCountKey() []byte {
	return t.getKeyWithPrefix("session-access-count-")
}

func (t *Throttle) getCurrentTimeInBytes() []byte {
	sessionTime := time.Now().Unix()

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(sessionTime))

	return b
}

func (t *Throttle) initSession() (int64) {
	t.Store.Set(t.getStartTimeKey(), t.getCurrentTimeInBytes())
	return int64(binary.BigEndian.Uint64(t.Store.Get(t.getStartTimeKey())))
}

func (t *Throttle) hasSession() (bool) {
	return t.Store.Has(t.getStartTimeKey())
}

func (t *Throttle)  getStoredStartTime() (int64) {
	value := t.Store.Get(t.getStartTimeKey())
	return int64(binary.BigEndian.Uint64(value))
}

func (t *Throttle)  isSessionExpired() (bool) {
	currentTime := time.Now().Unix()
	sessionStartTime := t.getSessionStartTime()
	return sessionStartTime + t.sessionSize <= currentTime
}

func (t *Throttle) getAccessCount() (int16) {
	return int16(binary.BigEndian.Uint16(t.Store.Get(t.getAccessCountKey())))
}

func (t *Throttle) setAccessCount(accessCount int16) {

	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, accessCount)
	if err != nil {
		panic(err)
	}

	t.Store.Set(t.getAccessCountKey(), buf.Bytes())
}

func (t *Throttle) incrementAccessCount() int16 {
	accessCount := t.getAccessCount()
	accessCount += 1
	t.setAccessCount(accessCount)
	return accessCount
}

func (t *Throttle) getSessionStartTime() int64 {
	var sessionStartTime int64
	if t.hasSession() {
		sessionStartTime = t.getStoredStartTime()
	}else{
		sessionStartTime = t.initSession()
		t.setAccessCount(0)
	}
	return sessionStartTime
}

