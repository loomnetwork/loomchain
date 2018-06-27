package throttle

import (
	"time"
	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/store/memory"
	"context"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/log"
	"errors"
	"fmt"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/gogo/protobuf/proto"
)

type Throttle struct {
	maxAccessCount 		int64
	sessionDuration 	int64
	limiterPool			map[string]*limiter.Limiter
	totalAccessCount	map[string]int64
}


func NewThrottle(maxAccessCount int64, sessionDuration int64) (*Throttle) {
	return &Throttle{
		maxAccessCount:			maxAccessCount,
		sessionDuration:		sessionDuration,
		limiterPool:			make(map[string]*limiter.Limiter),
		totalAccessCount:		make(map[string]int64),
	}
}

func (t *Throttle) getNewLimiter(ctx context.Context, totalKarma int64) *limiter.Limiter {
	rate := limiter.Rate{
		Period: time.Duration(t.sessionDuration) * time.Second,
		Limit:  t.maxAccessCount + int64(totalKarma),
	}
	limiterStore := memory.NewStore()
	return limiter.New(limiterStore, rate)
}

func (t *Throttle) getLimiterFromPool(ctx context.Context, totalKarma int64) *limiter.Limiter {
	address := auth.Origin(ctx).String()
	_, ok := t.limiterPool[address]
	if !ok {
		t.totalAccessCount[address] = int64(0)
		t.limiterPool[address] = t.getNewLimiter(ctx, totalKarma)
	}
	if t.limiterPool[address].Rate.Limit != t.maxAccessCount + int64(totalKarma){
		delete(t.limiterPool, address)
		t.limiterPool[address] = t.getNewLimiter(ctx, totalKarma)
	}
	t.totalAccessCount[address] += 1
	return t.limiterPool[address]
}

func (t *Throttle) run(state loomchain.State, key string) (limiter.Context, error) {
	totalKarma, err := t.getTotalKarma(state)
	if err != nil {
		log.Error(err.Error())
		return limiter.Context{}, err
	}

	log.Info(fmt.Sprintf("Total karma: %d", totalKarma))

	return t.getLimiterFromPool(state.Context(), totalKarma).Get(state.Context(), key)
}

func (t *Throttle) getTotalKarma(state loomchain.State) (int64, error) {
	origin := auth.Origin(state.Context())
	if origin.IsEmpty() {
		return 0, errors.New("transaction has no origin")
	}

	karmaState, err := t.getKarmaState(state)
	if err != nil {
		return 0.0, err
	}

	origin.String()

	var curConfig karma.Config
	if karmaState.Has(karma.GetConfigKey()) {
		curConfigB := karmaState.Get(karma.GetConfigKey())
		err := proto.Unmarshal(curConfigB, &curConfig)
		if err != nil {
			return 0.0, err
		}
	}else{
		return 0.0, errors.New("karma config not found")
	}

	stateKey := karma.GetUserStateKey(origin.String())
	var curState karma.State
	if karmaState.Has(stateKey) {
		curStateB := karmaState.Get(stateKey)
		err := proto.Unmarshal(curStateB, &curState)
		if err != nil {
			return 0.0, err
		}
	}

	var karmaValue int64 = 0
	for key := range curConfig.Sources {
		if value, ok := curState.SourceStates[key]; ok {
			karmaValue += curConfig.Sources[key] * value
		}
	}

	if karmaValue > curConfig.MaxKarma {
		karmaValue = curConfig.MaxKarma
	}

	if karmaValue > curConfig.MaxKarma {
		karmaValue = curConfig.MaxKarma
	}

	return karmaValue, nil
}

func (t *Throttle) getKarmaState(chainState loomchain.State) (loomchain.State, error) {
	registryObject := &registry.StateRegistry{
		State: chainState,
	}

	contractAddress, err := registryObject.Resolve("karma")
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	contractState := loomchain.StateWithPrefix(plugin.DataPrefix(contractAddress), chainState)

	return contractState, nil
}