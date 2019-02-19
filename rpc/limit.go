package rpc

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tendermint/tendermint/mempool"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	types "github.com/tendermint/tendermint/rpc/lib/types"
	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/store/memory"
)

const (
	limiterPeriod   = time.Duration(60) * time.Second
	limiterCount    = 10
	CleanupInterval = time.Duration(100) * time.Minute
	TimeKeepInCache = time.Duration(5) * time.Minute

	keyVisitors = "Visitors"
)

var (
	visitors = make(map[string]*visitor)
	mtx      sync.RWMutex
)

type visitor struct {
	limiter      *limiter.Limiter
	lastFailedTx time.Time
}

func init() {
	go cleanupVisitors()
}

func cleanupVisitors() {
	for {
		time.Sleep(CleanupInterval)
		mtx.Lock()
		for ip, v := range visitors {
			if time.Now().Sub(v.lastFailedTx) > TimeKeepInCache {
				delete(visitors, ip)
			}
		}
		mtx.Unlock()
	}
}

func addVistior(ip string) *limiter.Limiter {
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
	mtx.Lock()
	if visitor, exists := visitors[ip]; exists {
		vistorsLimiter = visitor.limiter
		visitor.lastFailedTx = time.Now()
		mtx.Unlock()
		return vistorsLimiter
	}
	mtx.Unlock()
	return addVistior(ip)
}

func isLimitReached(ip string) (bool, error) {
	mtx.RLock()
	defer mtx.RUnlock()
	if visitor, exists := visitors[ip]; exists {
		visitorLimiter, err := visitor.limiter.Peek(context.TODO(), keyVisitors)
		return visitorLimiter.Reached, err
	} else {
		return false, nil
	}
}

func limitVisits(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ipAddr := getRealAddr(r)
		limitReached, err := isLimitReached(ipAddr)
		if err != nil {
			// If using memory store, Peek & isLimitReached cannot return an error. Error only on redis store.
			http.Error(w, http.StatusText(400), http.StatusBadRequest)
			return
		}
		if limitReached {
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}
		writer := NewResponseWriterWithStatus(w)

		next.ServeHTTP(writer, r)

		if writer.statusCode != http.StatusOK {
			return
		}

		if fail, _ := isRestrictedFailTx(writer); fail {
			// Increment count for current visitor
			if _, err := getVisitor(ipAddr).Get(context.TODO(), keyVisitors); err != nil {
				// If using memory store, Get cannot return an error. Error only on redis store.
				http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
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

// Increase the tx fail limiter if CheckTx returns Code = 1 or if there is a ErrTxInCache error.
func isRestrictedFailTx(writer *responseWriterWithStatus) (bool, error) {
	var res types.RPCResponse
	if err := json.Unmarshal(writer.lastWrite, &res); err != nil {
		return false, err
	}

	var result ctypes.ResultBroadcastTx
	if res.Error != nil {
		return res.Error.Data == mempool.ErrTxInCache.Error(), nil
	}

	if len(res.Result) == 0 {
		// should not happen exactly one of Result or Error should be non nil.
		return false, nil
	}

	if err := cdc.UnmarshalJSON(res.Result, &result); err != nil {
		return false, err
	}
	return result.Code == 1, nil
}
