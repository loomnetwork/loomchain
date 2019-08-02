package common

import (
	"encoding/binary"

	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/pkg/errors"
)

const (
	StatusTxSuccess = int32(1)
	StatusTxFail    = int32(0)
)

var (
	ErrTxReceiptNotFound = errors.New("Tx receipt not found")
)

func BlockHeightToBytes(height uint64) []byte {
	heightB := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightB, height)
	return heightB
}

func ConvertEventData(events []*loomchain.EventData) []*types.EventData {
	typesEvents := make([]*types.EventData, 0, len(events))
	for _, event := range events {
		typeEvent := types.EventData(*event)
		typesEvents = append(typesEvents, &typeEvent)
	}
	return typesEvents
}
