// +build evm

package gateway

import (
	"testing"
	"time"

	loom "github.com/loomnetwork/go-loom"
	"github.com/stretchr/testify/require"
)

func TestRecentHashPool(t *testing.T) {
	recentHashPool := newRecentHashPool(4 * time.Second)
	recentHashPool.startCleanupRoutine()

	require.True(t, recentHashPool.addHash([]byte{1, 2, 3}), "adding hash for first time should succed")

	require.False(t, recentHashPool.addHash([]byte{1, 2, 3}), "adding duplicate hash shouldnt be allowed")

	time.Sleep(5 * time.Second)

	require.True(t, recentHashPool.addHash([]byte{1, 2, 3}), "after timeout, hash should be allowed")
}

func TestTransferGatewayOracleMainnetEventSort(t *testing.T) {
	events := []*mainnetEventInfo{
		&mainnetEventInfo{BlockNum: 5, TxIdx: 0},
		&mainnetEventInfo{BlockNum: 5, TxIdx: 1},
		&mainnetEventInfo{BlockNum: 5, TxIdx: 4},
		&mainnetEventInfo{BlockNum: 3, TxIdx: 3},
		&mainnetEventInfo{BlockNum: 3, TxIdx: 7},
		&mainnetEventInfo{BlockNum: 3, TxIdx: 1},
		&mainnetEventInfo{BlockNum: 8, TxIdx: 4},
		&mainnetEventInfo{BlockNum: 8, TxIdx: 1},
		&mainnetEventInfo{BlockNum: 9, TxIdx: 0},
		&mainnetEventInfo{BlockNum: 10, TxIdx: 5},
		&mainnetEventInfo{BlockNum: 1, TxIdx: 2},
	}
	sortedEvents := []*mainnetEventInfo{
		&mainnetEventInfo{BlockNum: 1, TxIdx: 2},
		&mainnetEventInfo{BlockNum: 3, TxIdx: 1},
		&mainnetEventInfo{BlockNum: 3, TxIdx: 3},
		&mainnetEventInfo{BlockNum: 3, TxIdx: 7},
		&mainnetEventInfo{BlockNum: 5, TxIdx: 0},
		&mainnetEventInfo{BlockNum: 5, TxIdx: 1},
		&mainnetEventInfo{BlockNum: 5, TxIdx: 4},
		&mainnetEventInfo{BlockNum: 8, TxIdx: 1},
		&mainnetEventInfo{BlockNum: 8, TxIdx: 4},
		&mainnetEventInfo{BlockNum: 9, TxIdx: 0},
		&mainnetEventInfo{BlockNum: 10, TxIdx: 5},
	}
	sortMainnetEvents(events)
	require.EqualValues(t, sortedEvents, events, "wrong sort order")
}

func TestTransferGatewayOracleConfigWithdrawerAddressBlacklist(t *testing.T) {
	cfg := DefaultConfig(8888)
	addr1 := loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 := loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c5")
	cfg.WithdrawerAddressBlacklist = []string{
		addr1.String(),
		addr2.String(),
	}
	blacklist, err := cfg.GetWithdrawerAddressBlacklist()
	require.NoError(t, err)
	require.Equal(t, 2, len(blacklist))
	require.Equal(t, 0, addr1.Compare(blacklist[0]))
	require.Equal(t, 0, addr2.Compare(blacklist[1]))
}
