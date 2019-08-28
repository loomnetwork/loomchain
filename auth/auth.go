package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/gogo/protobuf/proto"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/crypto/ed25519"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/features"
	"github.com/loomnetwork/loomchain/store"
)

var (
	nonceErrorCount metrics.Counter
)

func init() {
	nonceErrorCount = kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "loomchain",
		Subsystem: "middleware",
		Name:      "nonce_error",
		Help:      "Number of invalid nonces.",
	}, []string{})
}

type contextKey string

func (c contextKey) String() string {
	return "auth " + string(c)
}

var (
	ContextKeyOrigin  = contextKey("origin")
	ContextKeyCheckTx = contextKey("CheckTx")
)

func Origin(ctx context.Context) loom.Address {
	return ctx.Value(ContextKeyOrigin).(loom.Address)
}

var SignatureTxMiddleware = loomchain.TxMiddlewareFunc(func(
	state loomchain.State,
	txBytes []byte,
	next loomchain.TxHandlerFunc,
	isCheckTx bool,
) (loomchain.TxHandlerResult, error) {
	var r loomchain.TxHandlerResult

	var tx SignedTx
	err := proto.Unmarshal(txBytes, &tx)
	if err != nil {
		return r, err
	}

	origin, err := GetOrigin(tx, state.Block().ChainID)
	if err != nil {
		return r, err
	}

	ctx := context.WithValue(state.Context(), ContextKeyOrigin, origin)
	return next(state.WithContext(ctx), tx.Inner, isCheckTx)
})

func GetOrigin(tx SignedTx, chainId string) (loom.Address, error) {
	if len(tx.PublicKey) != ed25519.PublicKeySize {
		return loom.Address{}, errors.New("invalid public key length")
	}

	if len(tx.Signature) != ed25519.SignatureSize {
		return loom.Address{}, errors.New("invalid signature ed25519 signature size length")
	}

	if !ed25519.Verify(tx.PublicKey, tx.Inner, tx.Signature) {
		return loom.Address{}, errors.New("invalid signature ed25519 verify")
	}

	return loom.Address{
		ChainID: chainId,
		Local:   loom.LocalAddressFromPublicKey(tx.PublicKey),
	}, nil
}

func nonceKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte("nonce"), addr.Bytes())
}

func Nonce(state loomchain.ReadOnlyState, addr loom.Address) uint64 {
	return loomchain.NewSequence(nonceKey(addr)).Value(state)
}

type NonceHandler struct {
	nonceCache map[string]uint64
	lastHeight int64
}

func (n *NonceHandler) Nonce(
	state loomchain.State,
	kvStore store.KVStore,
	txBytes []byte,
	next loomchain.TxHandlerFunc,
	isCheckTx bool,
) (loomchain.TxHandlerResult, error) {
	var r loomchain.TxHandlerResult
	origin := Origin(state.Context())
	if origin.IsEmpty() {
		return r, errors.New("transaction has no origin [nonce]")
	}
	if n.lastHeight != state.Block().Height {
		n.lastHeight = state.Block().Height
		n.nonceCache = make(map[string]uint64)
		//clear the cache for each block
	}
	var seq uint64
	if state.FeatureEnabled(features.IncrementNonceOnFailedTxFeature, false) && !isCheckTx {
		// Unconditionally increment the nonce in DeliverTx, regardless of whether the tx succeeds
		seq = loomchain.NewSequence(nonceKey(origin)).Next(kvStore)
	} else {
		seq = loomchain.NewSequence(nonceKey(origin)).Next(state)
	}

	var tx NonceTx
	err := proto.Unmarshal(txBytes, &tx)
	if err != nil {
		return r, err
	}

	//TODO nonce cache is temporary until we have a separate atomic state for the entire checktx flow
	cacheSeq := n.nonceCache[origin.String()]
	//If we have a client send multiple transactions in a single block we can run into this problem
	if cacheSeq != 0 && isCheckTx { //only run this code during checktx
		seq = cacheSeq
	} else {
		n.nonceCache[origin.String()] = seq
	}

	if tx.Sequence != seq {
		nonceErrorCount.Add(1)
		return r, fmt.Errorf("sequence number does not match expected %d got %d", seq, tx.Sequence)
	}

	return next(state, tx.Inner, isCheckTx)
}

func (n *NonceHandler) IncNonce(state loomchain.State,
	txBytes []byte,
	result loomchain.TxHandlerResult,
	postcommit loomchain.PostCommitHandler,
) error {

	origin := Origin(state.Context())
	if origin.IsEmpty() {
		return errors.New("transaction has no origin [IncNonce]")
	}

	//We only increment the nonce if the transaction is successful
	//There are situations in checktx where we may not have committed the transaction to the statestore yet
	n.nonceCache[origin.String()] = n.nonceCache[origin.String()] + 1

	return nil
}

var NonceTxHandler = NonceHandler{nonceCache: make(map[string]uint64), lastHeight: 0}

var NonceTxPostNonceMiddleware = loomchain.PostCommitMiddlewareFunc(NonceTxHandler.IncNonce)

var NonceTxMiddleware = func(kvStore store.KVStore) loomchain.TxMiddlewareFunc {
	nonceTxMiddleware := func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (loomchain.TxHandlerResult, error) {
		return NonceTxHandler.Nonce(state, kvStore, txBytes, next, isCheckTx)
	}
	return loomchain.TxMiddlewareFunc(nonceTxMiddleware)
}
