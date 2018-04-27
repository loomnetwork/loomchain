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
func NewRedisEventDispatcher(host string, queuename string) (*RedisEventDispatcher, error) {
	c, err := redis.Dial("tcp", host)
	if err != nil {
		return nil, err
	}
	return &RedisEventDispatcher{
		redis: c,
		queue: queuename,
		stash: "stash-" + queuename,
	}, nil
}

// Stash stashes the event to be sent later
func (ed *RedisEventDispatcher) Stash(index int64, msg []byte) error {
	log.Printf("Stashing event to queue: index: %d, msg: %s\n", index, msg)
	if _, err := ed.redis.Do("ZADD", ed.stash, index, msg); err != nil {
		return err
	}
	return nil
}

// Send sends the event
func (ed *RedisEventDispatcher) Send(index int64, msg []byte) error {
	log.Printf("Emiting event to queue: index: %d, msg: %s\n", index, msg)
	if _, err := ed.redis.Do("ZADD", ed.queue, index, msg); err != nil {
		return err
	}
	return nil
}

// FetchStash fetches events for a given block
func (ed *RedisEventDispatcher) FetchStash(index int64) ([][]byte, error) {
	values, err := redis.Values(ed.redis.Do("ZRANGEBYSCORE", ed.stash, index, index))
	if err != nil {
		return nil, err
	}
	retVals := make([][]byte, len(values))
	for i, val := range values {
		v, ok := val.([]byte)
		if !ok {
			return nil, err
		}
		retVals[i] = v
	}
	return retVals, nil
}

// PurgeStash purges stashed events for a given block
func (ed *RedisEventDispatcher) PurgeStash(index int64) error {
	log.Printf("Purging events from stash: index: %d\n", index)
	if _, err := ed.redis.Do("ZREMRANGEBYSCORE", ed.stash, index, index); err != nil {
		return err
	}
	return nil
}
