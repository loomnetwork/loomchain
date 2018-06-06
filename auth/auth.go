package auth

import (
	"context"
	"errors"

	"github.com/gogo/protobuf/proto"
	"golang.org/x/crypto/ed25519"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
	"time"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
)

type contextKey string

func (c contextKey) String() string {
	return "auth " + string(c)
}

var (
	contextKeyOrigin = contextKey("origin")
)

func Origin(ctx context.Context) loom.Address {
	return ctx.Value(contextKeyOrigin).(loom.Address)
}

var SignatureTxMiddleware = loomchain.TxMiddlewareFunc(func(
	state loomchain.State,
	txBytes []byte,
	next loomchain.TxHandlerFunc,
) (loomchain.TxHandlerResult, error) {
	var r loomchain.TxHandlerResult

	var tx SignedTx
	err := proto.Unmarshal(txBytes, &tx)
	if err != nil {
		return r, err
	}

	if len(tx.PublicKey) != ed25519.PublicKeySize {
		return r, errors.New("invalid public key length")
	}

	if len(tx.Signature) != ed25519.SignatureSize {
		return r, errors.New("invalid signature length")
	}

	if !ed25519.Verify(tx.PublicKey, tx.Inner, tx.Signature) {
		return r, errors.New("invalid signature")
	}

	origin := loom.Address{
		ChainID: state.Block().ChainID,
		Local:   loom.LocalAddressFromPublicKey(tx.PublicKey),
	}

	ctx := context.WithValue(state.Context(), contextKeyOrigin, origin)
	return next(state.WithContext(ctx), tx.Inner)
})

func nonceKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte("nonce"), addr.Bytes())
}

func Nonce(state loomchain.ReadOnlyState, addr loom.Address) uint64 {
	return loomchain.NewSequence(nonceKey(addr)).Value(state)
}

var NonceTxMiddleware = loomchain.TxMiddlewareFunc(func(
	state loomchain.State,
	txBytes []byte,
	next loomchain.TxHandlerFunc,
) (loomchain.TxHandlerResult, error) {
	var r loomchain.TxHandlerResult
	origin := Origin(state.Context())
	if origin.IsEmpty() {
		return r, errors.New("transaction has no origin")
	}
	seq := loomchain.NewSequence(nonceKey(origin)).Next(state)

	var tx NonceTx
	err := proto.Unmarshal(txBytes, &tx)
	if err != nil {
		return r, err
	}

	if tx.Sequence != seq {
		return r, errors.New("sequence number does not match")
	}

	return next(state, tx.Inner)
})

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
	state.Set(getSessionStartTimeKey(origin), startSessionTimeInBytes())
	return int64(binary.BigEndian.Uint64(state.Get(getSessionStartTimeKey(origin))))
}

func getSessionTime(state loomchain.State, origin loom.Address) (int64) {
	value := state.Get(getSessionStartTimeKey(origin))
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

	state.Set(getSessionAccessCountKey(origin), buf.Bytes())
}

func getSessionAccessCount(state loomchain.State, origin loom.Address) (int16) {
	return int16(binary.BigEndian.Uint16(state.Get(getSessionAccessCountKey(origin))))
}

var ThrottleTxMiddleware = loomchain.TxMiddlewareFunc(func(
	state loomchain.State,
	txBytes []byte,
	next loomchain.TxHandlerFunc,
) (res loomchain.TxHandlerResult, err error)  {
	// TODO: current session time limit 10 minutes
	var maxAccessCount = 100
	var sessionSize int64 = 600

	origin := Origin(state.Context())
	if origin.IsEmpty() {
		return res, errors.New("transaction has no origin")
	}

	currentTime := time.Now().Unix()

	var accessCount int16
	var sessionStartTime int64
	if state.Has(getSessionStartTimeKey(origin)) {
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