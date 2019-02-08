package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/types"
)

func TestCachedBlockStore(t *testing.T) {

	b := NewMockBlockStore()
	db := dbm.NewMemDB()
	b.Db = db
	for h := int64(1); h < 20; h++ {

		lastCommit := &types.Commit{
			Precommits: []*types.Vote{{
				Height:    h - 1,
				Timestamp: time.Now(),
			}},
		}

		block := types.MakeBlock(h, nil, lastCommit, nil)
		b.SaveBlock(block, block.MakePartSet(2), lastCommit)
	}

	BlockFetchAtHeight(t, b)
	GetBlockRangeByHeight(t, b)

}

func BlockFetchAtHeight(t *testing.T, blockstore *MockBlockStore) {

	cachedblockStore, err := NewLRUCacheBlockStore(200, blockstore)

	require.NoError(t, err)

	height := int64(19)

	//request for a block for more than maximum height
	blockstoreData, err := cachedblockStore.GetBlockByHeight(&height)
	require.NoError(t, err)
	require.Equal(t, height, blockstoreData.Block.Height, "Expecting Block Height stored in structure ctypes.ResultBlock equal to fetched from API for cache api data accuracy check")

	height = int64(20)

	blockstoreData, err = cachedblockStore.GetBlockByHeight(&height)
	require.Error(t, err, "Gives Error which can be returned when height for which block info is fetched supplied is greater than current max block height to avoid panic errors")

	lastCommit := &types.Commit{
		Precommits: []*types.Vote{{
			Height:    20 - 1,
			Timestamp: time.Now(),
		}},
	}

	block := types.MakeBlock(20, nil, lastCommit, nil)

	blockstore.SaveBlock(block, block.MakePartSet(2), lastCommit)
	require.Equal(t, blockstore.Height(), block.Header.Height, "expecting the new height to be changed")

	//Fetch block after block is added at given height
	blockstoreData, err = cachedblockStore.GetBlockByHeight(&height)
	require.NoError(t, err, "Expecting Block Height stored in structure ctypes.ResultBlock equal to fetched from API for cache api data accuracy check after new block save")

}

func GetBlockRangeByHeight(t *testing.T, blockstore *MockBlockStore) {

	cachedblockStore, err := NewLRUCacheBlockStore(200, blockstore)

	require.NoError(t, err)

	minheight := int64(10)
	maxheight := int64(20)

	blockrangeData, err := cachedblockStore.GetBlockRangeByHeight(minheight, maxheight)
	require.NoError(t, err)
	require.Equal(t, (maxheight-minheight)+1, int64(len(blockrangeData.BlockMetas)), "Expecting Number of blockMetas equal to difference in height for cache api data accuracy check")

	minheight = int64(10)

	maxheight = int64(21)

	blockrangeData, err = cachedblockStore.GetBlockRangeByHeight(minheight, maxheight)
	require.Error(t, err, "Gives Error which can be returned when max height supplied is greater than current max block height to avoid panic errors")

	lastCommit := &types.Commit{
		Precommits: []*types.Vote{{
			Height:    21 - 1,
			Timestamp: time.Now(),
		}},
	}

	block := types.MakeBlock(21, nil, lastCommit, nil)

	blockstore.SaveBlock(block, block.MakePartSet(2), lastCommit)
	require.Equal(t, blockstore.Height(), block.Header.Height, "expecting the new height to be changed")

	minheight = int64(10)

	maxheight = int64(21)

	blockrangeData, err = cachedblockStore.GetBlockRangeByHeight(minheight, maxheight)
	require.NoError(t, err)
	require.Equal(t, (maxheight-minheight)+1, int64(len(blockrangeData.BlockMetas)), "Expecting Number of blockMetas equal to difference in height to test block range cache api accuracy after new block save")

}
