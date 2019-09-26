package blockatlas

import (
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/loomnetwork/go-loom"
	"github.com/pkg/errors"
)

// https://github.com/ethereum/wiki/wiki/JSON-RPC#hex-value-encoding
// Eth JSON RPC define three types QUANTITIES, DATA and default block parameter.
// All represented by strings.
type Quantity string
type Data string
type BlockHeight string

const (
	ZeroedQuantity    string = "0x0"
	ZeroedData        string = "0x0"
	ZeroedData8Bytes  string = "0x0000000000000000"
	ZeroedData20Bytes string = "0x0000000000000000000000000000000000000000"
	ZeroedData32Bytes string = "0x0000000000000000000000000000000000000000000000000000000000000000"
	ZeroedData64bytes string = "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	StatusTxSuccess          = "0x1"
)

type JsonTxObject struct {
	Hash             string          `json:"hash,omitempty"`
	TransactionType  string          `json:"transactionType,omitempty"`
	ContractName     string          `json:"contractName,omitempty"`
	ContractMethod   string          `json:"contractMethod,omitempty"`
	Nonce            string          `json:"nonce,omitempty"`
	BlockHash        string          `json:"blockHash,omitempty"`
	BlockNumber      int64           `json:"blockNumber,omitempty"`
	TransactionIndex string          `json:"transactionIndex,omitempty"`
	From             string          `json:"from,omitempty"`
	To               string          `json:"to"`
	Value            json.RawMessage `json:"value"`
	GasPrice         string          `json:"gasPrice,omitempty"`
	Gas              string          `json:"gas,omitempty"`
}

type JsonBlockObject struct {
	Number           int64          `json:"number,omitempty"`
	Hash             string         `json:"hash,omitempty"`
	ParentHash       string         `json:"parentHash,omitempty"`
	Nonce            string         `json:"nonce,omitempty"`
	TransactionsRoot string         `json:"transactionsRoot,omitempty"`
	Size             string         `json:"size,omitempty"`
	GasLimit         string         `json:"gasLimit,omitempty"`
	GasUsed          string         `json:"gasUsed,omitempty"`
	Timestamp        int64          `json:"timestamp,omitempty"`
	Transactions     []JsonTxObject `json:"transactions"`
}

type ApproveValue struct {
	Spender string `json:"spender_address"`
	Amount  string `json:"amount"`
}
type TransferValue struct {
	To     string `json:"to_address"`
	Amount string `json:"amount"`
}

type DelegateValue struct {
	ValidatorAddress string `json:"validator_address"`
	Amount           string `json:"amount"`
	LockTimeTier     uint64 `json:"lock_time_tier"`
	Referrer         string `json:"referrer"`
}

type ReDelegateValue struct {
	ValidatorAddress       string `json:"validator_address"`
	FormerValidatorAddress string `json:"former_validator_address"`
	Index                  uint64 `json:"index"`
	Amount                 string `json:"amount"`
	NewLockTimeTier        uint64 `json:"lock_time_tier"`
	Referrer               string `json:"referrer"`
}

type UnbondValue struct {
	ValidatorAddress string `json:"validator_address"`
	Amount           string `json:"amount"`
	Index            uint64 `json:"index"`
}

// Hex
func EncBytes(value []byte) string {
	bytesStr := "0x" + hex.EncodeToString(value)
	if bytesStr == "0x" {
		bytesStr = "0x0"
	}
	return strings.ToLower(bytesStr)
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
		BlockNumber:      int64(0),
		TransactionIndex: ZeroedQuantity,
		To:               string(ZeroedData32Bytes),
		From:             string(ZeroedData32Bytes),
		Gas:              ZeroedQuantity,
		Value:            nil,
		GasPrice:         ZeroedQuantity,
	}
}

func GetBlockZero() JsonBlockObject {
	blockInfo := JsonBlockObject{
		Number:       int64(0),
		Hash:         "0x0000000000000000000000000000000000000000000000000000000000000001",
		ParentHash:   ZeroedData32Bytes,
		Timestamp:    int64(1526299200), // TODO get the right timestamp, maybe the timestamp for block 0x1
		GasLimit:     ZeroedQuantity,
		GasUsed:      ZeroedQuantity,
		Size:         ZeroedQuantity,
		Transactions: make([]JsonTxObject, 0),
		Nonce:        ZeroedData8Bytes,
	}

	return blockInfo
}
