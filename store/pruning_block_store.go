package store

import (
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/loomnetwork/loomchain/log"
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

type PruningBlockStore struct {
	blockStoreDB       dbm.DB
	prunedBlockStoreDB dbm.DB
	*blockchain.BlockStore
	cfg       *BlockStoreConfig
	batchSize uint64
}

func NewPruningBlockStore(blockStoreConfig *BlockStoreConfig) *PruningBlockStore {
	blockStoreDB := dbm.NewDB("blockstore", "leveldb", path.Join(blockStoreConfig.ChainDataPath, "data"))
	prunedBlockStoreDB := dbm.NewDB("pruned_blockstore", "leveldb", path.Join(blockStoreConfig.ChainDataPath, "data"))
	return &PruningBlockStore{
		blockStoreDB:       blockStoreDB,
		prunedBlockStoreDB: prunedBlockStoreDB,
		BlockStore:         blockchain.NewBlockStore(blockStoreDB),
		cfg:                blockStoreConfig,
	}
}

func (bs *PruningBlockStore) Close() {
	bs.blockStoreDB.Close()
	bs.prunedBlockStoreDB.Close()
}

func (bs *PruningBlockStore) Height() int64 {
	return blockchain.LoadBlockStoreStateJSON(bs.blockStoreDB).Height
}

func (bs *PruningBlockStore) Prune() error {
	latestHeight := bs.Height()
	var targetHeight int64
	targetHeight = latestHeight - bs.cfg.NumBlocksToRetain
	graceBlocks := (bs.cfg.PruneGraceFactor / 100) * bs.cfg.NumBlocksToRetain
	oldestHeight := int64(-1)
	// Find the oldest block
	it := bs.blockStoreDB.Iterator(calcBlockMetaPrefix, prefixRangeEnd(calcBlockMetaPrefix))
	for ; it.Valid(); it.Next() {
		oldestBlockMetaKey := it.Key()
		oldestHeight = getHeightFromKey(oldestBlockMetaKey)
		break
	}
	// Number of blocks to prune is less than grace blocks then skip pruning
	if ((targetHeight - oldestHeight) + 1) < graceBlocks {
		bs.blockStoreDB.Close()
		bs.prunedBlockStoreDB.Close()
		return nil
	}
	// If minimum height is greater than target height, there are no blocks to prune, skip pruning
	if oldestHeight >= targetHeight {
		bs.blockStoreDB.Close()
		bs.prunedBlockStoreDB.Close()
		return nil
	}
	batch := bs.prunedBlockStoreDB.NewBatch()
	numHeight := int64(0)
	for height := targetHeight; height <= latestHeight; height++ {
		log.Info("Copying block at height", "height", height)
		// skip if block metadata is not found
		if !bs.blockStoreDB.Has(calcBlockMetaKey(height)) {
			log.Info("block is missing at height", "height", height)
			continue
		}

		meta := bs.LoadBlockMeta(height)
		for i := 0; i < meta.BlockID.PartsHeader.Total; i++ {
			blockPartKey := calcBlockPartKey(height, i)
			batch.Set(blockPartKey, bs.blockStoreDB.Get(blockPartKey))
		}

		blockMetaKey := calcBlockMetaKey(height)
		blockCommitKey := calcBlockCommitKey(height - 1)
		SeenCommitKey := calcSeenCommitKey(height)

		batch.Set(blockMetaKey, bs.blockStoreDB.Get(blockMetaKey))
		batch.Set(blockCommitKey, bs.blockStoreDB.Get(blockCommitKey))
		batch.Set(SeenCommitKey, bs.blockStoreDB.Get(SeenCommitKey))

		if numHeight%bs.cfg.BatchSize == 0 {
			batch.Write()
			batch = bs.prunedBlockStoreDB.NewBatch()
		}
		numHeight++
	}
	batch.Set(blockStoreKey, bs.blockStoreDB.Get(blockStoreKey))
	batch.WriteSync()
	bs.blockStoreDB.Close()
	bs.prunedBlockStoreDB.Close()

	// Rename original blockstore to blockstore.db.bak{N}
	originalBlockStore := path.Join(bs.cfg.ChainDataPath, "data/blockstore.db")
	backupBlockStore := getAvailableBackupPath(bs.cfg.ChainDataPath)
	if err := os.Rename(originalBlockStore, backupBlockStore); err != nil {
		return err
	}
	// Rename pruned blockstore to blockstore.db
	prunedBlockStore := path.Join(bs.cfg.ChainDataPath, "data/pruned_blockstore.db")
	blockStore := path.Join(bs.cfg.ChainDataPath, "data/blockstore.db")
	if err := os.Rename(prunedBlockStore, blockStore); err != nil {
		return err
	}
	return nil
}

func getAvailableBackupPath(chainDataPath string) string {
	for i := 1; i < int(math.MaxInt16); i++ {
		path := fmt.Sprintf("%s%d", path.Join(chainDataPath, "data/blockstore.db.bak"), i)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path
		}
	}
	return path.Join(chainDataPath, "data/blockstore.db.bak")
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
