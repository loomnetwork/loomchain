package throttle

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/store/memory"

	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin/contractpb"

	"github.com/loomnetwork/loomchain/auth/keys"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	appstate "github.com/loomnetwork/loomchain/state"
)

const (
	deployId    = uint32(1)
	callId      = uint32(2)
	migrationId = uint32(3)
)

type Throttle struct {
	MaxCallCount         int64
	sessionDuration      int64
	callLimiterPool      map[string]*limiter.Limiter
	deployLimiterPool    map[string]*limiter.Limiter
	karmaContractAddress loom.Address

	lastAddress        string
	lastLimiterContext limiter.Context
	lastNonce          uint64
	lastId             uint32
}

func NewThrottle(
	sessionDuration int64,
	maxCallCount int64,
) *Throttle {
	return &Throttle{
		MaxCallCount:         maxCallCount,
		sessionDuration:      sessionDuration,
		callLimiterPool:      make(map[string]*limiter.Limiter),
		deployLimiterPool:    make(map[string]*limiter.Limiter),
		karmaContractAddress: loom.Address{},
	}
}

func (t *Throttle) GetNewLimiter(ctx context.Context, limit int64) *limiter.Limiter {
	rate := limiter.Rate{
		Period: time.Duration(t.sessionDuration) * time.Second,
		Limit:  limit,
	}
	limiterStore := memory.NewStore()
	return limiter.New(limiterStore, rate)
}

func (t *Throttle) getLimiterFromPool(ctx context.Context, limit int64) *limiter.Limiter {
	address := keys.Origin(ctx).String()
	_, ok := t.callLimiterPool[address]
	if !ok {
		t.callLimiterPool[address] = t.GetNewLimiter(ctx, limit)
	}
	if t.callLimiterPool[address].Rate.Limit != limit {
		delete(t.callLimiterPool, address)
		t.callLimiterPool[address] = t.GetNewLimiter(ctx, limit)
	}

	return t.callLimiterPool[address]
}

func (t *Throttle) getLimiterContext(
	ctx context.Context, nonce uint64, limit int64, txId uint32, key string,
) (limiter.Context, error) {
	address := keys.Origin(ctx).String()
	if address == t.lastAddress && nonce == t.lastNonce && t.lastId == txId {
		return t.lastLimiterContext, nil
	} else {
		t.lastAddress = address
		t.lastNonce = nonce
		t.lastId = txId
		limiterCtx, err := t.getLimiterFromPool(ctx, limit).Get(ctx, key)
		t.lastLimiterContext = limiterCtx
		return limiterCtx, err
	}
}

func (t *Throttle) RunThrottle(
	state appstate.State, nonce uint64, origin loom.Address, limit int64, txId uint32, key string,
) error {
	limitCtx, err := t.getLimiterContext(state.Context(), nonce, limit, txId, key)
	if err != nil {
		return errors.Wrap(err, "deploy limiter context")
	}

	if limitCtx.Reached {
		message := fmt.Sprintf(
			"Out of transactions of id %v, for current session: %d out of %d; Try after %v seconds!",
			txId,
			limitCtx.Limit-limitCtx.Remaining,
			limitCtx.Limit,
			t.sessionDuration,
		)
		return errors.New(message)
	}
	return nil
}

func (t *Throttle) GetKarmaForTransaction(
	karmaContractCtx contractpb.Context, origin loom.Address, isDeployTx bool,
) (*common.BigUInt, error) {
	// TODO: maybe should only count karma from active sources
	if isDeployTx {
		return karma.GetUserKarma(karmaContractCtx, origin, ktypes.KarmaSourceTarget_DEPLOY)
	}
	return karma.GetUserKarma(karmaContractCtx, origin, ktypes.KarmaSourceTarget_CALL)
}
