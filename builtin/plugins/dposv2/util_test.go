package dposv2

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalculateTierLockTime(t *testing.T) {
	electionCycleLength := uint64(60)

	tier0 := TierMap[0]
	tier1 := TierMap[1]
	tier2 := TierMap[2]
	tier3 := TierMap[3]

	assert.Equal(t, electionCycleLength, calculateTierLocktime(tier0, electionCycleLength))
	assert.NotEqual(t, TierLocktimeMap[0], calculateTierLocktime(tier0, electionCycleLength))
	assert.Equal(t, TierLocktimeMap[1], calculateTierLocktime(tier1, electionCycleLength))
	assert.Equal(t, TierLocktimeMap[2], calculateTierLocktime(tier2, electionCycleLength))
	assert.Equal(t, TierLocktimeMap[3], calculateTierLocktime(tier3, electionCycleLength))

	electionCycleLength = uint64(1209600) // Election cycle length = TierLockTimeMap[0] = 2 weeks

	assert.Equal(t, electionCycleLength, calculateTierLocktime(tier0, electionCycleLength))
	assert.Equal(t, TierLocktimeMap[0], calculateTierLocktime(tier0, electionCycleLength))
	assert.Equal(t, TierLocktimeMap[1], calculateTierLocktime(tier1, electionCycleLength))
	assert.Equal(t, TierLocktimeMap[2], calculateTierLocktime(tier2, electionCycleLength))
	assert.Equal(t, TierLocktimeMap[3], calculateTierLocktime(tier3, electionCycleLength))

	electionCycleLength = uint64(1209601) // For an election cycle larger than 2 weeks, the locktime is 2 weeks.

	assert.NotEqual(t, electionCycleLength, calculateTierLocktime(tier0, electionCycleLength))
	assert.Equal(t, TierLocktimeMap[0], calculateTierLocktime(tier0, electionCycleLength))
	assert.Equal(t, TierLocktimeMap[1], calculateTierLocktime(tier1, electionCycleLength))
	assert.Equal(t, TierLocktimeMap[2], calculateTierLocktime(tier2, electionCycleLength))
	assert.Equal(t, TierLocktimeMap[3], calculateTierLocktime(tier3, electionCycleLength))
}
