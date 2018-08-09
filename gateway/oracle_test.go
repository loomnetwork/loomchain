// +build evm

package gateway

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
