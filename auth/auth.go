package auth

import (
	"context"
	"errors"

	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/gogo/protobuf/proto"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/crypto/ed25519"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
)

var (
	nonceErrorCount metrics.Counter
)

func init() {
	fieldKeys := []string{"method", "error"}

	nonceErrorCount = kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "loomchain",
		Subsystem: "middleware",
		Name:      "nonce_error",
		Help:      "Number of invalid nonces.",
	}, fieldKeys)
}

type contextKey string

func (c contextKey) String() string {
	return "auth " + string(c)
}

var (
	ContextKeyOrigin = contextKey("origin")
)

func Origin(ctx context.Context) loom.Address {
	return ctx.Value(ContextKeyOrigin).(loom.Address)
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

	origin, err := GetOrigin(tx, state.Block().ChainID)
	if err != nil {
		return r, err
	}

	ctx := context.WithValue(state.Context(), ContextKeyOrigin, origin)
	return next(state.WithContext(ctx), tx.Inner)
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
		nonceErrorCount.Add(1)
		return r, errors.New("sequence number does not match")
	}

	return next(state, tx.Inner)
})
