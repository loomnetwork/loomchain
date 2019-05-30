package eth

import (
	"encoding/hex"
	"reflect"
	"strconv"
	"strings"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
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

	StatusTxFail    = "0x0"
	StatusTxSuccess = "0x1"
)

type JsonLog struct {
	Removed          bool     `json:"removed,omitempty"`
	LogIndex         Quantity `json:"logIndex,omitempty"`
	TransactionIndex Quantity `json:"transactionIndex,omitempty"`
	TransactionHash  Data     `json:"transactionHash,omitempty"`
	BlockHash        Data     `json:"blockHash,omitempty"`
	BlockNumber      Quantity `json:"blockNumber,omitempty"`
	Address          Data     `json:"address,omitempty"`
	Data             Data     `json:"data,omitempty"`
	Topics           []Data   `json:"topics,omitempty"`
	BlockTime        Quantity `json:"blockTime,omitempty"`
}

type JsonTxReceipt struct {
	TxHash            Data      `json:"transactionHash,omitempty"`
	TransactionIndex  Quantity  `json:"transactionIndex,omitempty"`
	BlockHash         Data      `json:"blockHash,omitempty"`
	BlockNumber       Quantity  `json:"blockNumber,omitempty"`
	CallerAddress     Data      `json:"from,omitempty"`
	CumulativeGasUsed Quantity  `json:"cumulativeGasUsed,omitempty"`
	GasUsed           Quantity  `json:"gasUsed,omitempty"`
	ContractAddress   Data      `json:"contractAddress,omitempty"`
	Logs              []JsonLog `json:"logs"`
	LogsBloom         Data      `json:"logsBloom,omitempty"`
	Status            Quantity  `json:"status,omitempty"`
}

type JsonTxObject struct {
	Hash             Data     `json:"hash,omitempty"`
	Nonce            Quantity `json:"nonce,omitempty"`
	BlockHash        Data     `json:"blockHash,omitempty"`
	BlockNumber      Quantity `json:"blockNumber,omitempty"`
	TransactionIndex Quantity `json:"transactionIndex,omitempty"`
	From             Data     `json:"from,omitempty"`
	To               Data     `json:"to,omitempty"`
	Value            Quantity `json:"value,omitempty"`
	GasPrice         Quantity `json:"gasPrice,omitempty"`
	Gas              Quantity `json:"gas,omitempty"`
	Input            Data     `json:"input,omitempty"`
}

type JsonBlockObject struct {
	Number           Quantity      `json:"number,omitempty"`
	Hash             Data          `json:"hash,omitempty"`
	ParentHash       Data          `json:"parentHash,omitempty"`
	Nonce            Data          `json:"nonce,omitempty"`
	Sha3Uncles       Data          `json:"sha3Uncles,omitempty"`
	LogsBloom        Data          `json:"logsBloom,omitempty"`
	TransactionsRoot Data          `json:"transactionsRoot,omitempty"`
	StateRoot        Data          `json:"stateRoot,omitempty"`
	ReceiptsRoot     Data          `json:"receiptsRoot,omitempty"`
	Miner            Data          `json:"miner,omitempty"`
	Difficulty       Quantity      `json:"difficulty,omitempty"`
	TotalDifficulty  Quantity      `json:"totalDifficulty,omitempty"`
	ExtraData        Data          `json:"extraData,omitempty"`
	Size             Quantity      `json:"size,omitempty"`
	GasLimit         Quantity      `json:"gasLimit,omitempty"`
	GasUsed          Quantity      `json:"gasUsed,omitempty"`
	Timestamp        Quantity      `json:"timestamp,omitempty"`
	Transactions     []interface{} `json:"transactions"` // Data or []Data
	Uncles           []Data        `json:"uncles"`
}

type JsonTxCallObject struct {
	From     Data     `json:"from,omitempty"`
	To       Data     `json:"to,omitempty"`
	Gas      Quantity `json:"gas,omitempty"`
	GasPrice Quantity `json:"gasPrice,omitempty"`
	Value    Quantity `json:"value,omitempty"`
	Data     Data     `json:"data,omitempty"`
	Nonce    Quantity `json:"nonce.omitempty"`
}

type JsonFilter struct {
	FromBlock BlockHeight   `json:"fromBlock,omitempty"`
	ToBlock   BlockHeight   `json:"toBlock,omitempty"`
	Address   interface{}   `json:"address,omitempty"` // Data or []Data
	Topics    []interface{} `json:"topics,omitempty"`  // (Data or nil or []Data)
	BlockHash Data          `json:"blockhash,omitempty"`
}

func EncTxReceipt(receipt types.EvmTxReceipt) JsonTxReceipt {
	return JsonTxReceipt{
		TransactionIndex:  EncInt(int64(receipt.TransactionIndex)),
		BlockHash:         EncBytes(receipt.BlockHash),
		BlockNumber:       EncInt(receipt.BlockNumber),
		CumulativeGasUsed: EncInt(int64(receipt.CumulativeGasUsed)),
		GasUsed:           EncInt(int64(receipt.GasUsed)),
		ContractAddress:   EncBytes(receipt.ContractAddress),
		Logs:              EncEvents(receipt.Logs),
		LogsBloom:         EncBytes(receipt.LogsBloom),
		Status:            EncInt(int64(receipt.Status)),
		TxHash:            EncBytes(receipt.TxHash),
		CallerAddress:     EncAddress(receipt.CallerAddress),
	}
}

func TxObjToReceipt(txObj JsonTxObject) JsonTxReceipt {
	return JsonTxReceipt{
		TransactionIndex:  txObj.TransactionIndex,
		BlockHash:         txObj.BlockHash,
		BlockNumber:       txObj.BlockNumber,
		CumulativeGasUsed: txObj.Gas,
		GasUsed:           txObj.Gas,
		ContractAddress:   txObj.To,
		Logs:              make([]JsonLog, 0),
		LogsBloom:         ZeroedData8Bytes,
		Status:            StatusTxSuccess,
		TxHash:            txObj.Hash,
		CallerAddress:     txObj.From,
	}
}

func EncEvents(logs []*types.EventData) []JsonLog {

	jLogs := make([]JsonLog, 0, len(logs))
	for i, log := range logs {
		jLog := EncEvent(*log)
		jLog.LogIndex = EncInt(int64(i))
		jLogs = append(jLogs, jLog)
	}

	if len(jLogs) == 0 {
		return make([]JsonLog, 0)
	}

	return jLogs
}

func EncEvent(log types.EventData) JsonLog {
	data := ZeroedData64bytes
	if len(log.EncodedBody) > 0 {
		data = EncBytes(log.EncodedBody)
	}

	jLog := JsonLog{
		TransactionHash:  EncBytes(log.TxHash),
		BlockNumber:      EncUint(log.BlockHeight),
		Address:          EncAddress(log.Caller),
		Data:             data,
		TransactionIndex: EncInt(int64(log.TransactionIndex)),
		BlockHash:        EncBytes(log.BlockHash),
	}
	for _, topic := range log.Topics {
		jLog.Topics = append(jLog.Topics, Data(topic))
	}

	return jLog
}

func EncLogs(logs []*types.EthFilterLog) []JsonLog {

	jLogs := make([]JsonLog, 0, len(logs))
	for _, log := range logs {
		jLogs = append(jLogs, EncLog(*log))
	}
	return jLogs
}

func EncLog(log types.EthFilterLog) JsonLog {
	jLog := JsonLog{
		Removed:          log.Removed,
		LogIndex:         EncInt(log.LogIndex),
		TransactionIndex: EncInt(int64(log.TransactionIndex)),
		TransactionHash:  EncBytes(log.TransactionHash),
		BlockHash:        EncBytes(log.BlockHash),
		BlockNumber:      EncInt(log.BlockNumber),
		Address:          EncBytes(log.Address),
		Data:             EncBytes(log.Data),
	}
	for _, topic := range log.Topics {
		jLog.Topics = append(jLog.Topics, Data(string(topic)))
	}
	return jLog
}

func EncInt(value int64) Quantity {
	return Quantity("0x" + strconv.FormatInt(value, 16))
}

func EncUint(value uint64) Quantity {
	return Quantity("0x" + strconv.FormatUint(value, 16))
}

// Hex
func EncBytes(value []byte) Data {
	bytesStr := "0x" + hex.EncodeToString(value)
	if bytesStr == "0x" {
		bytesStr = "0x0"
	}
	return Data(strings.ToLower(bytesStr))
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

type EthBlockFilter struct {
	Addresses []loom.LocalAddress
	Topics    [][]string
}

type EthFilter struct {
	EthBlockFilter
	FromBlock BlockHeight
	ToBlock   BlockHeight
}

func DecLogFilter(filter JsonFilter) (resp EthFilter, err error) {
	addresses := []loom.LocalAddress{}
	if filter.Address != nil {
		addrValue := reflect.ValueOf(filter.Address)
		switch addrValue.Kind() {
		case reflect.String:
			{
				addrValue := reflect.ValueOf(filter.Address)
				address, err := DecDataToBytes(Data(addrValue.String()))
				if err != nil {
					return resp, errors.Wrapf(err, "unwrap filter address %s", addrValue.String())
				}
				if len(address) > 0 {
					addresses = append(addresses, address)
				}
			}
		case reflect.Slice:
			{
				for i := 0; i < addrValue.Len(); i++ {
					kind := addrValue.Index(i).Kind()
					if kind != reflect.Ptr && kind != reflect.Interface {
						return resp, errors.Errorf("unrecognised address format %v", filter.Address)
					}
					addr := addrValue.Index(i).Elem()
					if addr.Kind() != reflect.String {
						return resp, errors.Errorf("unrecognised address format %v", addr)
					}
					address, err := DecDataToBytes(Data(addr.String()))
					if err != nil {
						return resp, errors.Wrapf(err, "unwrap filter address %s", addr.String())
					}
					if len(address) > 0 {
						addresses = append(addresses, address)
					}
				}
			}
		default:
			return resp, errors.Errorf("filter: unrecognised address format %v", filter.Address)
		}
	}

	topicsList := [][]string{}
	for _, topicInterface := range filter.Topics {
		topics := []string{}
		if topicInterface != nil {
			topicValue := reflect.ValueOf(topicInterface)
			switch topicValue.Kind() {
			case reflect.String:
				if len(topicValue.String()) > 0 {
					topics = append(topics, topicValue.String())
				}
			case reflect.Slice:
				{
					for i := 0; i < topicValue.Len(); i++ {
						kind := topicValue.Index(i).Kind()
						if kind == reflect.Ptr || kind == reflect.Interface {
							topic := topicValue.Index(i).Elem()
							if topic.Kind() == reflect.String {
								if len(topic.String()) > 0 {
									topics = append(topics, topic.String())
								}
							} else {
								return resp, errors.Errorf("unrecognised topic format %v", topic)
							}
						} else {
							return resp, errors.Errorf("unrecognised topic format %v", topicValue)
						}
					}
				}
			case reflect.Invalid:
				return resp, errors.Errorf("invalid topic format")
			default:
				return resp, errors.Errorf("unrecognised topic format %v", topicValue)
			}
		}
		topicsList = append(topicsList, topics)
	}

	ethFilter := EthFilter{
		FromBlock: filter.FromBlock,
		ToBlock:   filter.ToBlock,
		EthBlockFilter: EthBlockFilter{
			Addresses: addresses,
			Topics:    topicsList,
		},
	}
	if len(filter.FromBlock) > 0 {
		ethFilter.FromBlock = filter.FromBlock
	} else {
		ethFilter.FromBlock = "earliest"
	}
	if len(filter.ToBlock) > 0 {
		ethFilter.ToBlock = filter.ToBlock
	} else {
		ethFilter.ToBlock = "pending"
	}
	return ethFilter, nil
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

func GetBlockZero() JsonBlockObject {
	blockInfo := JsonBlockObject{
		Number:           ZeroedQuantity,
		Hash:             "0x0000000000000000000000000000000000000000000000000000000000000001",
		ParentHash:       ZeroedData32Bytes,
		Timestamp:        "0x5af97a40", // TODO get the right timestamp, maybe the timestamp for block 0x1
		GasLimit:         ZeroedQuantity,
		GasUsed:          ZeroedQuantity,
		Size:             ZeroedQuantity,
		Transactions:     nil,
		Nonce:            ZeroedData8Bytes,
		Sha3Uncles:       ZeroedData32Bytes,
		TransactionsRoot: ZeroedData32Bytes,
		StateRoot:        ZeroedData32Bytes,
		ReceiptsRoot:     ZeroedData32Bytes,
		Miner:            ZeroedData20Bytes,
		Difficulty:       ZeroedQuantity,
		TotalDifficulty:  ZeroedQuantity,
		ExtraData:        ZeroedData,
		Uncles:           []Data{},
		LogsBloom:        ZeroedData,
	}

	blockInfo.Transactions = make([]interface{}, 0)

	return blockInfo
}
