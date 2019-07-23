package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitValidators(t *testing.T) {
	validators, altValidators, err := splitValidators(0, 2, 3)
	require.NoError(t, err)
	require.Equal(t, uint64(0), validators)
	require.Equal(t, uint64(0), altValidators)

	validators, altValidators, err = splitValidators(1, 2, 3)
	require.NoError(t, err)
	require.Equal(t, uint64(1), validators)
	require.Equal(t, uint64(0), altValidators)

	validators, altValidators, err = splitValidators(4, 2, 2)
	require.NoError(t, err)
	require.Equal(t, uint64(2), validators)
	require.Equal(t, uint64(2), altValidators)

	validators, altValidators, err = splitValidators(6, 2, 2)
	require.NoError(t, err)
	require.Equal(t, uint64(4), validators)
	require.Equal(t, uint64(2), altValidators)

	validators, altValidators, err = splitValidators(3, 2, 2)
	require.NoError(t, err)
	require.Equal(t, uint64(2), validators)
	require.Equal(t, uint64(1), altValidators)
}
