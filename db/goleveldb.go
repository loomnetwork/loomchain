package db

import (
	"fmt"
	"github.com/loomnetwork/loomchain/db/metrics"
	"github.com/loomnetwork/loomchain/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	dbm "github.com/tendermint/tendermint/libs/db"
)

type GoLevelDB struct {
	*dbm.GoLevelDB
}

var _ DBWrapper = &GoLevelDB{}

func (g *GoLevelDB) Compact() error {
	return g.DB().CompactRange(util.Range{})
}

func (g *GoLevelDB) GetSnapshot() Snapshot {
	snap, err := g.DB().GetSnapshot()
	if err != nil {
		panic(err)
	}
	return &GoLevelDBSnapshot{
		Snapshot: snap,
	}
}

func LoadGoLevelDB(name, dir string, cacheSizeMeg int) (*GoLevelDB, error) {
	o := &opt.Options{
		BlockCacheCapacity: cacheSizeMeg * opt.MiB,
	}
	db, err := dbm.NewGoLevelDBWithOpts(name, dir, o)
	if err != nil {
		return nil, err
	}
	err = prometheus.Register(metrics.NewStatsCollector(fmt.Sprintf("goleveldb_%s", name), log.Default, db))
	if err != nil {
		log.Error("Registration error", "err", err)
	}

	return &GoLevelDB{GoLevelDB: db}, nil
}
