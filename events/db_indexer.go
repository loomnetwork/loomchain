package events

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
)

type DBIndexerEventDispatcher struct {
	store.EventStore
}

var _ loomchain.EventDispatcher = &DBIndexerEventDispatcher{}

func NewDBIndexerEventDispatcher(es store.EventStore) *DBIndexerEventDispatcher {
	return &DBIndexerEventDispatcher{EventStore: es}
}

func (ed *DBIndexerEventDispatcher) Send(blockHeight uint64, msg []byte) error {
	var eventData loomchain.EventData
	if err := json.Unmarshal(msg, &eventData); err != nil {
		return err
	}

	// TODO: more efficient way to persist event data
	ed.Set(PrefixBlockHightEventIndex(eventData.BlockHeight, eventData.TransactionIndex), msg)
	ed.Set(PrefixPluginName(eventData.PluginName), msg) // EVM use the address of EVM??
	for _, topic := range eventData.Topics {
		ed.Set(PrefixPluginNameTopic(eventData.PluginName, topic), msg)
	}

	log.Printf("Event emitted: index: %d, eventData: %+v\n", blockHeight, eventData)
	return fmt.Errorf("not implemented")
}

func PrefixBlockHightEventIndex(blockHeight uint64, eventIndex uint64) []byte {
	return util.PrefixKey([]byte{1}, []byte(fmt.Sprintf("%d", blockHeight)), []byte(fmt.Sprintf("%d", eventIndex)))
}

func PrefixPluginName(pluginName string) []byte {
	return util.PrefixKey([]byte{2}, []byte(pluginName))
}

func PrefixContractIDBlockHightEventIndex(contractID []byte, blockHeight uint64, eventIndex uint64) []byte {
	return util.PrefixKey([]byte{3}, contractID, []byte(fmt.Sprintf("%d", blockHeight)), []byte(fmt.Sprintf("%d", eventIndex)))
}

func PrefixPluginNameTopic(pluginName string, topic string) []byte {
	return util.PrefixKey([]byte{4}, []byte(pluginName), []byte(topic))
}
