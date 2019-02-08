package store

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru"
	"github.com/tendermint/go-amino"
	abci "github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/rpc/core"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/state"
	"github.com/tendermint/tendermint/types"
)

// BlockStore provides access to block info.
//
// TODO: This is a quick, dirty, and very leaky abstraction of the underlying TM block store
//       primarily so integration tests can use MockBlockStore, ideally this shouldn't be leaking
//       TM types.
// TODO: Since the block store is only used by the QueryServer the amount of data returned by each
//       function should be minimized, and probably aggressively cached.
type BlockStore interface {
	// GetBlockByHeight retrieves block info at the specified height,
	// specify nil to retrieve the latest block info.
	GetBlockByHeight(height *int64) (*ctypes.ResultBlock, error)
	// GetBlockRangeByHeight retrieves block info at the specified height range,
	// specify nil to retrieve the latest block info.
	GetBlockRangeByHeight(minHeight, maxHeight int64) (*ctypes.ResultBlockchainInfo, error)
	// GetBlockResults retrieves the results of the txs committed to the block at the specified height,
	// specify nil to retrieve results from the latest block.
	GetBlockResults(height *int64) (*ctypes.ResultBlockResults, error)
}

var cdc = amino.NewCodec()

var maxHeight = int64(20)

type MockBlockStore struct {
	Db dbm.DB

	mtx    sync.RWMutex
	height int64
}

func NewMockBlockStore() *MockBlockStore {
	return &MockBlockStore{}
}

func GetMockBlockStore(Db dbm.DB) *MockBlockStore {
	bsjson := LoadBlockStoreStateJSON(Db)
	return &MockBlockStore{
		height: bsjson.Height,
		Db:     Db,
	}
}

func (s *MockBlockStore) Height() int64 {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.height
}

func filterMinMax(height, min, max, limit int64) (int64, int64, error) {
	// filter negatives
	if min < 0 || max < 0 {
		return min, max, fmt.Errorf("heights must be non-negative")
	}
	// adjust for default values
	if min == 0 {
		min = 1
	}
	if max == 0 {
		max = height
	}
	// limit max to the height
	max = cmn.MinInt64(height, max)
	// limit min to within `limit` of max
	// so the total number of blocks returned will be `limit`
	min = cmn.MaxInt64(min, max-limit+1)

	if min > max {
		return min, max, fmt.Errorf("min height %d can't be greater than max height %d", min, max)
	}
	return min, max, nil
}

//It is used to save Blocks created for MockBlockstore

func (s *MockBlockStore) SaveBlock(block *types.Block, blockParts *types.PartSet, seenCommit *types.Commit) {
	if block == nil {
		cmn.PanicSanity("BlockStore can only save a non-nil block")
	}
	height := block.Height
	if g, w := height, s.Height()+1; g != w {
		cmn.PanicSanity(fmt.Sprintf("BlockStore can only save contiguous blocks. Wanted %v, got %v", w, g))
	}
	if !blockParts.IsComplete() {
		cmn.PanicSanity(fmt.Sprintf("BlockStore can only save complete block part sets"))
	}
	// Save block meta
	blockMeta := types.NewBlockMeta(block, blockParts)
	metaBytes := cdc.MustMarshalBinaryBare(blockMeta)
	s.Db.Set(calcBlockMetaKey(height), metaBytes)
	// Save block parts
	for i := 0; i < blockParts.Total(); i++ {
		part := blockParts.GetPart(i)
		s.saveBlockPart(height, i, part)
	}
	// Save block commit (duplicate and separate from the Block)
	blockCommitBytes := cdc.MustMarshalBinaryBare(block.LastCommit)
	s.Db.Set(calcBlockCommitKey(height-1), blockCommitBytes)
	// Save seen commit (seen +2/3 precommits for block)
	// NOTE: we can delete this at a later height
	seenCommitBytes := cdc.MustMarshalBinaryBare(seenCommit)
	s.Db.Set(calcSeenCommitKey(height), seenCommitBytes)
	// Save new BlockStoreStateJSON descriptor
	BlockStoreStateJSON{Height: height}.Save(s.Db)
	// Done!
	s.mtx.Lock()
	s.height = height
	s.mtx.Unlock()
	// Flush
	s.Db.SetSync(nil, nil)
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

//-----------------------------------------------------------------------------

var blockStoreKey = []byte("blockStore")

type BlockStoreStateJSON struct {
	Height int64 `json:"height"`
}

// Save persists the blockStore state to the database as JSON.
func (bsj BlockStoreStateJSON) Save(Db dbm.DB) {
	bytes, err := cdc.MarshalJSON(bsj)
	if err != nil {
		cmn.PanicSanity(fmt.Sprintf("Could not marshal state bytes: %v", err))
	}
	Db.SetSync(blockStoreKey, bytes)
}

// LoadBlockStoreStateJSON returns the BlockStoreStateJSON as loaded from disk.
// If no BlockStoreStateJSON was previously persisted, it returns the zero value.
func LoadBlockStoreStateJSON(Db dbm.DB) BlockStoreStateJSON {
	bytes := Db.Get(blockStoreKey)
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

func (s *MockBlockStore) saveBlockPart(height int64, index int, part *types.Part) {
	if height != s.Height()+1 {
		cmn.PanicSanity(fmt.Sprintf("BlockStore can only save contiguous blocks. Wanted %v, got %v", s.Height()+1, height))
	}
	partBytes := cdc.MustMarshalBinaryBare(part)
	s.Db.Set(calcBlockPartKey(height, index), partBytes)
}

// LoadBlock returns the block with the given height.
// If no block is found for that height, it returns nil.
func (s *MockBlockStore) LoadBlock(height int64) *types.Block {
	var blockMeta = s.LoadBlockMeta(height)
	if blockMeta == nil {
		return nil
	}

	var block = new(types.Block)
	buf := []byte{}
	for i := 0; i < blockMeta.BlockID.PartsHeader.Total; i++ {
		part := s.LoadBlockPart(height, i)
		buf = append(buf, part.Bytes...)
	}
	err := cdc.UnmarshalBinaryLengthPrefixed(buf, block)
	if err != nil {
		// NOTE: The existence of meta should imply the existence of the
		// block. So, make sure meta is only saved after blocks are saved.
		panic(cmn.ErrorWrap(err, "Error reading block"))
	}
	return block
}

// LoadBlockPart returns the Part at the given index
// from the block at the given height.
// If no part is found for the given height and index, it returns nil.
func (s *MockBlockStore) LoadBlockPart(height int64, index int) *types.Part {
	var part = new(types.Part)
	bz := s.Db.Get(calcBlockPartKey(height, index))
	if len(bz) == 0 {
		return nil
	}
	err := cdc.UnmarshalBinaryBare(bz, part)
	if err != nil {
		panic(cmn.ErrorWrap(err, "Error reading block part"))
	}
	return part
}

// LoadBlockMeta returns the BlockMeta for the given height.
// If no block is found for the given height, it returns nil.
func (s *MockBlockStore) LoadBlockMeta(height int64) *types.BlockMeta {
	var blockMeta = new(types.BlockMeta)
	bz := s.Db.Get(calcBlockMetaKey(height))
	if len(bz) == 0 {
		return nil
	}
	err := cdc.UnmarshalBinaryBare(bz, blockMeta)
	if err != nil {
		panic(cmn.ErrorWrap(err, "Error reading block meta"))
	}
	return blockMeta
}

// LoadBlockCommit returns the Commit for the given height.
// This commit consists of the +2/3 and other Precommit-votes for block at `height`,
// and it comes from the block.LastCommit for `height+1`.
// If no commit is found for the given height, it returns nil.
func (s *MockBlockStore) LoadBlockCommit(height int64) *types.Commit {
	var commit = new(types.Commit)
	bz := s.Db.Get(calcBlockCommitKey(height))
	if len(bz) == 0 {
		return nil
	}
	err := cdc.UnmarshalBinaryBare(bz, commit)
	if err != nil {
		panic(cmn.ErrorWrap(err, "Error reading block commit"))
	}
	return commit
}

// LoadSeenCommit returns the locally seen Commit for the given height.
// This is useful when we've seen a commit, but there has not yet been
// a new block at `height + 1` that includes this commit in its block.LastCommit.
func (s *MockBlockStore) LoadSeenCommit(height int64) *types.Commit {
	var commit = new(types.Commit)
	bz := s.Db.Get(calcSeenCommitKey(height))
	if len(bz) == 0 {
		return nil
	}
	err := cdc.UnmarshalBinaryBare(bz, commit)
	if err != nil {
		panic(cmn.ErrorWrap(err, "Error reading block seen commit"))
	}
	return commit
}

func getHeight(currentHeight int64, heightPtr *int64) (int64, error) {
	if heightPtr != nil {
		height := *heightPtr
		if height <= 0 {
			return 0, fmt.Errorf("Height must be greater than 0")
		}
		if height > currentHeight {
			return 0, fmt.Errorf("Height must be less than or equal to the current blockchain height")
		}
		return height, nil
	}
	return currentHeight, nil
}

func (s *MockBlockStore) GetBlockByHeight(heightptr *int64) (*ctypes.ResultBlock, error) {

	storeHeight := s.Height()
	height, err := getHeight(storeHeight, heightptr)
	if err != nil {
		return nil, err
	}

	blockMeta := s.LoadBlockMeta(height)
	block := s.LoadBlock(height)
	return &ctypes.ResultBlock{blockMeta, block}, nil
}

func (s *MockBlockStore) GetBlockRangeByHeight(minHeight, maxHeight int64) (*ctypes.ResultBlockchainInfo, error) {
	result := &ctypes.ResultBlockchainInfo{
		LastHeight: maxHeight,
	}
	// emulate core.BlockchainInfo which only returns 20 blocks at a time
	if (maxHeight - minHeight) > 20 {
		maxHeight = minHeight + 20
	}

	const limit int64 = 20
	var err error
	minHeight, maxHeight, err = filterMinMax(s.Height(), minHeight, maxHeight, limit)
	if err != nil {
		return nil, err
	}

	for i := minHeight; i <= maxHeight; i++ {
		block, err := s.GetBlockByHeight(&i)
		if err != nil {
			return nil, err
		}
		result.BlockMetas = append(result.BlockMetas, block.BlockMeta)
	}
	return result, nil
}

func (s *MockBlockStore) GetBlockResults(height *int64) (*ctypes.ResultBlockResults, error) {
	h := int64(10)
	if height != nil {
		h = *height
	}
	return &ctypes.ResultBlockResults{
		Height:  h,
		Results: nil,
	}, nil
}

var _ BlockStore = &MockBlockStore{}

type TendermintBlockStore struct {
}

var _ BlockStore = &TendermintBlockStore{}

func NewTendermintBlockStore() BlockStore {
	return &TendermintBlockStore{}
}

//Structure for cached fields representation

type CachedBlockData struct {
	BlockID           types.BlockID
	HeaderLastBlockID types.BlockID
	HeaderHeight      int64
	Timestmap         time.Time
	DeliverTx         []*abci.ResponseDeliverTx
}

type BlockStoreConfig struct {
	BlockStoretoCache   string
	BlockCacheAlgorithm string
	BlockCacheSize      int64
}

func DefaultBlockCacheConfig() *BlockStoreConfig {
	return &BlockStoreConfig{
		BlockStoretoCache:   "Tendermint",
		BlockCacheAlgorithm: "LRU",
		BlockCacheSize:      10000, //Size should be more because of blockrangebyheight API
	}
}

func NewBlockStore(cfg *BlockStoreConfig) (BlockStore, error) {
	var blockCacheStore BlockStore
	var cachedBlockStore BlockStore
	var err error
	if strings.EqualFold(cfg.BlockStoretoCache, "Tendermint") {
		cachedBlockStore = NewTendermintBlockStore()
	}
	if strings.EqualFold(cfg.BlockCacheAlgorithm, "None") {
		blockCacheStore = NewTendermintBlockStore()
	}
	if strings.EqualFold(cfg.BlockCacheAlgorithm, "LRU") {
		blockCacheStore, err = NewLRUCacheBlockStore(cfg.BlockCacheSize, cachedBlockStore)
	}
	if strings.EqualFold(cfg.BlockCacheAlgorithm, "2QCache") {
		blockCacheStore, err = NewTwoQueueCacheBlockStore(cfg.BlockCacheSize, cachedBlockStore)
	}
	if err != nil {
		return nil, err
	}
	return blockCacheStore, nil

}

func (s *TendermintBlockStore) GetBlockByHeight(height *int64) (*ctypes.ResultBlock, error) {
	blockResult, err := core.Block(height)
	if err != nil {
		return nil, err
	}
	blockMeta := types.BlockMeta{
		BlockID: blockResult.BlockMeta.BlockID,
	}
	header := types.Header{
		LastBlockID: blockResult.Block.Header.LastBlockID,
		Time:        blockResult.Block.Header.Time,
	}
	block := types.Block{
		Header: header,
	}
	resultBlock := ctypes.ResultBlock{
		BlockMeta: &blockMeta,
		Block:     &block,
	}
	return &resultBlock, nil
}

func (s *TendermintBlockStore) GetBlockRangeByHeight(minHeight, maxHeight int64) (*ctypes.ResultBlockchainInfo, error) {
	blockResult, err := core.BlockchainInfo(minHeight, maxHeight)
	if err != nil {
		return nil, err
	}
	blockchaininfo := ctypes.ResultBlockchainInfo{
		BlockMetas: blockResult.BlockMetas,
	}
	return &blockchaininfo, nil

}

func (s *TendermintBlockStore) GetBlockResults(height *int64) (*ctypes.ResultBlockResults, error) {
	blockResult, err := core.BlockResults(height)
	if err != nil {
		return nil, err
	}
	ABCIResponses := state.ABCIResponses{
		DeliverTx: blockResult.Results.DeliverTx,
	}
	blockchaininfo := ctypes.ResultBlockResults{
		Results: &ABCIResponses,
	}
	return &blockchaininfo, nil
}

type LRUCacheBlockStore struct {
	cachedBlockStore BlockStore
	cache            *lru.Cache
}

type TwoQueueCacheBlockStore struct {
	cachedBlockStore BlockStore
	twoQueueCache    *lru.TwoQueueCache
}

func NewLRUCacheBlockStore(size int64, blockstore BlockStore) (BlockStore, error) {
	var err error
	lruCacheBlockStore := &LRUCacheBlockStore{}
	lruCacheBlockStore.cachedBlockStore = blockstore
	lruCacheBlockStore.cache, err = lru.New(int(size))
	if err != nil {
		return nil, err
	}

	return lruCacheBlockStore, nil

}

func NewTwoQueueCacheBlockStore(size int64, blockstore BlockStore) (BlockStore, error) {
	var err error
	twoQueueCacheBlockStore := &TwoQueueCacheBlockStore{}
	twoQueueCacheBlockStore.cachedBlockStore = blockstore
	twoQueueCacheBlockStore.twoQueueCache, err = lru.New2Q(int(size))
	if err != nil {
		return nil, err
	}
	return twoQueueCacheBlockStore, nil
}

func (s *LRUCacheBlockStore) GetBlockByHeight(height *int64) (*ctypes.ResultBlock, error) {
	var blockinfo *ctypes.ResultBlock
	var err error
	h := int64(*height)
	cacheData, ok := s.cache.Get(h)
	if ok {
		blockinfo = cacheData.(*ctypes.ResultBlock)
	} else {
		blockinfo, err = s.cachedBlockStore.GetBlockByHeight(height)
		if err != nil {
			return nil, err
		}
		s.cache.Add(h, blockinfo)
	}
	return blockinfo, nil

}

func (s *LRUCacheBlockStore) GetBlockRangeByHeight(minHeight, maxHeight int64) (*ctypes.ResultBlockchainInfo, error) {
	blockMetas := []*types.BlockMeta{}
	for i := minHeight; i <= maxHeight; i++ {
		cacheData, ok := s.cache.Get("Meta" + strconv.Itoa(int(i)))
		if ok {
			blockMeta := cacheData.(*types.BlockMeta)
			blockMetas = append(blockMetas, blockMeta)
		} else {
			block, err := s.cachedBlockStore.GetBlockRangeByHeight(i, i)
			if err != nil {
				return nil, err
			}
			header := types.Header{
				Height: block.BlockMetas[0].Header.Height,
			}
			blockMeta := types.BlockMeta{
				BlockID: block.BlockMetas[0].BlockID,
				Header:  header,
			}
			blockMetas = append(blockMetas, &blockMeta)
			s.cache.Add("Meta"+strconv.Itoa(int(i)), &blockMeta)
		}
	}
	blockchaininfo := ctypes.ResultBlockchainInfo{
		BlockMetas: blockMetas,
	}
	return &blockchaininfo, nil

}

func (s *LRUCacheBlockStore) GetBlockResults(height *int64) (*ctypes.ResultBlockResults, error) {
	var blockinfo *ctypes.ResultBlockResults
	var err error
	h := int64(*height)
	cacheData, ok := s.cache.Get("BR:" + string(h))
	if ok {
		blockinfo = cacheData.(*ctypes.ResultBlockResults)
	} else {
		blockinfo, err = s.cachedBlockStore.GetBlockResults(height)
		if err != nil {
			return nil, err
		}
		s.cache.Add("BR:"+string(h), blockinfo)
	}
	return blockinfo, nil
}

func (s *TwoQueueCacheBlockStore) GetBlockByHeight(height *int64) (*ctypes.ResultBlock, error) {
	var blockinfo *ctypes.ResultBlock
	var err error
	h := int64(*height)
	cacheData, ok := s.twoQueueCache.Get(h)
	if ok {
		blockinfo = cacheData.(*ctypes.ResultBlock)
	} else {
		blockinfo, err = s.cachedBlockStore.GetBlockByHeight(height)

		if err != nil {
			return nil, err
		}
		s.twoQueueCache.Add(h, blockinfo)

	}
	return blockinfo, nil
}

func (s *TwoQueueCacheBlockStore) GetBlockRangeByHeight(minHeight, maxHeight int64) (*ctypes.ResultBlockchainInfo, error) {
	blockMetas := []*types.BlockMeta{}
	for i := minHeight; i <= maxHeight; i++ {
		cacheData, ok := s.twoQueueCache.Get("Meta" + strconv.Itoa(int(i)))
		if ok {
			blockMeta := cacheData.(*types.BlockMeta)
			blockMetas = append(blockMetas, blockMeta)
		} else {
			block, err := s.cachedBlockStore.GetBlockRangeByHeight(i, i)
			if err != nil {
				return nil, err
			}
			header := types.Header{
				Height: block.BlockMetas[0].Header.Height,
			}
			blockMeta := types.BlockMeta{
				BlockID: block.BlockMetas[0].BlockID,
				Header:  header,
			}
			blockMetas = append(blockMetas, &blockMeta)
			s.twoQueueCache.Add("Meta"+strconv.Itoa(int(i)), &blockMeta)
		}
	}
	blockchaininfo := ctypes.ResultBlockchainInfo{
		BlockMetas: blockMetas,
	}
	return &blockchaininfo, nil
}

func (s *TwoQueueCacheBlockStore) GetBlockResults(height *int64) (*ctypes.ResultBlockResults, error) {
	var blockinfo *ctypes.ResultBlockResults
	var err error
	h := int64(*height)
	cacheData, ok := s.twoQueueCache.Get("BR:" + string(h))
	if ok {
		blockinfo = cacheData.(*ctypes.ResultBlockResults)
	} else {
		blockinfo, err = s.cachedBlockStore.GetBlockResults(height)
		if err != nil {
			return nil, err
		}
		s.twoQueueCache.Add("BR:"+string(h), blockinfo)
	}
	return blockinfo, nil
}
