// +build evm

package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testFilter = "{\"fromBlock\":\"0x1\",\"toBlock\":\"0x2\",\"address\":\"0x8888f1f195afa192cfee860698584c030f4c9db1\",\"topics\":[\"0x000000000000000000000000a94f5374fce5edbc8e2a8697c15331677e6ebf0b\",null,[\"0x000000000000000000000000a94f5374fce5edbc8e2a8697c15331677e6ebf0b\",\"0x0000000000000000000000000aff3454fce5edbc8cca8697c15331677e6ebccc\"]]}"
	allFilter  = "{\"fromBlock\":\"0x0\",\"toBlock\":\"latest\",\"address\":\"\",\"topics\":[]}"
	test1      = "{\"fromBlock\":\"0x1\"}"
)

func TestEthUnmarshal(t *testing.T) {
	_, err := UnmarshalEthFilter([]byte(testFilter))
	require.NoError(t, err, "un-marshalling test filter")
	_, err = UnmarshalEthFilter([]byte(allFilter))
	require.NoError(t, err, "un-marshalling test filter")
	_, err = UnmarshalEthFilter([]byte(test1))
	require.NoError(t, err, "un-marshalling test filter")
}

func TestBlockNumber(t *testing.T) {
	const height = uint64(50)

	block, err := BlockNumber("23", height)
	require.NoError(t, err)
	require.Equal(t, block, uint64(23))

	block, err = BlockNumber("0x17", height)
	require.NoError(t, err)
	require.Equal(t, block, uint64(23))

	block, err = BlockNumber("latest", height)
	require.NoError(t, err)
	require.Equal(t, block, height-1)

	block, err = BlockNumber("earliest", height)
	require.NoError(t, err)
	require.Equal(t, block, uint64(1))

	block, err = BlockNumber("pending", height)
	require.NoError(t, err)
	require.Equal(t, block, height)

	_, err = BlockNumber("nonsense", height)
	require.Error(t, err)
}
