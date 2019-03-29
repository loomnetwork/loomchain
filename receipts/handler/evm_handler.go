// +build evm

package handler

import (
	eth_types "github.com/ethereum/go-ethereum/core/types"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/rpc/eth"
)

const (
	nilData eth.Data = "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
)

func (r *ReceiptHandler) GetEventsFromLogs(logs []*eth_types.Log, blockHeight int64, caller, contract loom.Address, input []byte) []*types.EventData {
	var events []*types.EventData
	for _, log := range logs {
		var topics []string
		for _, topic := range log.Topics {
			topics = append(topics, topic.String())
		}
		eventData := &types.EventData{
			Topics: topics,
			Caller: caller.MarshalPB(),
			Address: loom.Address{
				ChainID: caller.ChainID,
				Local:   log.Address.Bytes(),
			}.MarshalPB(),
			BlockHeight:     uint64(blockHeight),
			PluginName:      contract.Local.String(),
			EncodedBody:     log.Data,
			OriginalRequest: input,
		}
		if eventData.EncodedBody == nil {
			var err error
			eventData.EncodedBody, err = eth.DecDataToBytes(nilData)
			if err != nil {
				panic("cant covert nilData to bytes")
			}
		}
		events = append(events, eventData)
	}
	return events
}
