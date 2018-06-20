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
	FilterMaxOrTopics = 2
	SolidtyMaxTopics  = 4
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
			if len(topicPairUT) != FilterMaxOrTopics {
				return EthFilter{}, fmt.Errorf("invalid ethfilter, can only OR two topics")
			}
			var topic1, topic2 string
			switch topic := topicPairUT[0].(type) {
			case string:
				topic1 = string(topic)
			default:
				return EthFilter{}, fmt.Errorf("invalid ethfilter, unreconised topic pair")
			}
			switch topic := topicPairUT[1].(type) {
			case string:
				topic2 = string(topic)
			default:
				return EthFilter{}, fmt.Errorf("invalid ethfilter, unreconised topic pair")
			}
			rFilter.Topics = append(rFilter.Topics, []string{topic1, topic2})
		default:
			return EthFilter{}, fmt.Errorf("invalid ethfilter, unrecognised topic")
		}
	}

	return rFilter, nil
}

func BlockNumber(bockTag string, height uint64) (uint64, error) {
	var block uint64
	switch bockTag {
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
		block, err = strconv.ParseUint(bockTag, 0, 64)
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
