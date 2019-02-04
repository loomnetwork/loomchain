package store

import (
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/tendermint/tendermint/rpc/core"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
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
	return core.BlockchainInfo(minHeight, maxHeight)
}

func (s *TendermintBlockStore) GetBlockResults(height *int64) (*ctypes.ResultBlockResults, error) {
	return core.BlockResults(height)
}

type LRUCacheBlockStore struct {
	tendermintBlockStore BlockStore
	cache                *lru.Cache
}

type TwoQueueCacheBlockStore struct {
	tendermintBlockStore BlockStore
	twoQueueCache        *lru.TwoQueueCache
}

func NewLRUCacheBlockStore(size int64) BlockStore {
	lruCacheBlockStore := &LRUCacheBlockStore{}
	lruCacheBlockStore.tendermintBlockStore = NewTendermintBlockStore()
	lruCacheBlockStore.cache, _ = lru.New(int(size))
	return lruCacheBlockStore

}

func NewTwoQueueCacheBlockStore(size int64) BlockStore {
	twoQueueCacheBlockStore := &TwoQueueCacheBlockStore{}
	twoQueueCacheBlockStore.tendermintBlockStore = NewTendermintBlockStore()
	twoQueueCacheBlockStore.twoQueueCache, _ = lru.New2Q(int(size))
	return twoQueueCacheBlockStore
}

func (s *LRUCacheBlockStore) GetBlockByHeight(height *int64) (*ctypes.ResultBlock, error) {
	var blockinfo *ctypes.ResultBlock

	var err error

	h := int64(*height)
	cacheData, _ := s.cache.Get(h)

	if cacheData == nil {

		blockinfo, err = s.tendermintBlockStore.GetBlockByHeight(height)

		if err != nil {
			return nil, err
		}
		s.cache.Add(h, blockinfo)

	} else {

		blockinfo = cacheData.(*ctypes.ResultBlock)

	}

	return blockinfo, nil

}

func (s *LRUCacheBlockStore) GetBlockRangeByHeight(minHeight, maxHeight int64) (*ctypes.ResultBlockchainInfo, error) {
	return core.BlockchainInfo(minHeight, maxHeight)
}

func (s *LRUCacheBlockStore) GetBlockResults(height *int64) (*ctypes.ResultBlockResults, error) {
	return core.BlockResults(height)
}

func (s *TwoQueueCacheBlockStore) GetBlockByHeight(height *int64) (*ctypes.ResultBlock, error) {

	var blockinfo *ctypes.ResultBlock
	var err error
	h := int64(*height)

	cacheData, _ := s.twoQueueCache.Get(h)

	if cacheData == nil {

		blockinfo, err = s.tendermintBlockStore.GetBlockByHeight(height)

		if err != nil {
			return nil, err
		}
		s.twoQueueCache.Add(h, blockinfo)

	} else {

		blockinfo = cacheData.(*ctypes.ResultBlock)

	}

	return blockinfo, nil
}

func (s *TwoQueueCacheBlockStore) GetBlockRangeByHeight(minHeight, maxHeight int64) (*ctypes.ResultBlockchainInfo, error) {
	return core.BlockchainInfo(minHeight, maxHeight)
}

func (s *TwoQueueCacheBlockStore) GetBlockResults(height *int64) (*ctypes.ResultBlockResults, error) {
	return core.BlockResults(height)
}
