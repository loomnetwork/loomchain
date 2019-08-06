package receipts

import (
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/receipts/handler"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
)

type ReceiptReaderWriter interface {
	loomchain.ReadReceiptHandler
	loomchain.WriteReceiptHandler
	loomchain.ReceiptHandlerStore
}

// ReceiptHandlerProvider implements loomchain.ReceiptHandlerProvider interface
type ReceiptHandlerProvider struct {
	eventHandler loomchain.EventHandler
	handler      ReceiptReaderWriter
	evmAuxStore  *evmaux.EvmAuxStore
}

func NewReceiptHandlerProvider(
	eventHandler loomchain.EventHandler,
	maxPersistentReceipts uint64,
	evmAuxStore *evmaux.EvmAuxStore,
) *ReceiptHandlerProvider {
	return &ReceiptHandlerProvider{
		eventHandler: eventHandler,
		evmAuxStore:  evmAuxStore,
		handler:      handler.NewReceiptHandler(eventHandler, maxPersistentReceipts, evmAuxStore),
	}
}

func (h *ReceiptHandlerProvider) Store() loomchain.ReceiptHandlerStore {
	return h.handler
}

func (h *ReceiptHandlerProvider) Reader() loomchain.ReadReceiptHandler {
	return h.handler
}

func (h *ReceiptHandlerProvider) Writer() loomchain.WriteReceiptHandler {
	return h.handler
}
