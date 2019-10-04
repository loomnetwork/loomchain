package store

import (
	lru "github.com/hashicorp/golang-lru"
	abci "github.com/tendermint/tendermint/abci/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
)

type LRUBlockStoreCache struct {
	CachedBlockStore BlockStore
	Cache            *lru.Cache
}

func NewLRUBlockStoreCache(size int64, blockstore BlockStore) (*LRUBlockStoreCache, error) {
	var err error
	lruCacheBlockStore := &LRUBlockStoreCache{}
	lruCacheBlockStore.CachedBlockStore = blockstore
	lruCacheBlockStore.Cache, err = lru.New(int(size))
	if err != nil {
		return nil, err
	}

	return lruCacheBlockStore, nil
}

func (s *LRUBlockStoreCache) GetBlockByHeight(height *int64) (*ctypes.ResultBlock, error) {
	var blockinfo *ctypes.ResultBlock
	var err error
	var h int64
	if height != nil {
		h = int64(*height)
	}

	cacheData, ok := s.Cache.Get(h)
	if ok {
		blockinfo = cacheData.(*ctypes.ResultBlock)
	} else {
		blockinfo, err = s.CachedBlockStore.GetBlockByHeight(height)
		if err != nil {
			return nil, err
		}
		// Takes care of special case when height is nil and default maximum height block is returned by BlockStore API
		s.Cache.Add(blockinfo.Block.Height, blockinfo)
	}
	return blockinfo, nil

}

func (s *LRUBlockStoreCache) GetBlockRangeByHeight(minHeight, maxHeight int64) (*ctypes.ResultBlockchainInfo, error) {
	const limit int64 = 20
	var err error
	// Get filterMinMax added to emulate error handling covered in tendermint blockstore
	minHeight, maxHeight, err = filterMinMaxforCache(minHeight, maxHeight)
	if err != nil {
		return nil, err
	}
	// Caches maximum 20 blocks per API call
	if (maxHeight - minHeight) > limit {
		minHeight = maxHeight - limit + 1
	}

	blockMetas := []*types.BlockMeta{}
	for i := minHeight; i <= maxHeight; i++ {
		cacheData, ok := s.Cache.Get(blockMetaKey(i))
		if ok {
			blockMeta := cacheData.(*types.BlockMeta)
			blockMetas = append(blockMetas, blockMeta)
		} else {
			// Called to fetch limited BlockInformation - BlockMetasOnly
			blockRange, err := s.CachedBlockStore.GetBlockRangeByHeight(i, i)
			if err != nil {
				break
				// This error can be ignored as it arise when i is greater than blockstore height,
				// for which nothing is to be done.
				// Blocks till maximum blockchain height will already be cached till this point.
				// Core tendermint API does not throw error in this case (maxheight > blockchain height in height range)
				// so cache wrapper is also not throwing error.
			} else if (len(blockRange.BlockMetas) > 0) && (blockRange.BlockMetas[0] != nil) {
				header := types.Header{
					Height: blockRange.BlockMetas[0].Header.Height,
				}
				blockMeta := types.BlockMeta{
					BlockID: blockRange.BlockMetas[0].BlockID,
					Header:  header,
				}
				blockMetas = append(blockMetas, &blockMeta)
				s.Cache.Add(blockMetaKey(blockRange.BlockMetas[0].Header.Height), &blockMeta)
			}
		}
	}
	blockchaininfo := ctypes.ResultBlockchainInfo{
		BlockMetas: blockMetas,
	}
	return &blockchaininfo, nil

}

func (s *LRUBlockStoreCache) GetBlockResults(height *int64) (*ctypes.ResultBlockResults, error) {
	var blockinfo *ctypes.ResultBlockResults
	var err error
	var h int64
	if height != nil {
		h = int64(*height)
	}

	cacheData, ok := s.Cache.Get(blockResultKey(h))
	if ok {
		blockinfo = cacheData.(*ctypes.ResultBlockResults)
	} else {
		blockinfo, err = s.CachedBlockStore.GetBlockResults(height)
		if err != nil {
			return nil, err
		}
		s.Cache.Add(blockResultKey(blockinfo.Height), blockinfo)
	}
	return blockinfo, nil
}

func (s *LRUBlockStoreCache) GetTxResult(txHash []byte) (*ctypes.ResultTx, error) {
	var txResult *ctypes.ResultTx
	cacheData, ok := s.Cache.Get(txHashKey(txHash))
	if ok {
		txResult = cacheData.(*ctypes.ResultTx)
	} else {
		var err error
		txResult, err = s.CachedBlockStore.GetTxResult(txHash)
		if err != nil {
			return nil, err
		}
		s.Cache.Add(txHashKey(txResult.Hash), txResult)
	}
	return txResult, nil
}

func (s *LRUBlockStoreCache) GetTxResultByHeightAndIndex(height *int64, index int) (*ctypes.ResultTx, error) {
	blockResult, err := s.GetBlockResults(height)
	if err != nil {
		return nil, err
	}

	if len(blockResult.Results.DeliverTx) <= index {
		return nil, ErrIndexOutOfRange
	}

	return &ctypes.ResultTx{
		Index:  uint32(index),
		Height: *height,
		TxResult: abci.ResponseDeliverTx{
			Code: blockResult.Results.DeliverTx[index].Code,
			Data: blockResult.Results.DeliverTx[index].Data,
			Info: blockResult.Results.DeliverTx[index].Info,
		},
	}, nil
}
