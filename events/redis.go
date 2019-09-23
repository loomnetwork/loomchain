package events

import (
	"github.com/loomnetwork/loomchain/log"

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

//var _ loomchain.EventDispatcher = &RedisEventDispatcher{}

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
func (ed *RedisEventDispatcher) Send(index uint64, eventIndex int, msg []byte) error {
	log.Info("Emitting event", "index", index, "msg", msg)
	if _, err := ed.redis.Do("ZADD", ed.queue, index, msg); err != nil {
		return err
	}
	return nil
}

func (ed *RedisEventDispatcher) Flush() {
}
