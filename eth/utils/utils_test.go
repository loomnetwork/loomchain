// +build evm

package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	addr1      = "0x8888f1f195afa192cfee860698584c030f4c9db1"
	addr2      = "0x5ea31a5614f66526b00b9858b4db079178b5aea2"
	testFilter = "{\"fromBlock\":\"0x1\",\"toBlock\":\"0x2\",\"addresses\":[\"" + addr1 + "\",\"" + addr2 + "\"],\"topics\":[\"0x000000000000000000000000a94f5374fce5edbc8e2a8697c15331677e6ebf0b\",null,[\"0x000000000000000000000000a94f5374fce5edbc8e2a8697c15331677e6ebf0b\",\"0x0000000000000000000000000aff3454fce5edbc8cca8697c15331677e6ebccc\"]]}"
	allFilter  = "{\"fromBlock\":\"0x0\",\"toBlock\":\"latest\",\"addresses\":\"\",\"topics\":[]}"
	test1      = "{\"fromBlock\":\"0x1\"}"
)

func TestEthUnmarshal(t *testing.T) {
	ethFilter, err := UnmarshalEthFilter([]byte(testFilter))
	require.NoError(t, err, "un-marshalling test filter")
	require.Equal(t, addr1, strings.ToLower(ethFilter.Addresses[0].String()))
	require.Equal(t, addr2, strings.ToLower(ethFilter.Addresses[1].String()))
	_, err = UnmarshalEthFilter([]byte(allFilter))
	require.NoError(t, err, "un-marshalling test filter")
	_, err = UnmarshalEthFilter([]byte(test1))
	require.NoError(t, err, "un-marshalling test filter")
}
