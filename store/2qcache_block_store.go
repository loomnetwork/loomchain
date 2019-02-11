package store

import (
	lru "github.com/hashicorp/golang-lru"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
)

type TwoQueueCacheBlockStore struct {
	CachedBlockStore BlockStore
	TwoQueueCache    *lru.TwoQueueCache
}

func NewTwoQueueCacheBlockStore(size int64, blockstore BlockStore) (*TwoQueueCacheBlockStore, error) {
	var err error
	twoQueueCacheBlockStore := &TwoQueueCacheBlockStore{}
	twoQueueCacheBlockStore.CachedBlockStore = blockstore
	twoQueueCacheBlockStore.TwoQueueCache, err = lru.New2Q(int(size))
	if err != nil {
		return nil, err
	}
	return twoQueueCacheBlockStore, nil
}

func (s *TwoQueueCacheBlockStore) GetBlockByHeight(height *int64) (*ctypes.ResultBlock, error) {
	var blockinfo *ctypes.ResultBlock
	var err error
	var h int64
	if height != nil {
		h = int64(*height)
	}

	cacheData, ok := s.TwoQueueCache.Get(h)
	if ok {
		blockinfo = cacheData.(*ctypes.ResultBlock)
	} else {
		blockinfo, err = s.CachedBlockStore.GetBlockByHeight(height)
		if err != nil {
			return nil, err
		}
		//Takes care of special case when height is nil and default maximum height block is returned by BlockStore API
		s.TwoQueueCache.Add(blockinfo.Block.Height, blockinfo)
	}
	return blockinfo, nil

}

func (s *TwoQueueCacheBlockStore) GetBlockRangeByHeight(minHeight, maxHeight int64) (*ctypes.ResultBlockchainInfo, error) {
	const limit int64 = 20
	var err error
	//Get filterMinMax added to emulate error handling covered in tendermint blockstore
	minHeight, maxHeight, err = filterMinMaxforCache(minHeight, maxHeight, limit)
	if err != nil {
		return nil, err
	}
	//Caches maximum 20 blocks per API call
	if (maxHeight - minHeight) > limit {
		minHeight = maxHeight - limit + 1
	}

	blockMetas := []*types.BlockMeta{}
	for i := minHeight; i <= maxHeight; i++ {
		cacheData, ok := s.TwoQueueCache.Get(blockMetaKey(i))
		if ok {
			blockMeta := cacheData.(*types.BlockMeta)
			blockMetas = append(blockMetas, blockMeta)
		} else {
			//Called to fetch limited BlockInformation - BlockMetasOnly
			blockRange, err := s.CachedBlockStore.GetBlockRangeByHeight(i, i)
			if err != nil {
				break
				//This error can be ignored as it arise when i is greater than blockstore height, for which nothing is to be done
				//Blocks till maximum blockchain height will already be cached till this point. Core tendermint API does not throw error in this case (maxheight > blockchain height in height range)so cache wrapper is also not throwing error
			} else {
				header := types.Header{
					Height: blockRange.BlockMetas[0].Header.Height,
				}
				blockMeta := types.BlockMeta{
					BlockID: blockRange.BlockMetas[0].BlockID,
					Header:  header,
				}
				blockMetas = append(blockMetas, &blockMeta)
				s.TwoQueueCache.Add(blockMetaKey(blockRange.BlockMetas[0].Header.Height), &blockMeta)
			}
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
	var h int64
	if height != nil {
		h = int64(*height)
	}

	cacheData, ok := s.TwoQueueCache.Get(blockResultKey(h))
	if ok {
		blockinfo = cacheData.(*ctypes.ResultBlockResults)
	} else {
		blockinfo, err = s.CachedBlockStore.GetBlockResults(height)
		if err != nil {
			return nil, err
		}
		s.TwoQueueCache.Add(blockResultKey(blockinfo.Height), blockinfo)
	}
	return blockinfo, nil
}
