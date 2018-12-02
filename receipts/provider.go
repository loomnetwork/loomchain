package receipts

import (
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/receipts/handler"
	"github.com/pkg/errors"
)

type ResolveReceiptHandlerCfg func(blockHeight int64) (handler.ReceiptHandlerVersion, uint64, error)

// ReceiptHandlerProvider implements loomchain.ReceiptHandlerProvider interface
type ReceiptHandlerProvider struct {
	eventHandler loomchain.EventHandler
	resolveCfg   ResolveReceiptHandlerCfg
	handler      *handler.ReceiptHandler
}

func NewReceiptHandlerProvider(
	eventHandler loomchain.EventHandler, resolveCfg ResolveReceiptHandlerCfg,
) *ReceiptHandlerProvider {
	return &ReceiptHandlerProvider{
		eventHandler: eventHandler,
		resolveCfg:   resolveCfg,
	}
}

func (h *ReceiptHandlerProvider) StoreAt(blockHeight int64) (loomchain.ReceiptHandlerStore, error) {
	return h.resolve(blockHeight)
}

func (h *ReceiptHandlerProvider) ReaderAt(blockHeight int64) (loomchain.ReadReceiptHandler, error) {
	return h.resolve(blockHeight)
}

func (h *ReceiptHandlerProvider) WriterAt(blockHeight int64) (loomchain.WriteReceiptHandler, error) {
	return h.resolve(blockHeight)
}

// Resolve returns the receipt handler that should be used at the specified block height.
func (h *ReceiptHandlerProvider) resolve(blockHeight int64) (*handler.ReceiptHandler, error) {
	ver, maxPersistentReceipts, err := h.resolveCfg(blockHeight)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve receipt handler at height %d", blockHeight)
	}
	// Reuse previously created handler if the version hasn't changed
	if (h.handler == nil) || (ver != h.handler.Version()) {
		handler, err := handler.NewReceiptHandler(ver, h.eventHandler, maxPersistentReceipts)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create receipt handler at height %d", blockHeight)
		}
		h.handler = handler
	}
	return h.handler, nil
}
