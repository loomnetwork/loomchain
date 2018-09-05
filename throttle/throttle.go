package throttle

import (
	"context"
	"errors"
	"fmt"
	`github.com/loomnetwork/loomchain/plugin`
	//`github.com/loomnetwork/loomchain/store`
	"time"
	
	"github.com/loomnetwork/go-loom"
	
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/log"
	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/store/memory"
)

type Throttle struct {
	maxAccessCount        int64
	sessionDuration       int64
	limiterPool           map[string]*limiter.Limiter
	totalAccessCount      map[string]int64
	karmaEnabled          bool
	deployKarmaCount      int64
	totaldeployKarmaCount map[string]int64
	deployLimiterPool     map[string]*limiter.Limiter
	karmaContractAddress  loom.Address
	
	lastAddress                  string
	lastDeployLimiterContext     limiter.Context
	lastNonce                    uint64
}

func NewThrottle(maxAccessCount int64, sessionDuration int64, karmaEnabled bool, deployKarmaCount int64) *Throttle {
	return &Throttle{
		maxAccessCount:        maxAccessCount,
		sessionDuration:       sessionDuration,
		limiterPool:           make(map[string]*limiter.Limiter),
		totalAccessCount:      make(map[string]int64),
		karmaEnabled:          karmaEnabled,
		deployKarmaCount:      deployKarmaCount,
		totaldeployKarmaCount: make(map[string]int64),
		deployLimiterPool:     make(map[string]*limiter.Limiter),
		karmaContractAddress:  loom.Address{},
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

func (t *Throttle) getNewDeployLimiter(ctx context.Context) *limiter.Limiter {
	rate := limiter.Rate{
		Period: time.Duration(t.sessionDuration) * time.Second,
		Limit:  t.deployKarmaCount,
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
	if t.limiterPool[address].Rate.Limit != t.maxAccessCount+int64(totalKarma) {
		delete(t.limiterPool, address)
		t.limiterPool[address] = t.getNewLimiter(ctx, totalKarma)
	}
	t.totalAccessCount[address] += 1
	return t.limiterPool[address]
}

func (t *Throttle) getDeployLimiterFromPool(ctx context.Context) *limiter.Limiter {
	address := auth.Origin(ctx).String()

	_, ok := t.deployLimiterPool[address]
	if !ok {
		// t.totaldeployKarmaCount[address] = t.deployKarmaCount
		t.deployLimiterPool[address] = t.getNewDeployLimiter(ctx)
	}
	return t.deployLimiterPool[address]
}

func (t *Throttle) getDeployLimiterContext(ctx context.Context, nonce uint64, key string) (limiter.Context, error) {
	address := auth.Origin(ctx).String()
	if address == t.lastAddress && nonce == t.lastNonce {
		return t.lastDeployLimiterContext, nil
	} else {
		t.lastAddress = address
		t.lastNonce = nonce
		limiterCtx, err := t.getDeployLimiterFromPool(ctx).Get(ctx, key)
		return limiterCtx, err
	}
}

func (t *Throttle) run(state loomchain.State, key string, txType uint32, nonce uint64) (limiter.Context, limiter.Context, error, error) {

	var totalKarma int64 = 0
	var err error
	delpoyKey := "deploy" + key

	var lctxDeploy limiter.Context
	var err1 error
	if txType == 1 {
		lctxDeploy, err1 = t.getDeployLimiterContext(state.Context(), nonce, delpoyKey)
	} else {
		lctxDeploy = limiter.Context{}
		err1 = nil
	}

	if t.karmaEnabled {
		totalKarma, err = t.getTotalKarma(state)
		if err != nil {
			log.Error(err.Error())
			return limiter.Context{}, lctxDeploy, err, err1
		}

		log.Info(fmt.Sprintf("Total karma: %d", totalKarma))
		if totalKarma == 0 {
			return limiter.Context{}, lctxDeploy, errors.New("origin has no karma"), err1
		}
	}
	
	lctx, err := t.getLimiterFromPool(state.Context(), totalKarma).Get(state.Context(), key)
	return lctx, lctxDeploy, err, err1
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

	var curConfig karma.Config
	if karmaState.Has(karma.GetConfigKey()) {
		curConfigB := karmaState.Get(karma.GetConfigKey())
		err := proto.Unmarshal(curConfigB, &curConfig)
		if err != nil {
			return 0.0, err
		}
	} else {
		return 0.0, errors.New("karma config not found")
	}

	stateKey := karma.GetUserStateKey(origin.MarshalPB())
	var curState karma.State
	if karmaState.Has(stateKey) {
		curStateB := karmaState.Get(stateKey)
		err := proto.Unmarshal(curStateB, &curState)
		if err != nil {
			return 0.0, err
		}
	}

	var karmaValue int64 = 0
	for _, c := range curConfig.Sources {
		for _, s := range curState.SourceStates {
			if c.Name == s.Name {
				karmaValue += c.Reward * s.Count
			}
		}
	}

	return karmaValue, nil
}

func (t *Throttle) getKarmaState(chainState loomchain.State) (loomchain.State, error) {
	contractState := loomchain.StateWithPrefix(plugin.DataPrefix(t.karmaContractAddress), chainState)
	return contractState, nil
}
