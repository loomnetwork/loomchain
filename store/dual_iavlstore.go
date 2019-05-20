package store

import (
	"github.com/pkg/errors"
	"github.com/tendermint/iavl"
	dbtm "github.com/tendermint/tendermint/libs/db"
)

type DualIavlStore struct {
	IAVLStore
	diskSaveFrequency uint64
	saveCount         uint64
	dualDb            *dualMemDb
}

func NewDualIavlStore(db dbtm.DB, memoryVersions, diskSaveFrequency, targetVersion int64) (*DualIavlStore, error) {
	dualDb := newDualMemDb(db)
	tree := iavl.NewMutableTree(dualDb, 10000)
	_, err := tree.LoadVersion(targetVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "load iavl tree version %v", targetVersion)
	}

	dualIavlStore := &DualIavlStore{
		IAVLStore: IAVLStore{
			tree:        tree,
			maxVersions: memoryVersions,
		},
		diskSaveFrequency: uint64(diskSaveFrequency),
		saveCount:         0,
		dualDb:            dualDb,
	}
	return dualIavlStore, nil
}

func (ds *DualIavlStore) SaveVersion() ([]byte, int64, error) {
	hash, version, err := ds.IAVLStore.SaveVersion()
	if err != nil {
		return hash, version, err
	}
	ds.saveCount++
	if ds.saveCount%ds.diskSaveFrequency == 0 {
		ds.dualDb.writeToDisk()
	}
	return hash, version, err
}
