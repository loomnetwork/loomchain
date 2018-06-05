package auth

import (
	"context"
	"errors"

	"github.com/gogo/protobuf/proto"
	"golang.org/x/crypto/ed25519"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
	"fmt"
	"time"
	"runtime/debug"
	"bytes"
	"encoding/binary"
	"github.com/loomnetwork/loomchain/log"
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

func getSessionKey(origin loom.Address) []byte {
	return util.PrefixKey([]byte("session-start-time") , origin.Bytes())
}

func startSessionTime(state loomchain.State, origin loom.Address) (int64) {
	fmt.Println("----- No session found -----")

	sessionTime := time.Now().Unix()

	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, sessionTime)
	if err != nil {
		panic(err)
	}

	state.Set(getSessionKey(origin), buf.Bytes())

	return int64(binary.BigEndian.Uint64(state.Get(getSessionKey(origin))))
}

func getSessionTime(state loomchain.State, origin loom.Address) (int64) {
	return int64(binary.BigEndian.Uint64(state.Get(getSessionKey(origin))))
}

func isSessionExpired(sessionStartTime, currentTime int64) (bool) {
	// TODO: current session time limit 10 minutes
	var sessionSize int64 = 600
	return sessionStartTime + sessionSize <= currentTime
}

func setSessionAccessCount(state loomchain.State, accessCount int16, origin loom.Address) {

	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, accessCount)
	if err != nil {
		panic(err)
	}

	state.Set([]byte("session-access-count"), buf.Bytes())
}

func getSessionAccessCount(state loomchain.State, origin loom.Address) (int16) {
	return int16(binary.BigEndian.Uint16(state.Get(getSessionKey(origin))))
}

var ThrottleTxMiddleware = loomchain.TxMiddlewareFunc(func(
	state loomchain.State,
	txBytes []byte,
	next loomchain.TxHandlerFunc,
) (res loomchain.TxHandlerResult, err error)  {
	origin := Origin(state.Context())
	if origin.IsEmpty() {
		return res, errors.New("transaction has no origin")
	}

	fmt.Println("------------------------------------------------------------")
	fmt.Println("ThrottleTxMiddleware: ", origin)

	currentTime := time.Now().Unix()

	var accessCount int16
	var sessionStartTime int64
	if state.Has(getSessionKey(origin)) {
		sessionStartTime = getSessionTime(state, origin)
	}else{
		sessionStartTime = startSessionTime(state, origin)
		setSessionAccessCount(state, 0, origin)
	}
	fmt.Println("start time: ",sessionStartTime)

	if isSessionExpired(sessionStartTime, currentTime) {
		fmt.Println("session expired:")
		setSessionAccessCount(state, 0, origin)
	} else {
		accessCount = getSessionAccessCount(state, origin) + 1
		setSessionAccessCount(state, accessCount, origin)
	}

	fmt.Println("---------------------- Current access count: ", accessCount)


	defer func() {
		if accessCount > 100 {
			fmt.Println("---------------------- Ran out of access count: ", accessCount)
			fmt.Println(accessCount)
			logger := log.Root
			message := fmt.Sprintf("Ran out of access count: %d",  accessCount)
			logger.Error(message)
			println(debug.Stack())
			err = errors.New(message)
		}
	}()

	fmt.Println("------------------------------------------------------------")


	return next(state, txBytes)
})