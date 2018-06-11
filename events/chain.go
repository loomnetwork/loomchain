package events

import (
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/tendermint/tmlibs/common"
	"log"
)

type ChainEventDispatcher struct {
}

// NewLogEventDispatcher create a new redis dispatcher
func NewChainEventDispatcher() *ChainEventDispatcher {
	return &ChainEventDispatcher{}
}

// Send sends the event
func (ed *ChainEventDispatcher) Send(index uint64, msg []byte) error {
	log.Printf("Event emitted: index: %d, length: %d, msg: %s\n", index, len(msg), msg)
	return nil
}

func (ed *ChainEventDispatcher) SaveToChain(msgs []*types.EventData, tags *[]common.KVPair) {
	for _, msg := range msgs {
		for _, topic := range msg.Topics {
			*tags = append(*tags, common.KVPair{
				Key:   []byte("topic"),
				Value: []byte(topic),
			})
		}
		*tags = append(*tags, common.KVPair{
			Key:   []byte("contract"),
			Value: msg.Address.Local,
		})
	}
}
