package events

import (
	"log"

	"github.com/gomodule/redigo/redis"
)

// RedisEventDispatcher is a post commit hook to dispatch events to redis
type RedisEventDispatcher struct {
	redis redis.Conn
	queue string
}

// NewRedisEventDispatcher create a new redis dispatcher
func NewRedisEventDispatcher(host string, queuename string) (*RedisEventDispatcher, error) {
	c, err := redis.Dial("tcp", host)
	if err != nil {
		return nil, err
	}
	return &RedisEventDispatcher{
		redis: c,
		queue: queuename,
	}, nil
}

// Send sends the message to redis queue (sorted set)
func (ed *RedisEventDispatcher) Send(index int64, msg []byte) error {
	log.Printf("Adding event to queue: index: %d, msg: %s\n", index, msg)
	ed.redis.Do("ZADD", ed.queue, index, msg)
	return nil
}
