// +build evm

package polls

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/store"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
	"github.com/pkg/errors"
)

type EthTxPoll struct {
	startBlock    uint64
	lastBlockRead uint64
	evmAuxStore   *evmaux.EvmAuxStore
	blockStore    store.BlockStore
}

func NewEthTxPoll(height uint64, evmAuxStore *evmaux.EvmAuxStore, blockStore store.BlockStore) *EthTxPoll {
	p := &EthTxPoll{
		startBlock:    height,
		lastBlockRead: height,
		evmAuxStore:   evmAuxStore,
		blockStore:    blockStore,
	}
	return p
}

func (p *EthTxPoll) Poll(
	state loomchain.State,
	id string,
	readReceipts loomchain.ReadReceiptHandler,
	_ *auth.Config,
	_ func(state loomchain.State) (contractpb.StaticContext, error),
) (EthPoll, interface{}, error) {
	if p.lastBlockRead+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}
	lastBlock, results, err := getTxHashes(state, p.lastBlockRead, readReceipts, p.evmAuxStore)
	if err != nil {
		return p, nil, nil
	}
	p.lastBlockRead = lastBlock
	return p, eth.EncBytesArray(results), nil
}

func (p *EthTxPoll) AllLogs(
	state loomchain.State,
	id string,
	readReceipts loomchain.ReadReceiptHandler,
	_ *auth.Config,
	_ func(state loomchain.State) (contractpb.StaticContext, error),
) (interface{}, error) {
	_, results, err := getTxHashes(state, p.startBlock, readReceipts, p.evmAuxStore)
	return eth.EncBytesArray(results), err
}

func getTxHashes(state loomchain.State, lastBlockRead uint64,
	readReceipts loomchain.ReadReceiptHandler, evmAuxStore *evmaux.EvmAuxStore) (uint64, [][]byte, error) {
	var txHashes [][]byte
	for height := lastBlockRead + 1; height < uint64(state.Block().Height); height++ {
		txHashList, err := evmAuxStore.GetTxHashList(height)

		if err != nil {
			return lastBlockRead, nil, errors.Wrapf(err, "reading tx hashes at height %d", height)
		}
		if len(txHashList) > 0 {
			txHashes = append(txHashes, txHashList...)
		}
		lastBlockRead = height
	}
	return lastBlockRead, txHashes, nil
}

func (p *EthTxPoll) LegacyPoll(
	state loomchain.State,
	id string,
	readReceipts loomchain.ReadReceiptHandler,
	_ *auth.Config,
	_ func(state loomchain.State) (contractpb.StaticContext, error),
) (EthPoll, []byte, error) {
	if p.lastBlockRead+1 > uint64(state.Block().Height) {
		return p, nil, nil
	}

	var txHashes [][]byte
	for height := p.lastBlockRead + 1; height < uint64(state.Block().Height); height++ {
		txHashList, err := p.evmAuxStore.GetTxHashList(height)
		if err != nil {
			return p, nil, errors.Wrapf(err, "reading tx hash at heght %d", height)
		}
		if len(txHashList) > 0 {
			txHashes = append(txHashes, txHashList...)
		}
	}
	p.lastBlockRead = uint64(state.Block().Height)

	blocksMsg := types.EthFilterEnvelope_EthTxHashList{
		EthTxHashList: &types.EthTxHashList{EthTxHash: txHashes},
	}
	r, err := proto.Marshal(&types.EthFilterEnvelope{Message: &blocksMsg})
	return p, r, err
}
