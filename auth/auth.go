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

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/state"
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
	s state.State,
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

	origin, err := GetOrigin(tx, s.Block().ChainID)
	if err != nil {
		return r, err
	}

	ctx := context.WithValue(s.Context(), ContextKeyOrigin, origin)
	return next(s.WithContext(ctx), tx.Inner, isCheckTx)
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

func Nonce(s state.ReadOnlyState, addr loom.Address) uint64 {
	return loomchain.NewSequence(nonceKey(addr)).Value(s)
}

type NonceHandler struct {
	nonceCache map[string]uint64 // stores the next nonce expected to be seen for each account
	lastHeight int64
}

func (n *NonceHandler) Nonce(
	s state.State,
	kvStore store.KVStore,
	txBytes []byte,
	next loomchain.TxHandlerFunc,
	isCheckTx bool,
) (loomchain.TxHandlerResult, error) {
	var r loomchain.TxHandlerResult
	origin := Origin(s.Context())
	if origin.IsEmpty() {
		return r, errors.New("transaction has no origin [nonce]")
	}
	if n.lastHeight != s.Block().Height {
		n.lastHeight = s.Block().Height
		// Clear the cache for each block
		n.nonceCache = make(map[string]uint64)
	}
	var seq uint64

	incrementNonceOnFailedTx := s.Config().GetNonceHandler().GetIncNonceOnFailedTx()
	if incrementNonceOnFailedTx && !isCheckTx {
		// Unconditionally increment the nonce in DeliverTx, regardless of whether the tx succeeds
		seq = loomchain.NewSequence(nonceKey(origin)).Next(kvStore)
	} else {
		seq = loomchain.NewSequence(nonceKey(origin)).Next(s)
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

	return next(s, tx.Inner, isCheckTx)
}

func (n *NonceHandler) IncNonce(s state.State,
	txBytes []byte,
	result loomchain.TxHandlerResult,
	postcommit loomchain.PostCommitHandler,
	isCheckTx bool,
) error {
	origin := Origin(s.Context())
	if origin.IsEmpty() {
		return errors.New("transaction has no origin [IncNonce]")
	}

	// We only increment the nonce if the transaction is successful
	// There are situations in checktx where we may not have committed the transaction to the statestore yet
	if s.Config().GetNonceHandler().GetIncNonceOnFailedTx() {
		if isCheckTx {
			n.nonceCache[origin.String()] = n.nonceCache[origin.String()] + 1
		}
	} else {
		n.nonceCache[origin.String()] = n.nonceCache[origin.String()] + 1
	}
	return nil
}

var NonceTxHandler = NonceHandler{nonceCache: make(map[string]uint64), lastHeight: 0}

var NonceTxPostNonceMiddleware = loomchain.PostCommitMiddlewareFunc(NonceTxHandler.IncNonce)

var NonceTxMiddleware = func(kvStore store.KVStore) loomchain.TxMiddlewareFunc {
	nonceTxMiddleware := func(
		s state.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (loomchain.TxHandlerResult, error) {
		return NonceTxHandler.Nonce(s, kvStore, txBytes, next, isCheckTx)
	}
	return loomchain.TxMiddlewareFunc(nonceTxMiddleware)
}
