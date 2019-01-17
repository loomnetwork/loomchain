package throttle

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/store/memory"
	"github.com/loomnetwork/go-loom/common"
)

const (
	key       = "ThrottleTxMiddleWare"
	delpoyKey = "deploy" + key
	deployId  = uint32(1)
	callId    = uint32(2)
)

type Throttle struct {
	maxCallCount         int64
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
		maxCallCount:         maxCallCount,
		sessionDuration:      sessionDuration,
		callLimiterPool:      make(map[string]*limiter.Limiter),
		deployLimiterPool:    make(map[string]*limiter.Limiter),
		karmaContractAddress: loom.Address{},
	}
}

func (t *Throttle) getNewLimiter(ctx context.Context, limit int64) *limiter.Limiter {
	rate := limiter.Rate{
		Period: time.Duration(t.sessionDuration) * time.Second,
		Limit:  limit,
	}
	limiterStore := memory.NewStore()
	return limiter.New(limiterStore, rate)
}

func (t *Throttle) getLimiterFromPool(ctx context.Context, limit int64) *limiter.Limiter {
	address := auth.Origin(ctx).String()
	_, ok := t.callLimiterPool[address]
	if !ok {
		t.callLimiterPool[address] = t.getNewLimiter(ctx, t.maxCallCount + limit)
	}
	if t.callLimiterPool[address].Rate.Limit != limit {
		delete(t.callLimiterPool, address)
		t.callLimiterPool[address] = t.getNewLimiter(ctx, limit)
	}

	return t.callLimiterPool[address]
}

func (t *Throttle) getLimiterContext(ctx context.Context, nonce uint64, limit int64, txId uint32, key string) (limiter.Context, error) {
	address := auth.Origin(ctx).String()
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

func (t *Throttle) runThrottle(state loomchain.State, nonce uint64, origin loom.Address, limit int64, txId uint32, key string) error {
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

func (t *Throttle) getTotalKarma(state loomchain.State, origin loom.Address, txId uint32) (*common.BigUInt, error) {
	karmaState, err := t.getKarmaState(state)
	if err != nil {
		return nil, err
	}

	var sources ktypes.KarmaSources
	if karmaState.Has(karma.SourcesKey) {
		if err := proto.Unmarshal(karmaState.Get(karma.SourcesKey), &sources); err != nil {
			return nil, errors.Wrap(err, "throttle: unmarshal karma sources")
		}
	} else {
		return nil, errors.New("throttle: karma sources not found")
	}

	stateKey := karma.UserStateKey(origin.MarshalPB())
	var curState ktypes.KarmaState
	if karmaState.Has(stateKey) {
		curStateB := karmaState.Get(stateKey)
		err := proto.Unmarshal(curStateB, &curState)
		if err != nil {
			return nil, errors.Wrap(err, "throttle: unmarshal karma states")
		}
	}
	if txId == deployId {
		if curState.DeployKarmaTotal == nil {
			return common.BigZero(), nil
		}
		return &curState.DeployKarmaTotal.Value, nil
	} else if txId == callId {
		if curState.CallKarmaTotal == nil {
			return common.BigZero(), nil
		}
		return &curState.CallKarmaTotal.Value, nil
	} else 	{
		return nil, errors.Errorf("unknown transaction id %d", txId)
	}


}

func (t *Throttle) getKarmaState(chainState loomchain.State) (loomchain.State, error) {
	contractState := loomchain.StateWithPrefix(loom.DataPrefix(t.karmaContractAddress), chainState)
	return contractState, nil
}
