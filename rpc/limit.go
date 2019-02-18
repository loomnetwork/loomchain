package rpc

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/store/memory"
)

const (
	limiterPeriod   = 60
	limiterCount    = 10
	keyVisitors     = "Visitors"
	CleanupInterval = 10 * time.Minute
	TimeKeepInCache = 5 * time.Minute
)

var (
	visitors = make(map[string]*visitor)
	mtx      sync.RWMutex
)

type visitor struct {
	limiter  *limiter.Limiter
	lastSeen time.Time
}

func init() {
	go cleanupVisitors()
}

func cleanupVisitors() {
	for {
		time.Sleep(CleanupInterval)
		mtx.Lock()
		for ip, v := range visitors {
			if time.Now().Sub(v.lastSeen) > TimeKeepInCache {
				delete(visitors, ip)
			}
		}
		mtx.Unlock()
	}
}

func addVisitor(ip string) *limiter.Limiter {
	newLimiter := limiter.New(memory.NewStore(), limiter.Rate{
		Period: limiterPeriod,
		Limit:  limiterCount,
	})
	mtx.Lock()
	visitors[ip] = &visitor{newLimiter, time.Now()}
	mtx.Unlock()
	return newLimiter
}

func getVisitor(ip string) *limiter.Limiter {
	var vistorsLimiter *limiter.Limiter
	mtx.RLock()
	visitor, exists := visitors[ip]
	if exists {
		vistorsLimiter = visitor.limiter
		visitor.lastSeen = time.Now()
	}
	mtx.RUnlock()
	if !exists {
		vistorsLimiter = addVisitor(ip)
	}
	return vistorsLimiter
}

func limitVisits(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visitorCtx, err := getVisitor(r.RemoteAddr).Get(context.TODO(), keyVisitors)
		if err != nil {
			// If using memory store, Get cannot return an error. Error only on redis store.
			http.Error(w, http.StatusText(400), http.StatusBadRequest)
			return
		}
		if visitorCtx.Reached == false {
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
