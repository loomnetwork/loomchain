package store

import (
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/loomnetwork/loomchain/log"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/tendermint/tendermint/blockchain"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/types"
)

var (
	calcBlockMetaPrefix = []byte("H:")
)

type PruningBlockStore struct {
	blockStoreDB dbm.DB
	*blockchain.BlockStore
	chainDataDir string
	db           dbm.DB
	mtx          sync.RWMutex
	height       int64
}

func NewPruningBlockStore(chainDataDir string, readOnly bool, pruningAlgorithm string) *PruningBlockStore {
	var db, blockStoreDB dbm.DB
	if readOnly {
		var err error
		blockStoreDB, err = dbm.NewGoLevelDBWithOpts(
			"blockstore", path.Join(chainDataDir, "data"),
			&opt.Options{
				ReadOnly: true,
			},
		)
		if err != nil {
			panic("failed to load block store")
		}
	} else {
		blockStoreDB = dbm.NewDB("blockstore", "leveldb", path.Join(chainDataDir, "data"))
	}
	if strings.EqualFold(pruningAlgorithm, "Copy") {
		db = dbm.NewDB("blockstorev1", "leveldb", path.Join(chainDataDir, "data"))
	}
	return &PruningBlockStore{
		blockStoreDB: blockStoreDB,
		db:           db,
		BlockStore:   blockchain.NewBlockStore(blockStoreDB),
		chainDataDir: chainDataDir,
	}
}

func (bs *PruningBlockStore) Close() {
	bs.blockStoreDB.Close()
}

func (bs *PruningBlockStore) Has(key []byte) bool {
	return bs.blockStoreDB.Has(key)
}

func (bs *PruningBlockStore) Height() int64 {
	return blockchain.LoadBlockStoreStateJSON(bs.blockStoreDB).Height
}

func (bs *PruningBlockStore) LoadBlock(height int64) *types.Block {
	return bs.BlockStore.LoadBlock(height)
}

func (bs *PruningBlockStore) LoadSeenCommit(height int64) *types.Commit {
	return bs.BlockStore.LoadSeenCommit(height)
}

func (bs *PruningBlockStore) SaveSeenCommit(blockHeight int64, newSeenCommit *types.Commit) error {
	binary, err := cdc.MarshalBinaryBare(newSeenCommit)
	if err != nil {
		return err
	}

	bs.blockStoreDB.Set(calcSeenCommitKey(blockHeight), binary)
	return nil
}

//This Pruning Algorithm works by removing any blocks in the block store below the target height and original database is backed up
func (bs *PruningBlockStore) PruneviaDeletion(chainDataDir string, numBlocksToRetain, pruneGraceFactor, batchSize, logLevel int64, skipMissing, skipCompaction bool) error {
	srcblockstorePath := path.Join(chainDataDir, "data/blockstore.db")
	destblockstorePath := path.Join(chainDataDir, "data/blockstore.db.bak")
	Dir(srcblockstorePath, destblockstorePath)
	latestHeight := bs.Height()
	var targetHeight int64
	targetHeight = latestHeight - numBlocksToRetain
	//Compute grace blocks by using pruneGraceFactor
	graceBlocks := (pruneGraceFactor / 100) * numBlocksToRetain
	oldestHeight := int64(-1)
	// find the oldest block
	it := bs.blockStoreDB.Iterator(calcBlockMetaPrefix, prefixRangeEnd(calcBlockMetaPrefix))
	for ; it.Valid(); it.Next() {
		oldestBlockMetaKey := it.Key()
		oldestHeight = getHeightFromKey(oldestBlockMetaKey)
		break
	}
	//Number of blocks to prune is less than grace blocks then skip pruning
	if ((targetHeight - oldestHeight) + 1) < graceBlocks {
		bs.blockStoreDB.Close()
		bs.db.Close()
		return nil
	}
	//Minimum Height is greater than Target Height so there are no blocks to prune
	if oldestHeight >= targetHeight {
		bs.blockStoreDB.Close()
		bs.db.Close()
		return fmt.Errorf("no block below block %d", targetHeight)
	}
	var progressInterval int64
	if logLevel > 0 {
		progressInterval = int64(targetHeight / int64(math.Pow(10, float64(logLevel))))
	}
	batch := bs.blockStoreDB.NewBatch()
	numHeight := int64(0)
	for height := targetHeight; height >= oldestHeight; height-- {
		log.Info("Pruning Block at height", "height", height)
		// if block metadata is not found, stop purging
		if !bs.Has(calcBlockMetaKey(height)) {
			log.Info("block is missing at height", "height", height)
			if skipMissing {
				continue
			}
			break
		}

		meta := bs.LoadBlockMeta(height)
		batch.Delete(calcBlockMetaKey(height))
		for i := 0; i < meta.BlockID.PartsHeader.Total; i++ {
			batch.Delete(calcBlockPartKey(height, i))
		}
		batch.Delete(calcBlockCommitKey(height - 1))
		batch.Delete(calcSeenCommitKey(height))

		if progressInterval > 0 && numHeight%progressInterval == 0 {
			log.Info("blocks processed: current height", "height", height)
		}

		if numHeight%batchSize == 0 {
			batch.Write()
			batch = bs.blockStoreDB.NewBatch()
		}
		numHeight++
	}
	batch.WriteSync()

	if !skipCompaction {
		bs.blockStoreDB.Close()
		db, err := leveldb.OpenFile(path.Join(bs.chainDataDir, "data", "blockstore.db"), nil)
		if err != nil {
			return fmt.Errorf("cannot open blockstore.db for compaction, %s", err.Error())
		}
		defer db.Close()
		if err := db.CompactRange(util.Range{}); err != nil {
			return fmt.Errorf("failed to compact db, %s", err.Error())
		}
		log.Info("finished DB compaction")
	}

	return nil
}

//This Pruning Algorithm works by copying recent blocks in the block store i.e blocks which are above target height to the new blockstore database and original database is backed up
func (bs *PruningBlockStore) PruneviaCopying(chainDataDir string, numBlocksToRetain, pruneGraceFactor int64, skipMissing bool) error {
	latestHeight := bs.Height()
	var targetHeight int64
	targetHeight = latestHeight - numBlocksToRetain
	graceBlocks := (pruneGraceFactor / 100) * numBlocksToRetain
	oldestHeight := int64(-1)
	// find the oldest block
	it := bs.blockStoreDB.Iterator(calcBlockMetaPrefix, prefixRangeEnd(calcBlockMetaPrefix))
	for ; it.Valid(); it.Next() {
		oldestBlockMetaKey := it.Key()
		oldestHeight = getHeightFromKey(oldestBlockMetaKey)
		break
	}
	//Number of blocks to prune is less than grace blocks then skip pruning
	if ((targetHeight - oldestHeight) + 1) < graceBlocks {
		bs.blockStoreDB.Close()
		bs.db.Close()
		return nil
	}
	//Minimum Height is greater than Target Height so there are no blocks to prune, so there is no need to copy blocks to new blockstore database
	if oldestHeight >= targetHeight {
		bs.blockStoreDB.Close()
		bs.db.Close()
		return fmt.Errorf("no block below block %d", targetHeight)
	}
	for height := targetHeight + 1; height <= latestHeight; height++ {
		log.Info("Copying Block at height", "height", height)
		// If block metadata is not found, stop purging
		if !bs.Has(calcBlockMetaKey(height)) {
			log.Info("block is missing at height", "height", height)
			if skipMissing {
				continue
			}
			break
		}

		meta := bs.LoadBlockMeta(height)
		partset := types.NewPartSetFromHeader(meta.BlockID.PartsHeader)
		for i := 0; i < meta.BlockID.PartsHeader.Total; i++ {
			part := bs.LoadBlockPart(height, i)
			partset.AddPart(part)
		}
		seenCommit := bs.LoadSeenCommit(height)
		block := bs.LoadBlock(height)

		bs.SaveBlock(block, partset, seenCommit)

	}
	bs.blockStoreDB.Close()
	bs.db.Close()
	//After copying original blockstore database is copied to back up blockstore database
	srcblockstore := path.Join(chainDataDir, "data/blockstore.db")
	renamedblockstore := path.Join(chainDataDir, "data/blockstore.db.bak")
	err := os.Rename(srcblockstore, renamedblockstore)
	if err != nil {
		return err
	}
	//Copied blockstore database is renamed to blockstore.db
	srcblockstore = path.Join(chainDataDir, "data/blockstorev1.db")
	renamedblockstore = path.Join(chainDataDir, "data/blockstore.db")
	err = os.Rename(srcblockstore, renamedblockstore)
	if err != nil {
		return err
	}
	return nil
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

func (bs *PruningBlockStore) SaveBlock(block *types.Block, blockParts *types.PartSet,
	seenCommit *types.Commit) {

	if block == nil {
		cmn.PanicSanity("BlockStore can only save a non-nil block")
	}
	height := block.Height

	if !blockParts.IsComplete() {
		cmn.PanicSanity(fmt.Sprintf("BlockStore can only save complete block part sets"))
	}

	// Save block meta
	blockMeta := types.NewBlockMeta(block, blockParts)
	metaBytes := cdc.MustMarshalBinaryBare(blockMeta)
	bs.db.Set(calcBlockMetaKey(height), metaBytes)

	// Save block parts
	for i := 0; i < blockParts.Total(); i++ {
		part := blockParts.GetPart(i)
		bs.saveBlockPart(height, i, part)
	}

	// Save block commit (duplicate and separate from the Block)
	blockCommitBytes := cdc.MustMarshalBinaryBare(block.LastCommit)
	bs.db.Set(calcBlockCommitKey(height-1), blockCommitBytes)

	// Save seen commit (seen +2/3 precommits for block)
	// NOTE: we can delete this at a later height
	seenCommitBytes := cdc.MustMarshalBinaryBare(seenCommit)
	bs.db.Set(calcSeenCommitKey(height), seenCommitBytes)

	// Save new BlockStoreStateJSON descriptor
	BlockStoreStateJSON{Height: height}.Save(bs.db)

	// Done!
	bs.mtx.Lock()
	bs.height = height
	bs.mtx.Unlock()

	// Flush
	bs.db.SetSync(nil, nil)
}

func (bs *PruningBlockStore) saveBlockPart(height int64, index int, part *types.Part) {
	partBytes := cdc.MustMarshalBinaryBare(part)
	bs.db.Set(calcBlockPartKey(height, index), partBytes)
}

var blockStoreKey = []byte("blockStore")

type BlockStoreStateJSON struct {
	Height int64 `json:"height"`
}

// Save persists the blockStore state to the database as JSON.
func (bsj BlockStoreStateJSON) Save(db dbm.DB) {
	bytes, err := cdc.MarshalJSON(bsj)
	if err != nil {
		cmn.PanicSanity(fmt.Sprintf("Could not marshal state bytes: %v", err))
	}
	db.SetSync(blockStoreKey, bytes)
}

// LoadBlockStoreStateJSON returns the BlockStoreStateJSON as loaded from disk.
// If no BlockStoreStateJSON was previously persisted, it returns the zero value.
func LoadBlockStoreStateJSON(db dbm.DB) BlockStoreStateJSON {
	bytes := db.Get(blockStoreKey)
	if len(bytes) == 0 {
		return BlockStoreStateJSON{
			Height: 0,
		}
	}
	bsj := BlockStoreStateJSON{}
	err := cdc.UnmarshalJSON(bytes, &bsj)
	if err != nil {
		panic(fmt.Sprintf("Could not unmarshal bytes: %X", bytes))
	}
	return bsj
}

func Dir(src string, dst string) error {
	var err error
	var fds []os.FileInfo
	var srcinfo os.FileInfo

	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}

	if err = os.MkdirAll(dst, srcinfo.Mode()); err != nil {
		return err
	}

	if fds, err = ioutil.ReadDir(src); err != nil {
		return err
	}
	for _, fd := range fds {
		srcfp := path.Join(src, fd.Name())
		dstfp := path.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = Dir(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		} else {
			if err = File(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		}
	}
	return nil
}

// File copies a single file from src to dst
func File(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo

	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcinfo.Mode())
}
