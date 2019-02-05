package store

import (
	"strconv"
	"time"

	lru "github.com/hashicorp/golang-lru"
	abci "github.com/tendermint/tendermint/abci/types"
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

type MockBlockStore struct {
}

func NewMockBlockStore() BlockStore {
	return &MockBlockStore{}
}

func (s *MockBlockStore) GetBlockByHeight(height *int64) (*ctypes.ResultBlock, error) {
	h := int64(10)
	if height != nil {
		h = *height
	}

	lastCommit := &types.Commit{
		Precommits: []*types.Vote{{
			Height:    h - 1,
			Timestamp: time.Now(),
		}},
	}

	block := types.MakeBlock(h, nil, lastCommit, nil)
	blockMeta := types.NewBlockMeta(block, block.MakePartSet(2))

	return &ctypes.ResultBlock{
		BlockMeta: blockMeta,
		Block:     block,
	}, nil
}

func (s *MockBlockStore) GetBlockRangeByHeight(minHeight, maxHeight int64) (*ctypes.ResultBlockchainInfo, error) {
	result := &ctypes.ResultBlockchainInfo{
		LastHeight: maxHeight,
	}
	// emulate core.BlockchainInfo which only returns 20 blocks at a time
	if (maxHeight - minHeight) > 20 {
		maxHeight = minHeight + 20
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

func CreateBlockStoreInstance(cfg *BlockStoreConfig) (BlockStore, error) {
	var blockCacheStore BlockStore
	var cachedBlockStore BlockStore
	var err error
	if cfg.BlockStoretoCache == "Tendermint" {
		cachedBlockStore = NewTendermintBlockStore()
	}
	if cfg.BlockCacheAlgorithm == "None" {
		blockCacheStore = NewTendermintBlockStore()
	}
	if cfg.BlockCacheAlgorithm == "LRU" {
		blockCacheStore, err = NewLRUCacheBlockStore(cfg.BlockCacheSize, cachedBlockStore)
	}
	if cfg.BlockCacheAlgorithm == "2QCache" {
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
			blockMeta := cacheData.(types.BlockMeta)
			blockMetas = append(blockMetas, &blockMeta)
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
			s.cache.Add("Meta"+strconv.Itoa(int(i)), blockMeta)
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
			blockMeta := cacheData.(types.BlockMeta)
			blockMetas = append(blockMetas, &blockMeta)
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
			s.twoQueueCache.Add("Meta"+strconv.Itoa(int(i)), blockMeta)
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
