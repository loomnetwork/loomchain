// +build evm

package chain

import (
	eth_types "github.com/ethereum/go-ethereum/core/types"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
)

func (r *ReceiptHandler) GetEventsFromLogs(
	logs []*eth_types.Log, blockHeight int64, caller, contract loom.Address, input []byte,
) []*types.EventData {
	var events []*types.EventData
	for _, log := range logs {
		var topics []string
		for _, topic := range log.Topics {
			topics = append(topics, topic.String())
		}
		eventData := &types.EventData{
			Topics:          topics,
			Caller:          caller.MarshalPB(),
			Address:         contract.MarshalPB(),
			BlockHeight:     uint64(blockHeight),
			PluginName:      contract.Local.String(),
			EncodedBody:     log.Data,
			OriginalRequest: input,
		}
		events = append(events, eventData)
	}
	return events
}
