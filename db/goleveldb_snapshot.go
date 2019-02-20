package db

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	dbm "github.com/tendermint/tendermint/libs/db"
)

type GoLevelDBSnapshot struct {
	*leveldb.Snapshot
}

var _ Snapshot = &GoLevelDBSnapshot{}

func (s *GoLevelDBSnapshot) Get(key []byte) []byte {
	if key == nil {
		key = []byte{}
	}
	val, err := s.Snapshot.Get(key, nil)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil
		}
		panic(err)
	}
	return val
}

func (s *GoLevelDBSnapshot) Has(key []byte) bool {
	return s.Get(key) != nil
}

func (s *GoLevelDBSnapshot) NewIterator(start, end []byte) dbm.Iterator {
	// TODO: Creating an iterator over the entire DB when start and/or end isn't nil seems
	//       sub-optimal, maybe worth rewriting the TM iterator so don't have to give it an overly
	//       broad source iterator.
	return dbm.NewGoLevelDBIterator(s.Snapshot.NewIterator(nil, nil), start, end, false)
}

func (s *GoLevelDBSnapshot) Release() {
	s.Snapshot.Release()
}
