package throttle

import (
	"errors"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/util"
	"time"
	"encoding/binary"
	"github.com/loomnetwork/loomchain"
	"bytes"
	"log"
	"fmt"
	"github.com/loomnetwork/loomchain/auth"
)

var cfg = Singletone()

func getSessionKeyWithPrefix(prefix string,origin loom.Address) []byte {
	return util.PrefixKey([]byte(prefix) , []byte(origin.String()))
}

func getSessionStartTimeKey(origin loom.Address) []byte {
	return getSessionKeyWithPrefix("session-start-time-", origin)
}

func getSessionAccessCountKey(origin loom.Address) []byte {
	return getSessionKeyWithPrefix("session-access-count-", origin)
}

func startSessionTimeInBytes() []byte {
	sessionTime := time.Now().Unix()

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(sessionTime))

	return b
}

func startSessionTime(state loomchain.State, origin loom.Address) (int64) {
	cfg.Store.Set(getSessionStartTimeKey(origin), startSessionTimeInBytes())
	return int64(binary.BigEndian.Uint64(cfg.Store.Get(getSessionStartTimeKey(origin))))
}

func getSessionTime(state loomchain.State, origin loom.Address) (int64) {
	value := cfg.Store.Get(getSessionStartTimeKey(origin))
	return int64(binary.BigEndian.Uint64(value))
}

func isSessionExpired(sessionStartTime, currentTime, sessionSize int64) (bool) {
	return sessionStartTime + sessionSize <= currentTime
}

func setSessionAccessCount(state loomchain.State, accessCount int16, origin loom.Address) {

	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, accessCount)
	if err != nil {
		panic(err)
	}

	cfg.Store.Set(getSessionAccessCountKey(origin), buf.Bytes())
}

func getSessionAccessCount(state loomchain.State, origin loom.Address) (int16) {
	return int16(binary.BigEndian.Uint16(cfg.Store.Get(getSessionAccessCountKey(origin))))
}

var ThrottleTxMiddleware = loomchain.TxMiddlewareFunc(func(
	state loomchain.State,
	txBytes []byte,
	next loomchain.TxHandlerFunc,
) (res loomchain.TxHandlerResult, err error)  {

	var maxAccessCount = cfg.ThrottleMaxAccessCount
	var sessionSize = cfg.ThrottleSessionSize

	origin := auth.Origin(state.Context())
	if origin.IsEmpty() {
		return res, errors.New("transaction has no origin")
	}

	currentTime := time.Now().Unix()

	var accessCount int16
	var sessionStartTime int64

	if cfg.Store.Has(getSessionStartTimeKey(origin)) {
		sessionStartTime = getSessionTime(state, origin)
	}else{
		sessionStartTime = startSessionTime(state, origin)
		setSessionAccessCount(state, 0, origin)
	}
	if isSessionExpired(sessionStartTime, currentTime, sessionSize) {
		setSessionAccessCount(state, 0, origin)
	} else {
		accessCount = getSessionAccessCount(state, origin)
		accessCount += 1
		setSessionAccessCount(state, accessCount, origin)
	}

	log.Printf("Current session access count: %d out of %d\n", accessCount, maxAccessCount)

	message := fmt.Sprintf("Ran out of access count for current session: %d out of %d, Try after sometime!",  accessCount, maxAccessCount)

	if accessCount > 100 {
		panic(message)
	}


	return next(state, txBytes)
})