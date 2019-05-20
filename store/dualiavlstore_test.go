package store

import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/db"
)

const (
	lcAppDir          = "/home/piers/go/src/github.com/loomnetwork/loomchain"
	pdAppDir          = "/media/piers/3CA744637F048E2E/appDbtest"
	diskSaveFrequency = 5
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
		}, {
			{"9", "nine"},
			{"99", "nine nine"},
		},
		{
			{"10", "eleven"},
			{"1010", "eleven eleven"},
		},
	}
)

func TestDualIavlStore(t *testing.T) {
	diskDb := db.NewMemDB()
	appDb, err := NewIAVLStore(diskDb, 0, 0)
	store, err := NewDualIavlStore(diskDb, 10, diskSaveFrequency, 0)
	require.NoError(t, err)

	for _, testVersion := range tests {
		for _, test := range testVersion {
			store.Set([]byte(test.key), []byte(test.value))
		}
		_, version, err := store.SaveVersion()
		require.NoError(t, err)
		for _, test := range testVersion {
			require.True(t, diskDb.Has([]byte(test.key)))
		}

		for v, tv := range tests {
			updated := int64(v)/diskSaveFrequency < version/diskSaveFrequency || version%diskSaveFrequency == 0
			for _, test := range tv {
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
	log.Println("appdb---------------------")
	log.Println("appdb stats", appDb.Stats())
	//appDb.Print()
	memDb := db.NewMemDB()
	{
		startTime := time.Now()

		for iter := appDb.Iterator(nil, nil); iter.Valid(); iter.Next() {
			memDb.Set(iter.Key(), iter.Value())
		}

		now := time.Now()
		elapsed := now.Sub(startTime)
		log.Printf("Finished reeading in database, time taken %v seconds", elapsed)

		log.Printf("memdb----------------------")
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
		log.Printf("Finished reeading in database, time taken %v seconds", elapsed)
		appDbNew.Stats()
	}
}
