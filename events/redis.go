package events

import (
	"log"

	"github.com/gomodule/redigo/redis"
)

// RedisEventDispatcher is a post commit hook to dispatch events to redis
type RedisEventDispatcher struct {
	redis redis.Conn
	queue string
	stash string
}

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
		stash: "stash-" + queuename,
	}, nil
}

// Send sends the event
func (ed *RedisEventDispatcher) Send(index int64, msg []byte) error {
	log.Printf("Emiting event to queue: index: %d, msg: %s\n", index, msg)
	if _, err := ed.redis.Do("ZADD", ed.queue, index, msg); err != nil {
		return err
	}
	return nil
}
