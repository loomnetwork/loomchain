package store

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tendermint/libs/db"
)

// PruneDatabase will delete old versions of the IAVL tree from app.db.
// The numVersions parameter can be used to limit the pruning to the oldest N versions,
// useful for incrementally reclaiming disk space from nodes in a cluster without shutting any one
// of them down for too long.
func PruneDatabase(dbName, dbPath string, numVersions int64) error {
	startTime := time.Now()

	db, err := dbm.NewGoLevelDB(dbName, dbPath)
	if err != nil {
		return errors.Wrapf(err, "failed to load %s/%s", dbPath, dbName)
	}
	defer db.Close()

	stats, err := db.DB().GetProperty("leveldb.stats")
	if err != nil {
		return err
	}
	fmt.Printf("\n--- app.db stats before pruning ---\n%v------\n", stats)

	tree := iavl.NewMutableTree(db, 10000)
	latestVer, err := tree.Load()
	if err != nil {
		return errors.Wrap(err, "failed to load IAVL tree")
	}
	fmt.Printf("latest tree version %d (loaded in %v secs)\n", latestVer, time.Since(startTime).Seconds())

	startTime = time.Now()
	oldestVer := latestVer
	for i := int64(1); i < latestVer; i++ {
		if tree.VersionExists(i) {
			oldestVer = i
			break
		}
	}
	fmt.Printf("oldest tree version %d (found in %v secs)\n", oldestVer, time.Since(startTime).Seconds())

	maxVer := oldestVer + numVersions
	if (numVersions == 0) || (maxVer >= latestVer) {
		maxVer = latestVer - 2
	}
	if oldestVer > maxVer {
		fmt.Print("nothing to prune\n")
		return nil
	}
	fmt.Printf("pruning from version %d to %d\n", oldestVer, maxVer)

	startTime = time.Now()
	for i := oldestVer; i <= maxVer; i++ {
		if tree.VersionExists(i) {
			if err := tree.DeleteVersion(i); err != nil {
				return errors.Wrapf(err, "deleting tree version %d", i)
			}
		}
	}

	fmt.Printf("pruning complete (took %v mins)\n", time.Since(startTime).Minutes())

	stats, err = db.DB().GetProperty("leveldb.stats")
	if err != nil {
		return err
	}
	fmt.Printf("\n--- app.db stats after pruning ---\n%v------\n", stats)
	return nil
}

func CompactDatabase(dbName, dbPath string) error {
	db, err := dbm.NewGoLevelDB(dbName, dbPath)
	if err != nil {
		return errors.Wrapf(err, "failed to load %s/%s", dbPath, dbName)
	}
	defer db.Close()

	stats, err := db.DB().GetProperty("leveldb.stats")
	if err != nil {
		return err
	}
	fmt.Printf("--- app.db stats before compacting ---\n%v------\n", stats)

	if err := db.DB().CompactRange(util.Range{}); err != nil {
		return errors.Wrap(err, "failed to compact db")
	}

	stats, err = db.DB().GetProperty("leveldb.stats")
	if err != nil {
		return err
	}
	fmt.Printf("--- app.db stats after compacting ---\n%v------\n", stats)
	return nil
}
