package store

import (
	"testing"
	"time"

	"github.com/loomnetwork/loomchain/log"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/db"
)

const (
	lcAppDir          = "/home/piers/go/src/github.com/loomnetwork/loomchain"
	pdAppDir          = "/media/piers/3CA744637F048E2E/appDbtest"
	diskSaveFrequency = 3
)

var (
	tests = [][]struct {
		key   string
		value string
	}{
		{
			{"1", "one"},
			{"11", "one one"},
		},
		{
			{"2", "two"},
			{"22", "two two"},
		},
		{
			{"3", "three"},
			{"33", "three three"},
		},
		{
			{"4", "four"},
			{"44", "four four"},
		},
		{
			{"5", "five"},
			{"55", "five five"},
		},
		{
			{"6", "six"},
			{"66", "six six"},
		},
		{
			{"7", "seven"},
			{"77", "seven seven"},
		},
		{
			{"8", "eight"},
			{"88", "eight eight"},
		},
		{
			{"9", "nine"},
			{"99", "nine nine"},
		},
		{
			{"10", "ten"},
			{"1010", "ten ten"},
		},
	}
)

func TestDualIavlStore(t *testing.T) {
	log.Setup("debug", "file://-")
	log.Root.With("module", "dual-iavlstore")

	diskDb := db.NewMemDB()
	appDb, err := NewIAVLStore(diskDb, 0, 0)
	store, err := NewDualIavlStore(diskDb, 10, diskSaveFrequency, 0)
	require.NoError(t, err)

	for index, testVersion := range tests {
		for _, test := range testVersion {
			store.Set([]byte(test.key), []byte(test.value))
		}
		_, version, err := store.SaveVersion()
		require.NoError(t, err)
		for _, test := range testVersion {
			require.True(t, store.Has([]byte(test.key)))
		}

		_, err = appDb.tree.Load()
		require.NoError(t, err)
		for i := 1; i <= index; i++ {
			updated := int64(i)/diskSaveFrequency < version/diskSaveFrequency || version%diskSaveFrequency == 0
			for _, test := range tests[i] {
				require.Equal(t, updated, appDb.Has([]byte(test.key)))
			}
		}
	}
}

func TestCopyIAVL(t *testing.T) {
	t.Skip()
	appDb, err := db.NewGoLevelDBWithOpts("app", lcAppDir, nil)
	//appDb, err := db.NewGoLevelDBWithOpts("app", pdAppDir, nil)
	require.NoError(t, err)
	require.NotNil(t, appDb)
	log.Info("appdb---------------------")
	log.Info("appdb stats", appDb.Stats())
	//appDb.Print()
	memDb := db.NewMemDB()
	{
		startTime := time.Now()

		for iter := appDb.Iterator(nil, nil); iter.Valid(); iter.Next() {
			memDb.Set(iter.Key(), iter.Value())
		}

		now := time.Now()
		elapsed := now.Sub(startTime)
		log.Info("Finished reeading in database, time taken %v seconds", elapsed)

		log.Info("memdb----------------------")
		//memDb.Print()
		memDb.Stats()
	}
	appDb.Close()
	{
		appDbNew, err := db.NewGoLevelDBWithOpts("newapp", pdAppDir, nil)
		require.NotNil(t, appDbNew)
		require.NoError(t, err)
		startTime := time.Now()
		for iter := memDb.Iterator(nil, nil); iter.Valid(); iter.Next() {
			appDbNew.Set(iter.Key(), iter.Value())
		}
		now := time.Now()
		elapsed := now.Sub(startTime)
		log.Info("Finished reeading in database, time taken %v seconds", elapsed)
		appDbNew.Stats()
	}
}
