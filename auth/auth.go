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

	ctmp := state.Context()
	if ctmp == nil {
		ctmp = context.Background()
	}
	ctx := context.WithValue(ctmp, ContextKeyOrigin, origin)
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
	nonceCache map[string]uint64 // stores the next nonce expected to be seen for each account
	lastHeight int64
}

func NewNonceHandler() *NonceHandler {
	return &NonceHandler{nonceCache: make(map[string]uint64), lastHeight: 0}
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
		// Clear the cache for each block
		n.nonceCache = make(map[string]uint64)
	}
	var seq uint64

	incrementNonceOnFailedTx := state.Config().GetNonceHandler().GetIncNonceOnFailedTx()
	if incrementNonceOnFailedTx && !isCheckTx {
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
	// The client may speculatively increment nonces without waiting for previous txs to be committed,
	// so it's possible for a single account to submit multiple transactions in a single block.
	if cacheSeq != 0 && isCheckTx {
		// In CheckTx we only update the cache if the tx is successful (see IncNonce)
		seq = cacheSeq
	} else {
		if incrementNonceOnFailedTx {
			if isCheckTx {
				n.nonceCache[origin.String()] = seq
			} else {
				// In DeliverTx we update the cache unconditionally, because even if the tx fails the
				// nonce change will be persisted. We do this here because post commit middleware doesn't
				// run for failed txs, so IncNonce can't be relied upon.
				n.nonceCache[origin.String()] = seq + 1
			}
		} else {
			n.nonceCache[origin.String()] = seq
		}
	}

	if tx.Sequence != seq {
		nonceErrorCount.Add(1)
		return r, fmt.Errorf("sequence number does not match expected %d got %d", seq, tx.Sequence)
	}

	return next(state, tx.Inner, isCheckTx)
}

func (n *NonceHandler) IncNonce(
	state loomchain.State,
	txBytes []byte,
	result loomchain.TxHandlerResult,
	postcommit loomchain.PostCommitHandler,
	isCheckTx bool,
) error {
	origin := Origin(state.Context())
	if origin.IsEmpty() {
		return errors.New("transaction has no origin [IncNonce]")
	}

	// We only increment the nonce if the transaction is successful
	// There are situations in checktx where we may not have committed the transaction to the statestore yet
	if state.Config().GetNonceHandler().GetIncNonceOnFailedTx() {
		if isCheckTx {
			n.nonceCache[origin.String()] = n.nonceCache[origin.String()] + 1
		}
	} else {
		n.nonceCache[origin.String()] = n.nonceCache[origin.String()] + 1
	}
	return nil
}

func (n *NonceHandler) TxMiddleware(kvStore store.KVStore) loomchain.TxMiddlewareFunc {
	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (res loomchain.TxHandlerResult, err error) {
		return n.Nonce(state, kvStore, txBytes, next, isCheckTx)
	})
}

func (n *NonceHandler) PostCommitMiddleware() loomchain.PostCommitMiddlewareFunc {
	return loomchain.PostCommitMiddlewareFunc(n.IncNonce)
}
