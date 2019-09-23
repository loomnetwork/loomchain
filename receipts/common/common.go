package common

import (
	"encoding/binary"

	"github.com/pkg/errors"
)

const (
	StatusTxSuccess       = int32(1)
	StatusTxFail          = int32(0)
	ReceiptHandlerLevelDb = 2 //ctypes.ReceiptStorage_LEVELDB
	DefaultMaxReceipts    = uint64(2000)
)

var (
	ErrTxReceiptNotFound      = errors.New("Tx receipt not found")
	ErrPendingReceiptNotFound = errors.New("Pending receipt not found")
)

func BlockHeightToBytes(height uint64) []byte {
	heightB := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightB, height)
	return heightB
}
