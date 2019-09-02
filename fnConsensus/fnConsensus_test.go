package fnConsensus

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbm "github.com/tendermint/tendermint/libs/db"
)

func TestUnmarshalReactorState(t *testing.T) {
	db := dbm.NewMemDB()
	rs := NewReactorState()

	rsByte, err := rs.Marshal()
	require.NoError(t, err)
	err = rs.Unmarshal(rsByte)
	require.NoError(t, err)
	require.NotNil(t, rs.Messages)

	err = saveReactorState(db, rs, false)
	require.NoError(t, err)

	rs, err = loadReactorState(db)
	require.NoError(t, err)
	require.NotNil(t, rs.Messages)
}
