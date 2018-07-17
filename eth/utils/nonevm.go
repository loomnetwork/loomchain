// +build !evm

package utils

import (
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
