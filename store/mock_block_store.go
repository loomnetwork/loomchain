package store

import (
	"fmt"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/state"
	"github.com/tendermint/tendermint/types"
)

var _ BlockStore = &MockBlockStore{}

type MockBlockStore struct {
	blocks       map[int64]*ctypes.ResultBlock
	blockResults map[int64]*ctypes.ResultBlockResults
}

func NewMockBlockStore() *MockBlockStore {
	return &MockBlockStore{
		blocks:       make(map[int64]*ctypes.ResultBlock),
		blockResults: make(map[int64]*ctypes.ResultBlockResults),
	}
}

func (s *MockBlockStore) GetBlockByHeight(height *int64) (*ctypes.ResultBlock, error) {
	//Taken as max blockchain height
	h := int64(50)
	//Get Height added to emulate error handling and nil height case covered in tendermint blockstore
	h, err := getHeight(h, height)
	if err != nil {
		return nil, err
	}

	if block, ok := s.blocks[h]; ok {
		return block, nil
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

	const limit int64 = 20
	var err error
	//Get filterMinMax added to emulate error handling covered in tendermint blockstore
	minHeight, maxHeight, err = filterMinMax(int64(50), minHeight, maxHeight, limit)
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
	h := int64(50)
	h, err := getHeight(h, height)
	if err != nil {
		return nil, err
	}

	if block, ok := s.blockResults[h]; ok {
		return block, nil
	}
	//To simulate error at a height
	//load the results, error returned for a height value in core tendermint API
	//results, err := sm.LoadABCIResponses(stateDB, height)
	if h == int64(15) {
		return nil, fmt.Errorf("Error Simulation")
	}

	return &ctypes.ResultBlockResults{
		Height:  h,
		Results: nil,
	}, nil
}

func (s *MockBlockStore) SetBlockResults(blockResult *ctypes.ResultBlockResults) {
	s.blockResults[blockResult.Height] = blockResult
}

func (s *MockBlockStore) SetBlock(block *ctypes.ResultBlock) {
	s.blocks[block.Block.Height] = block
}

func (s *MockBlockStore) GetTxResult(_ []byte) (*ctypes.ResultTx, error) {
	return nil, nil
}

func MockBlock(height int64, blockTxHash []byte, txs [][]byte) *ctypes.ResultBlock {
	blockTxs := []types.Tx{}
	for _, tx := range txs {
		blockTxs = append(blockTxs, tx)
	}
	return &ctypes.ResultBlock{
		BlockMeta: &types.BlockMeta{
			BlockID: types.BlockID{
				Hash: blockTxHash,
			},
		},
		Block: &types.Block{
			Data: types.Data{
				Txs: blockTxs,
			},
			Header: types.Header{
				Height: height,
			},
		},
	}
}

func MockBlockResults(height int64, data [][]byte) *ctypes.ResultBlockResults {
	deliverTx := []*abci.ResponseDeliverTx{}
	for _, d := range data {
		res := &abci.ResponseDeliverTx{
			Data: d,
		}
		deliverTx = append(deliverTx, res)
	}
	return &ctypes.ResultBlockResults{
		Height: height,
		Results: &state.ABCIResponses{
			DeliverTx: deliverTx,
		},
	}
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

func filterMinMaxforCache(min, max int64) (int64, int64, error) {
	// filter negatives
	if min < 0 || max < 0 {
		return min, max, fmt.Errorf("heights must be non-negative")
	}

	// adjust for default values
	if min == 0 {
		min = 1
	}

	if min > max {
		return min, max, fmt.Errorf("min height %d can't be greater than max height %d", min, max)
	}
	return min, max, nil
}
