// +build !evm

package utils

import (
	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/rpc/eth"
)

func GetId() string {
	return ""
}

func UnmarshalEthFilter(_ []byte) (eth.EthFilter, error) {
	return eth.EthFilter{}, nil
}

func MatchEthFilter(_ eth.EthBlockFilter, _ ptypes.EventData) bool {
	return true
}
