package receipts

import (
	"github.com/loomnetwork/loomchain"
	legacy_v1 "github.com/loomnetwork/loomchain/receipts/chain/v1"
	legacy_v2 "github.com/loomnetwork/loomchain/receipts/chain/v2"
	"github.com/loomnetwork/loomchain/receipts/handler"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
	"github.com/pkg/errors"
)

type ReceiptReaderWriter interface {
	loomchain.ReadReceiptHandler
	loomchain.WriteReceiptHandler
	loomchain.ReceiptHandlerStore
	Version() handler.ReceiptHandlerVersion
}

type ResolveReceiptHandlerCfg func(blockHeight int64, v2Feature bool) (handler.ReceiptHandlerVersion, uint64, error)

// ReceiptHandlerProvider implements loomchain.ReceiptHandlerProvider interface
type ReceiptHandlerProvider struct {
	eventHandler loomchain.EventHandler
	resolveCfg   ResolveReceiptHandlerCfg
	handler      ReceiptReaderWriter
	evmAuxStore  *evmaux.EvmAuxStore
}

func NewReceiptHandlerProvider(
	eventHandler loomchain.EventHandler, resolveCfg ResolveReceiptHandlerCfg,
	evmAuxStore *evmaux.EvmAuxStore,
) *ReceiptHandlerProvider {
	return &ReceiptHandlerProvider{
		eventHandler: eventHandler,
		resolveCfg:   resolveCfg,
		evmAuxStore:  evmAuxStore,
	}
}

func (h *ReceiptHandlerProvider) StoreAt(blockHeight int64, v2Feature bool) (loomchain.ReceiptHandlerStore, error) {
	return h.resolve(blockHeight, v2Feature)
}

func (h *ReceiptHandlerProvider) ReaderAt(blockHeight int64, v2Feature bool) (loomchain.ReadReceiptHandler, error) {
	return h.resolve(blockHeight, v2Feature)
}

func (h *ReceiptHandlerProvider) WriterAt(blockHeight int64, v2Feature bool) (loomchain.WriteReceiptHandler, error) {
	return h.resolve(blockHeight, v2Feature)
}

// Resolve returns the receipt handler that should be used at the specified block height.
func (h *ReceiptHandlerProvider) resolve(blockHeight int64, v2Feature bool) (ReceiptReaderWriter, error) {
	ver, maxPersistentReceipts, err := h.resolveCfg(blockHeight, v2Feature)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve receipt handler at height %d", blockHeight)
	}
	// Reuse previously created handler if the version hasn't changed
	if (h.handler == nil) || (ver != h.handler.Version()) {
		// TODO: if h.handler != nil then we should probably call h.handler.Close()
		switch ver {
		case handler.ReceiptHandlerLegacyV1:
			h.handler = legacy_v1.NewReceiptHandler(h.eventHandler)

		case handler.ReceiptHandlerLegacyV2:
			h.handler = legacy_v2.NewReceiptHandler(h.eventHandler)

		default:
			handler, err := handler.NewReceiptHandler(ver, h.eventHandler, maxPersistentReceipts, h.evmAuxStore)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to create receipt handler at height %d", blockHeight)
			}
			h.handler = handler
		}
	}
	return h.handler, nil
}
