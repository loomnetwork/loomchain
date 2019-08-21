package eth

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlockNumber(t *testing.T) {
	const height = int64(50)

	block, err := DecBlockHeight(height, "23")
	require.NoError(t, err)
	require.Equal(t, block, uint64(23))

	block, err = DecBlockHeight(height, "0x17")
	require.NoError(t, err)
	require.Equal(t, block, uint64(23))

	block, err = DecBlockHeight(height, "latest")
	require.NoError(t, err)
	require.Equal(t, block, uint64(height))

	block, err = DecBlockHeight(height, "earliest")
	require.NoError(t, err)
	require.Equal(t, block, uint64(1))

	block, err = DecBlockHeight(height, "pending")
	require.NoError(t, err)
	require.Equal(t, block, uint64(height+1))

	_, err = DecBlockHeight(height, "nonsense")
	require.Error(t, err)
}

func TestEncBigInt(t *testing.T) {
	big0 := big.NewInt(0)
	require.Equal(t, Quantity("0x0"), EncBigInt(*big0))

	big37 := big.NewInt(37)
	require.Equal(t, Quantity("0x25"), EncBigInt(*big37))

	bigMaxInt := big.NewInt(math.MaxInt64)
	require.Equal(t, Quantity("0x7fffffffffffffff"), EncBigInt(*bigMaxInt))

	var bigLots big.Int
	bigLots.Mul(bigMaxInt, bigMaxInt)
	require.Equal(t, Quantity("0x3fffffffffffffff0000000000000001"), EncBigInt(bigLots))
}