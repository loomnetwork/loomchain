package store

import (
	"fmt"
	"strconv"

	lru "github.com/hashicorp/golang-lru"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
)

type LRUCacheBlockStore struct {
	CachedBlockStore BlockStore
	Cache            *lru.Cache
}

func NewLRUCacheBlockStore(size int64, blockstore BlockStore) (*LRUCacheBlockStore, error) {
	var err error
	lruCacheBlockStore := &LRUCacheBlockStore{}
	lruCacheBlockStore.CachedBlockStore = blockstore
	lruCacheBlockStore.Cache, err = lru.New(int(size))
	if err != nil {
		return nil, err
	}

	return lruCacheBlockStore, nil

}

func (s *LRUCacheBlockStore) GetBlockByHeight(height *int64) (*ctypes.ResultBlock, error) {
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
		//Takes care of special case when height is nil and default maximum height block is returned by BlockStore API
		s.Cache.Add(blockinfo.Block.Height, blockinfo)
	}
	return blockinfo, nil

}

func (s *LRUCacheBlockStore) GetBlockRangeByHeight(minHeight, maxHeight int64) (*ctypes.ResultBlockchainInfo, error) {
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
		cacheData, ok := s.Cache.Get("Meta" + strconv.FormatInt(i, 10))
		if ok {
			blockMeta := cacheData.(*types.BlockMeta)
			blockMetas = append(blockMetas, blockMeta)
		} else {
			//Called to fetch limited BlockInformation - BlockMetasOnly
			block, err := s.CachedBlockStore.GetBlockRangeByHeight(i, i)
			if err != nil {
				fmt.Println(err)
				//This error can be ignored as it arise when i is greater than blockstore height, for which nothing is to be done
				//Blocks till maximum blockchain height will already be cached till this point. Core tendermint API does not throw error in this case (maxheight > blockchain height in height range)so cache wrapper is also not throwing error
			} else {
				header := types.Header{
					Height: block.BlockMetas[0].Header.Height,
				}
				blockMeta := types.BlockMeta{
					BlockID: block.BlockMetas[0].BlockID,
					Header:  header,
				}
				blockMetas = append(blockMetas, &blockMeta)
				s.Cache.Add("Meta"+strconv.FormatInt(block.BlockMetas[0].Header.Height, 10), &blockMeta)
			}
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
	var h int64
	if height != nil {
		h = int64(*height)
	}

	cacheData, ok := s.Cache.Get("BR:" + strconv.FormatInt(h, 10))
	if ok {
		blockinfo = cacheData.(*ctypes.ResultBlockResults)
	} else {
		blockinfo, err = s.CachedBlockStore.GetBlockResults(height)
		if err != nil {
			return nil, err
		}
		s.Cache.Add("BR:"+strconv.FormatInt(blockinfo.Height, 10), blockinfo)
	}
	return blockinfo, nil
}
