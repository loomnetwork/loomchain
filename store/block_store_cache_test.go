package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	maxHeight = 2000
)

func TestBlockFetchAtHeightLRU(t *testing.T) {
	b := NewMockBlockStore()
	cachedblockStore, err := NewLRUBlockStoreCache(200, b)
	require.NoError(t, err)
	height := int64(19)

	//Cache Empty at present resulting in Cache miss
	_, ok := cachedblockStore.Cache.Get(height)
	require.Equal(t, ok, false, "Cache miss")

	//request for a block at a given height. Data is returned from Mockstore API and is cached in LRU Cache
	blockstoreData, err := cachedblockStore.GetBlockByHeight(&height)
	require.NoError(t, err)
	require.Equal(t, height, blockstoreData.Block.Height, "Block height matches requested height")

	//Block info in above provided height gets cached resulting in cache hit
	_, ok = cachedblockStore.Cache.Get(height)
	require.Equal(t, ok, true, "Cache hit")

	//request for a block at a given height, requested earlier as well, Data is returned from  Cache
	blockstoreData, err = cachedblockStore.GetBlockByHeight(&height)
	require.NoError(t, err)
	require.Equal(t, height, blockstoreData.Block.Height, "Block height matches requested height")

	//request for a block for more than maximum height, error is returned by Cache and no caching occurs
	height = int64(2100) // Maximum block height is now 2000
	blockstoreData, err = cachedblockStore.GetBlockByHeight(&height)
	require.Error(t, err, "Cache Gives Error as block fetched is greater than maximum height")

	//request for a block for negative height, error is returned by Cache and no caching occurs
	height = int64(-1)
	blockstoreData, err = cachedblockStore.GetBlockByHeight(&height)
	require.Error(t, err, "Cache Gives Error as block fetched is for height <= 0")

	//block at maximum height not present in cache
	height = int64(maxHeight)
	_, ok = cachedblockStore.Cache.Get(height)
	require.Equal(t, ok, false, "Cache miss")

	//request for a block for nil height, maximum height data is returned and cached
	blockstoreData, err = cachedblockStore.GetBlockByHeight(nil)
	require.NoError(t, err, "Gives maximum height block")
	require.Equal(t, int64(maxHeight), blockstoreData.Block.Height, "maximum height block was fetched")

	//block at maximum height present in cache
	_, ok = cachedblockStore.Cache.Get(height)
	require.Equal(t, ok, true, "Cache hit")

}

func TestGetBlockRangeByHeightLRU(t *testing.T) {
	b := NewMockBlockStore()
	cachedblockStore, err := NewLRUBlockStoreCache(200, b)
	require.NoError(t, err)

	minheight := int64(1)
	maxheight := int64(15)

	//Cache Empty at present resulting in Cache miss
	for i := minheight; i <= maxheight; i++ {
		height := int64(i)
		_, ok := cachedblockStore.Cache.Get(blockMetaKey(height))
		require.Equal(t, ok, false, "Cache miss")
	}

	//request for a blocks in a given height range. Data is returned from Mockstore API and is cached in LRU Cache
	blockrangeData, err := cachedblockStore.GetBlockRangeByHeight(minheight, maxheight)
	require.NoError(t, err)

	//Blocks returned by MockBlockStore are verified for data accuracy
	for i := minheight; i <= maxheight; i++ {
		require.Equal(
			t, int64(i), int64(blockrangeData.BlockMetas[i-1].Header.Height), "Block height matches request height",
		)
	}

	//Block infos in above provided height range gets cached resulting in cache hit for each height in height range
	for i := minheight; i <= maxheight; i++ {
		height := int64(i)
		_, ok := cachedblockStore.Cache.Get(blockMetaKey(height))
		require.Equal(t, ok, true, "Cache hit")
	}

	blockrangeData, err = cachedblockStore.GetBlockRangeByHeight(minheight, maxheight)
	require.NoError(t, err)

	//Blocks returned by Cached are verified for data accuracy
	for i := minheight; i <= maxheight; i++ {
		require.Equal(
			t, int64(i), int64(blockrangeData.BlockMetas[i-1].Header.Height), "Block height matches expected height",
		)
	}

	//request for blockinfo for height range where max height is greater than maximum height of blockchain, error is returned by Cache, caching occurs till max height of blockchain
	minheight = int64(45)
	maxheight = int64(51)

	//Initially there is a cache miss
	for i := minheight; i <= maxheight; i++ {
		height := int64(i)
		_, ok := cachedblockStore.Cache.Get(blockMetaKey(height))
		require.Equal(t, ok, false, "Cache miss")
	}

	blockrangeData, err = cachedblockStore.GetBlockRangeByHeight(minheight, maxheight)
	require.NoError(t, err, "MockstoreBlock APIs are called,caching of BlockMetas occurs till maximum height of blockchain")

	//Cache hit till maximum height of blockchain
	for i := minheight; i < maxheight; i++ {
		height := int64(i)
		_, ok := cachedblockStore.Cache.Get(blockMetaKey(height))
		require.Equal(t, ok, true, "Cache hit till maximum height of blockchain")
	}

	height := int64(maxheight)
	_, ok := cachedblockStore.Cache.Get(blockMetaKey(height))
	require.Equal(t, ok, false, "Cache miss at height greater than maximum height of blockchain")

}

func TestGetBlockResultsLRU(t *testing.T) {
	b := NewMockBlockStore()
	cachedblockStore, err := NewLRUBlockStoreCache(200, b)
	require.NoError(t, err)
	height := int64(10)
	//Cache Empty at present resulting in Cache miss
	_, ok := cachedblockStore.Cache.Get(blockResultKey(height))
	require.Equal(t, ok, false, "Cache miss")

	//request for  block Result info at a given height. Data is returned from Mockstore API and is cached in LRU Cache
	blockstoreData, err := cachedblockStore.GetBlockResults(&height)
	require.NoError(t, err)
	require.Equal(t, height, blockstoreData.Height, "Expecting data from API")

	//Block Result info in above provided height gets cached resulting in cache hit
	_, ok = cachedblockStore.Cache.Get(blockResultKey(height))
	require.Equal(t, ok, true, "Cache hit")

	//request for a block at a given height, requested earlier as well, Data is returned from  Cache
	blockstoreData, err = cachedblockStore.GetBlockResults(&height)
	require.NoError(t, err)
	require.Equal(t, height, blockstoreData.Height, "Expecting data from Cache,Block Height stored in structure ctypes.ResultBlock equal to fetched from API for Cache api data accuracy check")

	//request for a block for more than maximum height, error is returned by Cache and no caching occurs
	height = int64(2100)
	blockstoreData, err = cachedblockStore.GetBlockResults(&height)
	require.Error(t, err, "Cache Gives Error as block fetched is greater than maximum height")

	//If a particular height gives error then error is returned by cache as well
	height = int64(15)
	blockstoreData, err = cachedblockStore.GetBlockResults(&height)
	require.Error(t, err, "Cache Gives Error as this is height corresponding to error simulation described in MockBlockStore")

	//request for a block for negative height, error is returned by Cache and no caching occurs
	height = int64(-1)
	blockstoreData, err = cachedblockStore.GetBlockResults(&height)
	require.Error(t, err, "Cache Gives Error as block fetched is for height <= 0")

	//blockresult at maximum height not present in cache
	height = int64(maxHeight)
	_, ok = cachedblockStore.Cache.Get(blockResultKey(height))
	require.Equal(t, ok, false, "Cache miss")

	//request for a block for nil height, maximum height data is returned and cached
	blockstoreData, err = cachedblockStore.GetBlockResults(nil)
	require.NoError(t, err, "Gives maximum height block result info")
	require.Equal(t, int64(maxHeight), blockstoreData.Height, "Expecting blockstore height 50 as maximum height block is fetched")

	//blockresult at maximum height present in cache
	_, ok = cachedblockStore.Cache.Get(blockResultKey(height))
	require.Equal(t, ok, true, "Cache hit")

}

func TestBlockFetchAtHeight2Q(t *testing.T) {
	b := NewMockBlockStore()
	cachedblockStore, err := NewTwoQueueBlockStoreCache(200, b)
	require.NoError(t, err)
	height := int64(19)
	//Cache Empty at present resulting in Cache miss
	_, ok := cachedblockStore.TwoQueueCache.Get(height)
	require.Equal(t, ok, false, "Cache miss")

	//request for a block at a given height. Data is returned from Mockstore API and is cached in LRU Cache
	blockstoreData, err := cachedblockStore.GetBlockByHeight(&height)
	require.NoError(t, err)
	require.Equal(t, height, blockstoreData.Block.Height, "Expecting data from API")

	//Block info in above provided height gets cached resulting in cache hit
	_, ok = cachedblockStore.TwoQueueCache.Get(height)
	require.Equal(t, ok, true, "Cache hit")

	//request for a block at a given height, requested earlier as well, Data is returned from  Cache
	blockstoreData, err = cachedblockStore.GetBlockByHeight(&height)
	require.NoError(t, err)
	require.Equal(t, height, blockstoreData.Block.Height, "Expecting data from Cache,Block Height stored in structure ctypes.ResultBlock equal to fetched from API for Cache api data accuracy check")

	//request for a block for more than maximum height, error is returned by Cache and no caching occurs
	height = int64(2100)
	blockstoreData, err = cachedblockStore.GetBlockByHeight(&height)
	require.Error(t, err, "Cache Gives Error as block fetched is greater than maximum height,this is default functionality of corresponding tendermint blockstore API also")

	//request for a block for negative height, error is returned by Cache and no caching occurs
	height = int64(-1)
	blockstoreData, err = cachedblockStore.GetBlockByHeight(&height)
	require.Error(t, err, "Cache Gives Error as block fetched is for height <= 0,this is default functionality of corresponding tendermint blockstore API also")

	//block at maximum height not present in cache
	height = int64(maxHeight)
	_, ok = cachedblockStore.TwoQueueCache.Get(height)
	require.Equal(t, ok, false, "Cache miss")

	//request for a block for nil height, maximum height data is returned and cached
	blockstoreData, err = cachedblockStore.GetBlockByHeight(nil)
	require.NoError(t, err, "Gives maximum height block")
	require.Equal(t, int64(maxHeight), blockstoreData.Block.Height, "Expecting blockstore height 50 as maximum height block is fetched,this is default functionality of corresponding tendermint blockstore API also")

	//block at maximum height present in cache
	_, ok = cachedblockStore.TwoQueueCache.Get(height)
	require.Equal(t, ok, true, "Cache hit")

}

func TestGetBlockRangeByHeight2Q(t *testing.T) {
	b := NewMockBlockStore()
	cachedblockStore, err := NewTwoQueueBlockStoreCache(200, b)
	require.NoError(t, err)

	minheight := int64(1)
	maxheight := int64(15)

	//Cache Empty at present resulting in Cache miss
	for i := minheight; i <= maxheight; i++ {
		height := int64(i)
		_, ok := cachedblockStore.TwoQueueCache.Get(blockMetaKey(height))
		require.Equal(t, ok, false, "Cache miss")
	}

	//request for a blocks in a given height range. Data is returned from Mockstore API and is cached in LRU Cache
	blockrangeData, err := cachedblockStore.GetBlockRangeByHeight(minheight, maxheight)
	require.NoError(t, err)

	//Blocks returned by MockBlockStore are verified for data accuracy
	for i := minheight; i <= maxheight; i++ {
		require.Equal(t, int64(i), int64(blockrangeData.BlockMetas[i-1].Header.Height), "Expecting height field in blockMetas equal height supplied to API for accuracy check,Expecting Data from API")
	}

	//Block infos in above provided height range gets cached resulting in cache hit for each height in height range
	for i := minheight; i <= maxheight; i++ {
		height := int64(i)
		_, ok := cachedblockStore.TwoQueueCache.Get(blockMetaKey(height))
		require.Equal(t, ok, true, "Cache hit")
	}

	blockrangeData, err = cachedblockStore.GetBlockRangeByHeight(minheight, maxheight)
	require.NoError(t, err)

	//Blocks returned by Cached are verified for data accuracy
	for i := minheight; i <= maxheight; i++ {
		require.Equal(t, int64(i), int64(blockrangeData.BlockMetas[i-1].Header.Height), "Expecting Data from Cache,Expecting height field in blockMetas equal height supplied to API for Cache accuracy check")
	}

	//request for blockinfo for height range where max height is greater than maximum height of blockchain, error is returned by Cache, caching occurs till max height of blockchain
	minheight = int64(45)
	maxheight = int64(51)

	//Initially there is a cache miss
	for i := minheight; i <= maxheight; i++ {
		height := int64(i)
		_, ok := cachedblockStore.TwoQueueCache.Get(blockMetaKey(height))
		require.Equal(t, ok, false, "Cache miss")
	}

	blockrangeData, err = cachedblockStore.GetBlockRangeByHeight(minheight, maxheight)
	require.NoError(t, err, "MockstoreBlock APIs are called,caching of BlockMetas occurs till maximum height of blockchain")

	//Cache hit till maximum height of blockchain
	for i := minheight; i < maxheight; i++ {
		height := int64(i)
		_, ok := cachedblockStore.TwoQueueCache.Get(blockMetaKey(height))
		require.Equal(t, ok, true, "Cache hit till maximum height of blockchain")
	}

	height := int64(maxheight)
	_, ok := cachedblockStore.TwoQueueCache.Get(blockMetaKey(height))
	require.Equal(t, ok, false, "Cache miss at height greater than maximum height of blockchain")

}

func TestGetBlockResults2Q(t *testing.T) {
	b := NewMockBlockStore()
	cachedblockStore, err := NewTwoQueueBlockStoreCache(200, b)
	require.NoError(t, err)
	height := int64(10)
	//Cache Empty at present resulting in Cache miss
	_, ok := cachedblockStore.TwoQueueCache.Get(blockResultKey(height))
	require.Equal(t, ok, false, "Cache miss")

	//request for  block Result info at a given height. Data is returned from Mockstore API and is cached in LRU Cache
	blockstoreData, err := cachedblockStore.GetBlockResults(&height)
	require.NoError(t, err)
	require.Equal(t, height, blockstoreData.Height, "Expecting data from API")

	//Block Result info in above provided height gets cached resulting in cache hit
	_, ok = cachedblockStore.TwoQueueCache.Get(blockResultKey(height))
	require.Equal(t, ok, true, "Cache hit")

	//request for a block at a given height, requested earlier as well, Data is returned from  Cache
	blockstoreData, err = cachedblockStore.GetBlockResults(&height)
	require.NoError(t, err)
	require.Equal(t, height, blockstoreData.Height, "Expecting data from Cache,Block Height stored in structure ctypes.ResultBlock equal to fetched from API for Cache api data accuracy check")

	//request for a block for more than maximum height, error is returned by Cache and no caching occurs
	height = int64(2100)
	blockstoreData, err = cachedblockStore.GetBlockResults(&height)
	require.Error(t, err, "Cache Gives Error as block fetched is greater than maximum height")

	//If a particular height gives error then error is returned by cache as well
	height = int64(15)
	blockstoreData, err = cachedblockStore.GetBlockResults(&height)
	require.Error(t, err, "Cache Gives Error as this is height corresponding to error simulation described in MockBlockStore")

	//request for a block for negative height, error is returned by Cache and no caching occurs
	height = int64(-1)
	blockstoreData, err = cachedblockStore.GetBlockResults(&height)
	require.Error(t, err, "Cache Gives Error as block fetched is for height <= 0")

	//blockresult at maximum height not present in cache
	height = int64(maxHeight)
	_, ok = cachedblockStore.TwoQueueCache.Get(blockResultKey(height))
	require.Equal(t, ok, false, "Cache miss")

	//request for a block for nil height, maximum height data is returned and cached
	blockstoreData, err = cachedblockStore.GetBlockResults(nil)
	require.NoError(t, err, "Gives maximum height block result info")
	require.Equal(t, int64(maxHeight), blockstoreData.Height, "Expecting blockstore height 50 as maximum height block is fetched")

	//blockresult at maximum height present in cache
	_, ok = cachedblockStore.TwoQueueCache.Get(blockResultKey(height))
	require.Equal(t, ok, true, "Cache hit")

}
