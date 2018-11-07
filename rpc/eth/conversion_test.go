package eth

import (
	"github.com/stretchr/testify/require"
	"testing"
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
	require.Equal(t, block, uint64(height-1))

	block, err = DecBlockHeight(height, "earliest")
	require.NoError(t, err)
	require.Equal(t, block, uint64(1))

	block, err = DecBlockHeight(height, "pending")
	require.NoError(t, err)
	require.Equal(t, block, uint64(height))

	_, err = DecBlockHeight(height, "nonsense")
	require.Error(t, err)
}