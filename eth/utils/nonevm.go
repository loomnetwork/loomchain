// +build !evm

package utils

import (
	"encoding/binary"

	ptypes "github.com/loomnetwork/go-loom/plugin/types"
)

func GetId() string {
	return ""
}

func UnmarshalEthFilter(query []byte) (EthFilter, error) {
	return EthFilter{}, nil
}

func MatchEthFilter(filter EthBlockFilter, eventLog ptypes.EventData) bool {
	return true
}

func BlockHeightToBytes(height uint64) []byte {
	heightB := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightB, height)
	return heightB
}
