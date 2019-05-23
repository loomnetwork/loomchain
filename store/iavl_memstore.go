package store

import (
	"github.com/tendermint/tendermint/libs/db"
)

type DelayIavlStore struct {
	IAVLStore
}

func NewDelayIavlStore(diskDb db.DB, maxVersions, targetVersion int64, saveFrequency uint64) (*IAVLStore, error) {
	difDb := &differenceDb{diskDb}
	return NewIAVLStore(difDb, maxVersions, targetVersion, saveFrequency)

}
