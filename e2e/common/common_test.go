package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitValidators(t *testing.T) {
	validators, altValidators := splitValidators(0)
	require.Equal(t, uint64(0), validators)
	require.Equal(t, uint64(0), altValidators)

	validators, altValidators = splitValidators(1)
	require.Equal(t, uint64(1), validators)
	require.Equal(t, uint64(0), altValidators)

	validators, altValidators = splitValidators(4)
	require.Equal(t, uint64(3), validators)
	require.Equal(t, uint64(1), altValidators)

	validators, altValidators = splitValidators(6)
	require.Equal(t, uint64(5), validators)
	require.Equal(t, uint64(1), altValidators)

	validators, altValidators = splitValidators(3)
	require.Equal(t, uint64(2), validators)
	require.Equal(t, uint64(1), altValidators)

	validators, altValidators = splitValidators(30)
	require.Equal(t, uint64(29), validators)
	require.Equal(t, uint64(1), altValidators)
}

func TestDoCheckAppHash(t *testing.T) {
	require.True(t, doCheckAppHash(true, 3, 1))
	require.True(t, doCheckAppHash(true, 4, 1))
	require.True(t, doCheckAppHash(true, 5, 1))
	require.True(t, doCheckAppHash(true, 6, 1))
	require.True(t, doCheckAppHash(true, 6, 2))

	require.False(t, doCheckAppHash(true, 1, 1))
	require.False(t, doCheckAppHash(true, 1, 2))
	require.False(t, doCheckAppHash(true, 2, 1))
	require.False(t, doCheckAppHash(true, 2, 2))
	require.False(t, doCheckAppHash(true, 3, 2))
	require.False(t, doCheckAppHash(true, 4, 2))
	require.False(t, doCheckAppHash(true, 5, 2))

	require.False(t, doCheckAppHash(false, 3, 1))
	require.False(t, doCheckAppHash(false, 3, 2))
	require.False(t, doCheckAppHash(false, 4, 1))
	require.False(t, doCheckAppHash(false, 4, 2))
	require.False(t, doCheckAppHash(false, 5, 1))
	require.False(t, doCheckAppHash(false, 5, 2))
	require.False(t, doCheckAppHash(false, 6, 1))
	require.False(t, doCheckAppHash(false, 6, 2))
}
