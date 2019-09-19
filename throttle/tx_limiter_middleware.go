package throttle

import (
	"context"
	"time"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/state"

	"github.com/pkg/errors"
	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/store/memory"
)

type TxLimiterConfig struct {
	// Enables the tx limiter middleware
	Enabled bool
	// Number of seconds each session lasts
	SessionDuration int64
	// Maximum number of txs that should be allowed per session
	MaxTxsPerSession int64
}

func DefaultTxLimiterConfig() *TxLimiterConfig {
	return &TxLimiterConfig{
		SessionDuration:  60,
		MaxTxsPerSession: 60,
	}
}

// Clone returns a deep clone of the config.
func (c *TxLimiterConfig) Clone() *TxLimiterConfig {
	if c == nil {
		return nil
	}
	clone := *c
	return &clone
}

type txLimiter struct {
	*limiter.Limiter
}

func newTxLimiter(cfg *TxLimiterConfig) *txLimiter {
	return &txLimiter{
		Limiter: limiter.New(
			memory.NewStore(),
			limiter.Rate{
				Period: time.Duration(cfg.SessionDuration) * time.Second,
				Limit:  cfg.MaxTxsPerSession,
			},
		),
	}
}

func (txl *txLimiter) isAccountLimitReached(account loom.Address) bool {
	lmtCtx, err := txl.Limiter.Get(context.TODO(), account.String())
	// Doesn't look like the current implementation of the limit with the in-memory store will ever
	// return an error anyway.
	if err != nil {
		panic(err)
	}
	return lmtCtx.Reached
}

// NewTxLimiterMiddleware creates middleware that throttles txs (all types) in CheckTx, the rate
// can be configured in loom.yml. Since this middleware only runs in CheckTx the rate limit can
// differ between nodes on the same cluster, and private nodes don't really need to run the rate
// limiter at all.
func NewTxLimiterMiddleware(cfg *TxLimiterConfig) loomchain.TxMiddlewareFunc {
	txl := newTxLimiter(cfg)
	return loomchain.TxMiddlewareFunc(func(
		s state.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (loomchain.TxHandlerResult, error) {
		if !isCheckTx {
			return next(s, txBytes, isCheckTx)
		}

		origin := auth.Origin(s.Context())
		if origin.IsEmpty() {
			return loomchain.TxHandlerResult{}, errors.New("throttle: transaction has no origin [get-karma]")
		}

		if txl.isAccountLimitReached(origin) {
			return loomchain.TxHandlerResult{}, errors.New("tx limit reached, try again later")
		}

		return next(s, txBytes, isCheckTx)
	})
}
