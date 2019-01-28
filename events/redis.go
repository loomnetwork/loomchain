package events

import (
	"fmt"
	"strings"

	"github.com/loomnetwork/loomchain"
	log "github.com/loomnetwork/loomchain/log"

	"github.com/gomodule/redigo/redis"
)

type RedisEventDispatcherConfig struct {
	URI string
}

// RedisEventDispatcher is a post commit hook to dispatch events to redis
type RedisEventDispatcher struct {
	redis redis.Conn
	queue string
}

var _ loomchain.EventDispatcher = &RedisEventDispatcher{}

// NewRedisEventDispatcher create a new redis dispatcher
func NewRedisEventDispatcher(host string) (*RedisEventDispatcher, error) {
	c, err := redis.DialURL(host)
	if err != nil {
		return nil, err
	}
	queuename := "loomevents"
	return &RedisEventDispatcher{
		redis: c,
		queue: queuename,
	}, nil
}

// Send sends the event
func (ed *RedisEventDispatcher) Send(index uint64, msg []byte) error {
	log.Info("Emiting event", "index", index, "msg", msg)
	if _, err := ed.redis.Do("ZADD", ed.queue, index, msg); err != nil {
		return err
	}
	return nil
}

func NewEventDispatcher(uri string) (loomchain.EventDispatcher, error) {
	if strings.HasPrefix(uri, "redis") {
		return NewRedisEventDispatcher(uri)
	}
	return nil, fmt.Errorf("Cannot handle event dispatcher uri %s", uri)
}
