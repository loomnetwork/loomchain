package plant

import (
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/bloom"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/receipts/factory"
	registry "github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/store"
	"github.com/pkg/errors"
)

type receiptPlant struct {
	createRegistry registry.RegistryFactoryFunc

	readCache  loomchain.ReadReceiptCache
	writeCache loomchain.WriteReceiptCache
}

func NewReceiptPlant(
	eventHandler loomchain.EventHandler,
	createRegistry registry.RegistryFactoryFunc,
) loomchain.ReceiptPlant {
	rc := receiptCache{
		eventHandler: eventHandler,
		txReceipt:    types.EvmTxReceipt{},
	}
	rp := &receiptPlant{createRegistry, &rc, &rc}
	return rp
}

func (r *receiptPlant) ReadCache() *loomchain.ReadReceiptCache {
	return &r.readCache
}

func (r *receiptPlant) WriteCache() *loomchain.WriteReceiptCache {
	return &r.writeCache
}

func (r *receiptPlant) ReceiptReaderFactory() receipts.ReadReceiptHandlerFactoryFunc {
	return factory.NewStateReadReceiptHandlerFactory(r.createRegistry)
}

func (r *receiptPlant) ReciepWriterFactory() loomchain.WriteReceiptHandlerFactoryFunc {
	return factory.NewStateWriteReceiptHandlerFactory(r.createRegistry)
}

func (r *receiptPlant) CommitBloomFilters(state loomchain.State, height uint64) error {
	receiptReader, err := r.ReceiptReaderFactory()(state)
	if err != nil {
		return errors.Wrap(err, "receipt reader")
	}
	txHashList, err := common.GetTxHashList(state, height)
	if err != nil {
		return errors.Wrap(err, "tx hash list")
	}
	var events []*types.EventData
	for _, txHash := range txHashList {
		txReceipt, err := receiptReader.GetReceipt(txHash)
		if err != nil {
			return errors.Wrap(err, "get receipt")
		}
		events = append(events, txReceipt.Logs...)
	}
	filter := bloom.GenBloomFilter(events)
	txHashState := store.PrefixKVStore(loomchain.BloomPrefix, state)
	txHashState.Set(common.BlockHeightToBytes(height), filter)
	return nil
}

type receiptCache struct {
	eventHandler loomchain.EventHandler
	txReceipt    types.EvmTxReceipt
}

func (c *receiptCache) SaveEventsAndHashReceipt(
	state loomchain.State,
	caller, addr loom.Address,
	events []*loomchain.EventData,
	err error,
) ([]byte, error) {
	var errWrite error
	c.txReceipt, errWrite = common.WriteReceipt(state, caller, addr, events, err, c.eventHandler)
	if errWrite != nil {
		if err == nil {
			return nil, errors.Wrap(errWrite, "writing receipt")
		} else {
			return nil, errors.Wrapf(err, "follow up error writing reciept %v", errWrite)
		}
	}
	return c.txReceipt.TxHash, err
}

func (c *receiptCache) Empty() {
	c.txReceipt = types.EvmTxReceipt{}
}

func (c *receiptCache) GetReceipt() types.EvmTxReceipt {
	return c.txReceipt
}
