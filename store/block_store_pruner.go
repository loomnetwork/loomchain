package store

import (
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/loomnetwork/loomchain/log"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/tendermint/tendermint/blockchain"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var (
	calcBlockMetaPrefix = []byte("H:")
	blockStoreKey       = []byte("blockStore")
)

func calcBlockMetaKey(height int64) []byte {
	return []byte(fmt.Sprintf("H:%v", height))
}

func calcBlockPartKey(height int64, partIndex int) []byte {
	return []byte(fmt.Sprintf("P:%v:%v", height, partIndex))
}

func calcBlockCommitKey(height int64) []byte {
	return []byte(fmt.Sprintf("C:%v", height))
}

func calcSeenCommitKey(height int64) []byte {
	return []byte(fmt.Sprintf("SC:%v", height))
}

// PruneBlockStore makes a backup of blockstore.db, and replaces it with a new blockstore.db containing
// only the most recent blocks. Old blockstore.db backups will not be removed automatically, it's
// up to the node operator to periodically remove those backups.
// The source DB path should specify the parent directory within which blockstore.db can be found.
// The minimum height should specify the oldest block that must be retained in the new blockstore.db,
// it should be set to match the app height to ensure that the app can start processing blocks from
// that height.
func PruneBlockStore(srcDBPath string, cfg *BlockStoreConfig, minHeight int64) error {
	srcDB, err := dbm.NewGoLevelDBWithOpts("blockstore", srcDBPath, &opt.Options{ReadOnly: true})
	if err != nil {
		return err
	}
	defer srcDB.Close()

	srcBlockStore := blockchain.NewBlockStore(srcDB)

	destDB := dbm.NewDB("pruned_blockstore", "leveldb", srcDBPath)
	defer destDB.Close()

	latestHeight := blockchain.LoadBlockStoreStateJSON(srcDB).Height
	var targetHeight int64
	targetHeight = latestHeight - cfg.NumBlocksToRetain
	if minHeight > targetHeight {
		targetHeight = minHeight
	}
	graceBlocks := (cfg.PruneGraceFactor / 100) * cfg.NumBlocksToRetain
	oldestHeight := int64(-1)
	// Find the oldest block
	it := srcDB.Iterator(calcBlockMetaPrefix, prefixRangeEnd(calcBlockMetaPrefix))
	for ; it.Valid(); it.Next() {
		oldestBlockMetaKey := it.Key()
		oldestHeight = getHeightFromKey(oldestBlockMetaKey)
		break
	}
	// Number of blocks to prune is less than grace blocks then skip pruning
	if ((targetHeight - oldestHeight) + 1) < graceBlocks {
		return nil
	}
	// If minimum height is greater than target height, there are no blocks to prune, skip pruning
	if oldestHeight >= targetHeight {
		return nil
	}

	log.Info("[Block Store Pruner] Copying blocks", "fromHeight", targetHeight, "toHeight", latestHeight)

	batch := destDB.NewBatch()
	numBlocksWritten := int64(0)
	for height := targetHeight; height <= latestHeight; height++ {
		// skip if block metadata is not found
		if !srcDB.Has(calcBlockMetaKey(height)) {
			log.Info("[Block Store Pruner] Missing block", "height", height)
			continue
		}

		meta := srcBlockStore.LoadBlockMeta(height)
		for i := 0; i < meta.BlockID.PartsHeader.Total; i++ {
			blockPartKey := calcBlockPartKey(height, i)
			batch.Set(blockPartKey, srcDB.Get(blockPartKey))
		}

		blockMetaKey := calcBlockMetaKey(height)
		blockCommitKey := calcBlockCommitKey(height - 1)
		SeenCommitKey := calcSeenCommitKey(height)

		batch.Set(blockMetaKey, srcDB.Get(blockMetaKey))
		batch.Set(blockCommitKey, srcDB.Get(blockCommitKey))
		batch.Set(SeenCommitKey, srcDB.Get(SeenCommitKey))

		if numBlocksWritten%cfg.BatchSize == 0 {
			batch.Write()
			batch = destDB.NewBatch()
		}
		numBlocksWritten++
	}
	batch.Set(blockStoreKey, srcDB.Get(blockStoreKey))
	batch.WriteSync()

	srcDB.Close()
	destDB.Close()

	log.Info("[Block Store Pruner] Finished copying blocks", "count", numBlocksWritten)

	// Rename original blockstore to blockstore.db.bak{N}
	backupPath := getBackupDBPath(srcDBPath)
	if err := os.Rename(path.Join(srcDBPath, "blockstore.db"), backupPath); err != nil {
		return err
	}
	log.Info("[Block Store Pruner] Backed up block store", "path", backupPath)

	// Rename pruned blockstore to blockstore.db
	return os.Rename(
		path.Join(srcDBPath, "pruned_blockstore.db"),
		path.Join(srcDBPath, "blockstore.db"),
	)
}

func getBackupDBPath(dbDir string) string {
	backupPath := path.Join(dbDir, "blockstore.db.bak")
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return backupPath
	}

	for i := 1; i < int(math.MaxInt16); i++ {
		altPath := fmt.Sprintf("%s%d", backupPath, i)
		if _, err := os.Stat(altPath); os.IsNotExist(err) {
			return altPath
		}
	}

	panic("failed to generate unique name for blockstore.db backup")
}

func getHeightFromKey(key []byte) int64 {
	val := strings.Split(string(key), ":")
	if len(val) > 1 {
		height, err := strconv.ParseInt(val[1], 10, 64)
		if err != nil {
			return 0
		}
		return height
	}
	return 0
}
