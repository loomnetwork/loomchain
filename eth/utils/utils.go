// +build evm

package utils

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/loomnetwork/go-loom"
	"strconv"
)

const (
	SolidtyMaxTopics = 4
)

func UnmarshalEthFilter(query []byte) (EthFilter, error) {
	var filter struct {
		FromBlock string        `json:"fromBlock"`
		ToBlock   string        `json:"toBlock"`
		Address   string        `json:"address"`
		Topics    []interface{} `json:"topics"`
	}
	json.Unmarshal(query, &filter)

	rFilter := EthFilter{
		FromBlock: filter.FromBlock,
		ToBlock:   filter.ToBlock,
	}

	if len(filter.Address) > 0 {
		address, err := loom.LocalAddressFromHexString(filter.Address)
		if err != nil {
			return EthFilter{}, fmt.Errorf("invalid ethfilter, address")
		}
		rFilter.Addresses = append(rFilter.Addresses, address)
	}

	if len(filter.Topics) > SolidtyMaxTopics {
		return EthFilter{}, fmt.Errorf("invalid ethfilter, too many topics")
	}
	for _, topicUT := range filter.Topics {
		switch topic := topicUT.(type) {
		case string:
			rFilter.Topics = append(rFilter.Topics, []string{topic})
		case nil:
			rFilter.Topics = append(rFilter.Topics, nil)
		case []interface{}:
			topicPairUT := topicUT.([]interface{})
			var topics []string
			for _, topicUT := range topicPairUT {
				switch topic := topicUT.(type) {
				case string:
					topics = append(topics, topic)
				default:
					return EthFilter{}, fmt.Errorf("invalid ethfilter, unreconised topic type")
				}
			}
			rFilter.Topics = append(rFilter.Topics, topics)
		default:
			return EthFilter{}, fmt.Errorf("invalid ethfilter, unrecognised topic")
		}
	}

	return rFilter, nil
}

func BlockNumber(blockTag string, height uint64) (uint64, error) {
	var block uint64
	switch blockTag {
	case "":
		block = height - 1
	case "latest":
		block = height - 1
	case "pending":
		block = height
	case "earliest":
		return uint64(1), nil
	default:
		var err error
		block, err = strconv.ParseUint(blockTag, 0, 64)
		if err != nil {
			return block, err
		}
	}
	if block < 1 {
		block = 1
	}
	return block, nil
}

func BlockHeightToBytes(height uint64) []byte {
	heightB := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightB, height)
	return heightB
}

func GetId() string {
	return string(rpc.NewID())
}
