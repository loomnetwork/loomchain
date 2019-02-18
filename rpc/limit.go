package rpc

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	types "github.com/tendermint/tendermint/rpc/lib/types"
	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/store/memory"
)

const (
	limiterPeriod   = 600
	limiterCount    = 3
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
		ipAddr := getRealAddr(r)
		visitorCtx, err := getVisitor(ipAddr).Peek(context.TODO(), keyVisitors)
		if err != nil {
			// If using memory store, Peek cannot return an error. Error only on redis store.
			http.Error(w, http.StatusText(400), http.StatusBadRequest)
			return
		}
		if visitorCtx.Reached {
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}
		writer := NewResponseWriterWithStatus(w)

		next.ServeHTTP(writer, r)

		if writer.statusCode != http.StatusOK {
			// Call Get to increment counter.
			if _, err := getVisitor(ipAddr).Get(context.TODO(), keyVisitors); err != nil {
				// If using memory store, Get cannot return an error. Error only on redis store.
				http.Error(w, http.StatusText(400), http.StatusBadRequest)
				return
			}
		}
		code, err := getResponseCode(writer.lastWrite)
		if err == nil && code != abci.CodeTypeOK {
			if _, err := getVisitor(ipAddr).Get(context.TODO(), keyVisitors); err != nil {
				http.Error(w, http.StatusText(400), http.StatusBadRequest)
				return
			}
		}
	})
}

func getRealAddr(r *http.Request) string {
	remoteIP := ""
	// the default is the originating ip. but we try to find better options because this is almost
	// never the right IP
	if parts := strings.Split(r.RemoteAddr, ":"); len(parts) == 2 {
		remoteIP = parts[0]
	}
	// If we have a forwarded-for header, take the address from there
	if xff := strings.Trim(r.Header.Get("X-Forwarded-For"), ","); len(xff) > 0 {
		addrs := strings.Split(xff, ",")
		lastFwd := addrs[len(addrs)-1]
		if ip := net.ParseIP(lastFwd); ip != nil {
			remoteIP = ip.String()
		}
		// parse X-Real-Ip header
	} else if xri := r.Header.Get("X-Real-Ip"); len(xri) > 0 {
		if ip := net.ParseIP(xri); ip != nil {
			remoteIP = ip.String()
		}
	}

	return remoteIP

}

func getResponseCode(resultBytes []byte) (uint32, error) {
	var res types.RPCResponse
	if err := json.Unmarshal(resultBytes, &res); err != nil {
		return 0, err
	}
	var result ctypes.ResultBroadcastTx
	if err := cdc.UnmarshalJSON(res.Result, &result); err != nil {
		return 0, err
	}
	return result.Code, nil
}
