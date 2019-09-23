package blockatlas

import (
	"encoding/hex"
	"math/big"
	"strconv"
	"strings"

	"github.com/loomnetwork/go-loom"
	ltypes "github.com/loomnetwork/go-loom/types"
	"github.com/pkg/errors"
)

// https://github.com/ethereum/wiki/wiki/JSON-RPC#hex-value-encoding
// Eth JSON RPC define three types QUANTITIES, DATA and default block parameter.
// All represented by strings.
type Quantity string
type Data string
type BlockHeight string

const (
	ZeroedQuantity     Quantity = "0x0"
	ZeroedData         Data     = "0x0"
	ZeroedData8Bytes   Data     = "0x0000000000000000"
	ZeroedData20Bytes  Data     = "0x0000000000000000000000000000000000000000"
	ZeroedData32Bytes  Data     = "0x0000000000000000000000000000000000000000000000000000000000000000"
	ZeroedData64bytes  Data     = "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	ZeroedData256Bytes Data     = "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	StatusTxSuccess = "0x1"
)

type JsonTxObject struct {
	Hash             Data        `json:"hash,omitempty"`
	TransactionType  string      `json:"transactionType,omitempty"`
	ContractName     string      `json:"contractName,omitempty"`
	ContractMethod   string      `json:"contractMethod,omitempty"`
	Nonce            Quantity    `json:"nonce,omitempty"`
	BlockHash        Data        `json:"blockHash,omitempty"`
	BlockNumber      Quantity    `json:"blockNumber,omitempty"`
	TransactionIndex Quantity    `json:"transactionIndex,omitempty"`
	From             string      `json:"from,omitempty"`
	To               string      `json:"to"`
	Value            interface{} `json:"value"`
	GasPrice         Quantity    `json:"gasPrice,omitempty"`
	Gas              Quantity    `json:"gas,omitempty"`
}

type JsonBlockObject struct {
	Number           Quantity       `json:"number,omitempty"`
	Hash             Data           `json:"hash,omitempty"`
	ParentHash       Data           `json:"parentHash,omitempty"`
	Nonce            Data           `json:"nonce,omitempty"`
	TransactionsRoot Data           `json:"transactionsRoot,omitempty"`
	Size             Quantity       `json:"size,omitempty"`
	GasLimit         Quantity       `json:"gasLimit,omitempty"`
	GasUsed          Quantity       `json:"gasUsed,omitempty"`
	Timestamp        Quantity       `json:"timestamp,omitempty"`
	Transactions     []JsonTxObject `json:"transactions,omitempty"`
}

type DelegateValue struct {
	ValidatorAddress Data     `json:"validator_address,omitempty"`
	Amount           Quantity `json:"amount,omitempty"`
	LockTimeTier     Quantity `json:"lock_time_tier,omitempty"`
	Referrer         Data     `json:"referrer,omitempty"`
}

type ReDelegateValue struct {
	ValidatorAddress       Data     `json:"validator_address,omitempty"`
	FormerValidatorAddress Data     `former_validator_address,omitempty`
	Index                  Quantity `json:"index,omitempty"`
	Amount                 Quantity `json:"amount,omitempty"`
	NewLockTimeTier        Quantity `json:"lock_time_tier,omitempty"`
	Referrer               Data     `json:"referrer,omitempty"`
}

func EncInt(value int64) Quantity {
	return Quantity("0x" + strconv.FormatInt(value, 16))
}

func EncUint(value uint64) Quantity {
	return Quantity("0x" + strconv.FormatUint(value, 16))
}

func EncBigInt(value big.Int) Quantity {
	return Quantity("0x" + value.Text(16))
}

// Hex
func EncBytes(value []byte) Data {
	bytesStr := "0x" + hex.EncodeToString(value)
	if bytesStr == "0x" {
		bytesStr = "0x0"
	}
	return Data(strings.ToLower(bytesStr))
}

// Ptr to Hex
func EncPtrBytes(value []byte) *Data {
	if len(value) == 0 {
		return nil
	}
	bytesStr := "0x" + hex.EncodeToString(value)
	if bytesStr == "0x" {
		bytesStr = "0x0"
	}
	data := Data(strings.ToLower(bytesStr))
	return &data
}

func EncPtrData(value Data) *Data {
	if len(value) == 0 {
		return nil
	}
	return &value
}

func EncBytesArray(list [][]byte) []Data {
	dataArray := []Data{}
	for _, hash := range list {
		dataArray = append(dataArray, EncBytes(hash))
	}
	return dataArray
}

func EncAddress(value *ltypes.Address) Data {
	if value == nil {
		return ZeroedData
	} else {
		return EncBytes([]byte(value.Local))
	}
}

func EncPtrAddress(value *ltypes.Address) *Data {
	if value == nil {
		return nil
	} else {
		data := EncBytes([]byte(value.Local))
		return &data
	}
}

type EthBlockFilter struct {
	Addresses []loom.LocalAddress
	Topics    [][]string
}

type EthFilter struct {
	EthBlockFilter
	FromBlock BlockHeight
	ToBlock   BlockHeight
}

func DecQuantityToInt(value Quantity) (int64, error) {
	if len(value) <= 2 || value[0:2] != "0x" {
		return 0, errors.Errorf("invalid quantity format: %v", value)
	}
	return strconv.ParseInt(string(value), 0, 64)
}

func DecQuantityToUint(value Quantity) (uint64, error) {
	if len(value) <= 2 || value[0:2] != "0x" {
		return 0, errors.Errorf("invalid quantity format: %v", value)
	}
	return strconv.ParseUint(string(value), 0, 64)
}

func DecDataToBytes(value Data) ([]byte, error) {
	if len(value) <= 2 || value[0:2] != "0x" {
		return []byte{}, errors.Errorf("invalid data format: %v", value)
	}
	return hex.DecodeString(string(value[2:]))
}

func DecDataToAddress(chainID string, value Data) (loom.Address, error) {
	local, err := loom.LocalAddressFromHexString(string(value))
	if err != nil {
		return loom.Address{}, err
	}
	return loom.Address{
		ChainID: chainID,
		Local:   local,
	}, nil
}

func DecBlockHeight(lastBlockHeight int64, value BlockHeight) (uint64, error) {
	if lastBlockHeight < 1 {
		return 0, errors.Errorf("invalid last block height %v", lastBlockHeight)
	}

	switch value {
	case "earliest":
		return 1, nil
	case "genesis":
		return 1, nil
	case "latest":
		if (lastBlockHeight) > 0 {
			return uint64(lastBlockHeight), nil
		} else {
			return 0, errors.New("no block completed yet")
		}
	case "pending":
		return uint64(lastBlockHeight + 1), nil
	default:
		height, err := strconv.ParseUint(string(value), 0, 64)
		if err != nil {
			return 0, errors.Wrap(err, "parse block height")
		}
		if height > uint64(lastBlockHeight+1) {
			return 0, errors.Errorf("requested block height %v exceeds pending block height %v", height, lastBlockHeight+1)
		}
		if height == 0 {
			return 0, errors.Errorf("zero block height is not valid")
		}
		return height, nil
	}
}

func GetEmptyTxObject() JsonTxObject {
	return JsonTxObject{
		Hash:             ZeroedData64bytes,
		Nonce:            ZeroedQuantity,
		BlockHash:        ZeroedData64bytes,
		BlockNumber:      ZeroedQuantity,
		TransactionIndex: ZeroedQuantity,
		To:               string(ZeroedData32Bytes),
		From:             string(ZeroedData32Bytes),
		Gas:              ZeroedQuantity,
		Value:            ZeroedQuantity,
		GasPrice:         ZeroedQuantity,
	}
}

func GetBlockZero() JsonBlockObject {
	blockInfo := JsonBlockObject{
		Number:       ZeroedQuantity,
		Hash:         "0x0000000000000000000000000000000000000000000000000000000000000001",
		ParentHash:   ZeroedData32Bytes,
		Timestamp:    "0x5af97a40", // TODO get the right timestamp, maybe the timestamp for block 0x1
		GasLimit:     ZeroedQuantity,
		GasUsed:      ZeroedQuantity,
		Size:         ZeroedQuantity,
		Transactions: make([]JsonTxObject, 0),
		Nonce:        ZeroedData8Bytes,
	}

	return blockInfo
}
