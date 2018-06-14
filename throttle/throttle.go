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
)

type Throttle struct {
	limiter			*limiter.Limiter
}


func NewThrottle(maxAccessCount int64, sessionDuration int64) *Throttle {
	rate := limiter.Rate{
		Period: time.Duration(sessionDuration) * time.Second,
		Limit:  maxAccessCount,
	}
	limiterStore := memory.NewStore()
	return &Throttle{
		limiter:			limiter.New(limiterStore, rate),
	}
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

	//TODO: figure out a way to reset the counter limit


	return t.limiter.Get(ctx, key)
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
