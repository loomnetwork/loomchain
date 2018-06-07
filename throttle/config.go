package throttle

import (
	"sync"
	"github.com/loomnetwork/loomchain/store"
)

type Config struct {
	ThrottleMaxAccessCount int16
	ThrottleSessionSize int64
	Store *store.MemStore
}

var instance *Config
var once sync.Once

func Singletone() *Config {
	once.Do(func() {
		instance = &Config{
			ThrottleMaxAccessCount:  100,
			ThrottleSessionSize:     600,
			Store: store.NewMemStore(),
		}
	})
	return instance
}
