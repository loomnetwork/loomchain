package store

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
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
	// Get transaction result from Tendermint Tx Hash
	GetTxResult(txHash []byte) (*ctypes.ResultTx, error)
}
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
