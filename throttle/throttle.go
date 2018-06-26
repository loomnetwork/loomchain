package throttle

import (
	"time"
	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/store/memory"
	"context"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/client"
	"github.com/loomnetwork/loadtests/common"
	ktype "github.com/loomnetwork/loomchain/builtin/plugins/karma/types"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/log"
	"errors"
	"fmt"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/plugin"
)

type Throttle struct {
	maxAccessCount 		int64
	sessionDuration 	int64
	limiterPool			map[string]*limiter.Limiter
	totalAccessCount	map[string]int64
}


func NewThrottle(maxAccessCount int64, sessionDuration int64) *Throttle {
	return &Throttle{
		maxAccessCount:			maxAccessCount,
		sessionDuration:		sessionDuration,
		limiterPool:			make(map[string]*limiter.Limiter),
		totalAccessCount:		make(map[string]int64),
	}
}

func (t *Throttle) getNewLimiter(ctx context.Context, totalKarma float64) *limiter.Limiter {
	rate := limiter.Rate{
		Period: time.Duration(t.sessionDuration) * time.Second,
		Limit:  t.maxAccessCount + int64(totalKarma),
	}
	limiterStore := memory.NewStore()
	return limiter.New(limiterStore, rate)
}

func (t *Throttle) getLimiterFromPool(ctx context.Context, totalKarma float64) *limiter.Limiter {
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

func (t *Throttle) run(ctx context.Context, key string) (limiter.Context, error) {
	karmaContract, err := t.getKarmaContract(ctx)
	if err != nil {
		log.Error(err.Error())
		return limiter.Context{}, err
	}

	totalKarma, err := t.getTotalKarma(ctx, karmaContract)
	if err != nil {
		log.Error(err.Error())
		return limiter.Context{}, err
	}

	log.Info(fmt.Sprintf("Total karma: %f", totalKarma))

	return t.getLimiterFromPool(ctx, totalKarma).Get(ctx, key)
}

func (t *Throttle) getKarmaContract(ctx context.Context) (*client.Contract, error) {
	origin := auth.Origin(ctx)
	if origin.IsEmpty() {
		return nil, errors.New("transaction has no origin")
	}
	contractAddr, _ := loom.LocalAddressFromHexString("0xe288d6eec7150D6a22FDE33F0AA2d81E06591C4d")
	rpcClient := client.NewDAppChainRPCClient(origin.ChainID, "http://127.0.0.1:46658/rpc", "http://127.0.0.1:46658/query")
	return client.NewContract(rpcClient, contractAddr), nil

}

func (t *Throttle) getTotalKarma(ctx context.Context, contract *client.Contract) (float64, error) {
	origin := auth.Origin(ctx)
	if origin.IsEmpty() {
		return 0, errors.New("transaction has no origin")
	}
	var totalKarma ktype.KarmaTotal
	params := &ktype.KarmaUser{
		Address: origin.String(),
	}
	_, err := contract.StaticCall("GetTotal", params, loom.RootAddress(common.ChainID), &totalKarma)
	if err != nil {
		log.Error(err.Error())
	}

	return totalKarma.GetCount(), nil
}
