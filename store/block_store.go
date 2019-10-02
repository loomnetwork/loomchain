package store

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
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
	// Get Transaction Results from Tendermint Tx Hash
	GetTxResult(txHash []byte) (*ctypes.ResultTx, error)
}

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

	if block, ok := s.blocks[*height]; ok {
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

	if block, ok := s.blockResults[*height]; ok {
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

var _ BlockStore = &MockBlockStore{}

type TendermintBlockStore struct {
}

var _ BlockStore = &TendermintBlockStore{}

func NewTendermintBlockStore() BlockStore {
	return &TendermintBlockStore{}
}

type BlockStoreConfig struct {
	// Valid values: None | LRU | 2Q
	CacheAlgorithm string
	CacheSize      int64
}

func DefaultBlockStoreConfig() *BlockStoreConfig {
	return &BlockStoreConfig{
		CacheAlgorithm: "None",
		CacheSize:      10000, //Size should be more because of blockrangebyheight API
	}
}

func NewBlockStore(cfg *BlockStoreConfig) (BlockStore, error) {
	var err error
	blockStore := NewTendermintBlockStore()

	if strings.EqualFold(cfg.CacheAlgorithm, "LRU") {
		blockStore, err = NewLRUBlockStoreCache(cfg.CacheSize, blockStore)
	} else if strings.EqualFold(cfg.CacheAlgorithm, "2Q") {
		blockStore, err = NewTwoQueueBlockStoreCache(cfg.CacheSize, blockStore)
	} else if !strings.EqualFold(cfg.CacheAlgorithm, "None") {
		return nil, fmt.Errorf("Invalid value '%s' for BlockStore.CacheAlgorithm config setting", cfg.CacheAlgorithm)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create %s block store cache", cfg.CacheAlgorithm)
	}
	return blockStore, nil

}

func (s *TendermintBlockStore) GetBlockByHeight(height *int64) (*ctypes.ResultBlock, error) {
	blockResult, err := core.Block(height)
	if err != nil {
		return nil, err
	}
	if blockResult.BlockMeta == nil || blockResult.Block == nil {
		return nil, errors.New("block not found")
	}

	header := types.Header{
		Height:          blockResult.Block.Header.Height,
		LastBlockID:     blockResult.Block.Header.LastBlockID,
		Time:            blockResult.Block.Header.Time,
		ProposerAddress: blockResult.Block.Header.ProposerAddress,
	}
	blockMeta := types.BlockMeta{
		BlockID: blockResult.BlockMeta.BlockID,
		Header:  header,
	}
	block := types.Block{
		Header: header,
		Data:   blockResult.Block.Data,
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
	blockMetas := []*types.BlockMeta{}
	for _, meta := range blockResult.BlockMetas {
		if meta != nil {
			blockMetas = append(blockMetas, meta)
		}
	}
	blockchaininfo := ctypes.ResultBlockchainInfo{
		BlockMetas: blockMetas,
	}
	return &blockchaininfo, nil

}

func (s *TendermintBlockStore) GetBlockResults(height *int64) (*ctypes.ResultBlockResults, error) {
	blockResult, err := core.BlockResults(height)
	if err != nil {
		return nil, err
	}
	return &ctypes.ResultBlockResults{
		Results: &state.ABCIResponses{
			DeliverTx: blockResult.Results.DeliverTx,
		},
	}, nil
}

func (s *TendermintBlockStore) GetTxResult(txHash []byte) (*ctypes.ResultTx, error) {
	txResult, err := core.Tx(txHash, false)
	if err != nil {
		return nil, err
	}
	return &ctypes.ResultTx{
		Index:  txResult.Index,
		Height: txResult.Height,
		TxResult: abci.ResponseDeliverTx{
			Code: txResult.TxResult.Code,
			Data: txResult.TxResult.Data,
			Info: txResult.TxResult.Info,
		},
	}, nil
}

func blockMetaKey(height int64) string {
	return "M" + strconv.FormatInt(height, 10)
}

func blockResultKey(height int64) string {
	return "R" + strconv.FormatInt(height, 10)
}

func txHashKey(hash []byte) string {
	return "H" + string(hash)
}
