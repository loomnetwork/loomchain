package events

import (
	"github.com/loomnetwork/loomchain/log"

	"github.com/gomodule/redigo/redis"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
)

// RedisEventDispatcher is a post commit hook to dispatch events to redis
type RedisEventDispatcher struct {
	redis redis.Conn
	queue string
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

func (ed *RedisEventDispatcher) SaveToChain(msgs []*types.EventData, txRes *loomchain.TxHandlerResult) {

}
