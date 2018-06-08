package throttle

import (
	"time"
	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/store/memory"
	"context"
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
	return t.limiter.Get(ctx, key)
}