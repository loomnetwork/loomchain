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
	maxDeployCount       int64
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
	maxDeployCount int64,
) *Throttle {
	return &Throttle{
		maxCallCount:         maxCallCount,
		sessionDuration:      sessionDuration,
		callLimiterPool:      make(map[string]*limiter.Limiter),
		deployLimiterPool:    make(map[string]*limiter.Limiter),
		karmaContractAddress: loom.Address{},
		maxDeployCount:       maxDeployCount,
	}
}

func (t *Throttle) getNewCallLimiter(ctx context.Context, totalKarma int64) *limiter.Limiter {
	rate := limiter.Rate{
		Period: time.Duration(t.sessionDuration) * time.Second,
		Limit:  t.maxCallCount + int64(totalKarma),
	}
	limiterStore := memory.NewStore()
	return limiter.New(limiterStore, rate)
}

func (t *Throttle) getNewDeployLimiter(ctx context.Context) *limiter.Limiter {
	rate := limiter.Rate{
		Period: time.Duration(t.sessionDuration) * time.Second,
		Limit:  t.maxDeployCount,
	}
	limiterStore := memory.NewStore()
	return limiter.New(limiterStore, rate)
}

func (t *Throttle) getCallLimiterFromPool(ctx context.Context, totalKarma int64) *limiter.Limiter {
	address := auth.Origin(ctx).String()
	_, ok := t.callLimiterPool[address]
	if !ok {
		t.callLimiterPool[address] = t.getNewCallLimiter(ctx, totalKarma)
	}
	if t.callLimiterPool[address].Rate.Limit != t.maxCallCount+int64(totalKarma) {
		delete(t.callLimiterPool, address)
		t.callLimiterPool[address] = t.getNewCallLimiter(ctx, totalKarma)
	}

	return t.callLimiterPool[address]
}

func (t *Throttle) getDeployLimiterFromPool(ctx context.Context) *limiter.Limiter {
	address := auth.Origin(ctx).String()
	_, ok := t.deployLimiterPool[address]
	if !ok {
		t.deployLimiterPool[address] = t.getNewDeployLimiter(ctx)
	}
	return t.deployLimiterPool[address]
}

func (t *Throttle) getDeployLimiterContext(ctx context.Context, nonce uint64, key string) (limiter.Context, error) {
	address := auth.Origin(ctx).String()
	if address == t.lastAddress && nonce == t.lastNonce && t.lastId == deployId {
		return t.lastLimiterContext, nil
	} else {
		t.lastAddress = address
		t.lastNonce = nonce
		t.lastId = deployId
		limiterCtx, err := t.getDeployLimiterFromPool(ctx).Get(ctx, key)
		t.lastLimiterContext = limiterCtx
		return limiterCtx, err
	}
}

func (t *Throttle) getCallLimiterContext(ctx context.Context, nonce uint64, totalKarma int64, key string) (limiter.Context, error) {
	address := auth.Origin(ctx).String()
	if address == t.lastAddress && nonce == t.lastNonce && t.lastId == callId {
		return t.lastLimiterContext, nil
	} else {
		t.lastAddress = address
		t.lastNonce = nonce
		t.lastId = callId
		limiterCtx, err := t.getCallLimiterFromPool(ctx, totalKarma).Get(ctx, key)
		t.lastLimiterContext = limiterCtx
		return limiterCtx, err
	}
}

func (t *Throttle) runDeployThrottle(state loomchain.State, nonce uint64, origin loom.Address) error {
	deploylctx, err := t.getDeployLimiterContext(state.Context(), nonce, delpoyKey)
	if err != nil {
		return errors.Wrap(err, "deploy limiter context")
	}

	if deploylctx.Reached {
		message := fmt.Sprintf(
			"Out of deploys for current session: %d out of %d, Try after sometime!",
			deploylctx.Limit-deploylctx.Remaining, deploylctx.Limit,
		)
		return errors.New(message)
	}
	return nil
}

func (t *Throttle) runCallThrottle(state loomchain.State, nonce uint64, totalKarma int64, origin loom.Address) error {
	calllctx, err := t.getCallLimiterContext(state.Context(), nonce, totalKarma, key)
	if err != nil {
		return errors.Wrap(err, "deploy limiter context")
	}

	if calllctx.Reached {
		message := fmt.Sprintf(
			"Out of calls for current session: %d out of %d, Try after sometime!",
			calllctx.Limit-calllctx.Remaining,
			calllctx.Limit,
		)
		return errors.New(message)
	}
	return nil
}

func (t *Throttle) getTotalKarma(state loomchain.State, origin loom.Address, txId uint32) (int64, error) {
	karmaState, err := t.getKarmaState(state)
	if err != nil {
		return 0, err
	}

	var sources ktypes.KarmaSources
	if karmaState.Has(karma.SourcesKey) {
		if err := proto.Unmarshal(karmaState.Get(karma.SourcesKey), &sources); err != nil {
			return 0, errors.Wrap(err, "throttle: unmarshal karma sources")
		}
	} else {
		return 0, errors.New("throttle: karma sources not found")
	}

	stateKey := karma.GetUserStateKey(origin.MarshalPB())
	var curState ktypes.KarmaState
	if karmaState.Has(stateKey) {
		curStateB := karmaState.Get(stateKey)
		err := proto.Unmarshal(curStateB, &curState)
		if err != nil {
			return 0, errors.Wrap(err, "throttle: unmarshal karma states")
		}
	}
	if txId == 1 {
		return karma.CalculateTotalKarma(sources, curState, ktypes.SourceTarget_DEPLOY), nil
	} else {
		return karma.CalculateTotalKarma(sources, curState, ktypes.SourceTarget_CALL), nil
	}

}

func (t *Throttle) getKarmaState(chainState loomchain.State) (loomchain.State, error) {
	contractState := loomchain.StateWithPrefix(loom.DataPrefix(t.karmaContractAddress), chainState)
	return contractState, nil
}
