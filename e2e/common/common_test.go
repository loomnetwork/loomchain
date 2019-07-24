package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitValidators(t *testing.T) {
	validators, altValidators := splitValidators(0, 2, 3)
	require.Equal(t, uint64(0), validators)
	require.Equal(t, uint64(0), altValidators)

	validators, altValidators = splitValidators(1, 2, 3)
	require.Equal(t, uint64(1), validators)
	require.Equal(t, uint64(0), altValidators)

	validators, altValidators = splitValidators(4, 2, 2)
	require.Equal(t, uint64(2), validators)
	require.Equal(t, uint64(2), altValidators)

	validators, altValidators = splitValidators(6, 2, 2)
	require.Equal(t, uint64(4), validators)
	require.Equal(t, uint64(2), altValidators)

	validators, altValidators = splitValidators(3, 2, 2)
	require.Equal(t, uint64(2), validators)
	require.Equal(t, uint64(1), altValidators)

	validators, altValidators = splitValidators(3, 1, 3)
	require.Equal(t, uint64(1), validators)
	require.Equal(t, uint64(2), altValidators)
}
