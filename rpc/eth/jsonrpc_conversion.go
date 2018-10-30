package eth

import (
	"encoding/hex"
	"strconv"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	ltypes "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/pkg/errors"
)

// https://github.com/ethereum/wiki/wiki/JSON-RPC#hex-value-encoding
// Eth JSON RPC define three types QUANTITIES, DATA and default block parameter.
// All represented by strings.
type Quantity string
type Data string
type BlockHeight string

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
}

type JsonTxReceipt struct {
	TransactionIndex  Quantity  `json:"transactionIndex,omitempty"`
	BlockHash         Data      `json:"blockHash,omitempty"`
	BlockNumber       Quantity  `json:"blockNumber,omitempty"`
	CumulativeGasUsed Quantity  `json:"cumulativeGasUsed,omitempty"`
	GasUsed           Quantity  `json:"gasUsed,omitempty"`
	ContractAddress   Data      `json:"to,omitempty"`
	Logs              []JsonLog `json:"logs,omitempty"`
	LogsBloom         Data      `json:"logsBloom,omitempty"`
	Status            Quantity  `json:"status,omitempty"`
	TxHash            Data      `json:"transactionHash,omitempty"`
	CallerAddress     Data      `json:"from,omitempty"`
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
	Size_            Quantity      `json:"size,omitempty"`
	GasLimit         Quantity      `json:"gasLimit,omitempty"`
	GasUsed          Quantity      `json:"gasUsed,omitempty"`
	Timestamp        Quantity      `json:"timestamp,omitempty"`
	Transactions     []interface{} `json:"transactions,omitempty"`
	Uncles           []Data        `json:"uncles,omitempty"`
}

type JsonTxCallObject struct {
	From     Data     `json:"from,omitempty"`
	To       Data     `json:"to,omitempty"`
	Gas      Quantity `json:"gas,omitempty"`
	GasPrice Quantity `json:"gasPrice,omitempty"`
	Value    Quantity `json:"value,omitempty"`
	Data     Data     `json:"data,omitempty"`
}

type JsonFilter struct {
	FromBlock BlockHeight `json:"fromBlock,omitempty"`
	ToBlock   BlockHeight `json:"toBlock,omitempty"`
	Address   []Data      `json:"address,omitempty"`
	Topics    [][]Data    `json:"topics,omitempty"`
	BlockHash Data        `json:"blockhash,omitempty"`
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

func EncEvents(logs []*types.EventData) []JsonLog {
	var jLogs []JsonLog
	for i, log := range logs {
		jLog := EncEvent(*log)
		jLog.LogIndex = EncInt(int64(i))
		jLogs = append(jLogs, jLog)
	}
	return jLogs
}

func EncEvent(log types.EventData) JsonLog {
	jLog := JsonLog{
		TransactionHash: EncBytes(log.TxHash),
		BlockNumber:     EncUint(log.BlockHeight),
		Address:         EncAddress(log.Caller),
		Data:            EncBytes(log.EncodedBody),
		//TransactionIndex: EncUint(log.TransactionIndex),
	}
	for _, topic := range log.Topics {
		jLog.Topics = append(jLog.Topics, Data(topic))
	}
	return jLog
}

func EncLogs(logs []*types.EthFilterLog) []JsonLog {
	var jLogs []JsonLog
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
		jLog.Topics = append(jLog.Topics, EncBytes(topic))
	}
	return jLog
}

func EncInt(value int64) Quantity {
	return Quantity("0x" + strconv.FormatInt(value, 16))
}

func EncUint(value uint64) Quantity {
	return Quantity("0x" + strconv.FormatUint(value, 16))
}

func EncBytes(value []byte) Data {
	bytes := Data("0x" + hex.EncodeToString(value))
	if bytes == "0x" {
		bytes = "0x0"
	}
	return bytes
}

func EncBytesArray(list [][]byte) []Data {
	dataArray := []Data{}
	for _, hash := range list {
		dataArray = append(dataArray, EncBytes(hash))
	}
	return dataArray
}

func EncAddress(value *ltypes.Address) Data {
	return EncBytes([]byte(value.Local))
}

func DecLogFilter(chianId string, filter JsonFilter) (resp utils.EthFilter, err error) {
	addresses := []loom.LocalAddress{}
	for _, data := range filter.Address {
		address, err := DecDataToBytes(data)
		if err != nil {
			return resp, errors.Wrap(err, "unwrap filter address")
		}
		addresses = append(addresses, address)
	}

	var topicsFilter [][]string
	for _, topicList := range filter.Topics {
		var topics []string
		for _, data := range topicList {
			topic, err := DecDataToBytes(data)
			if err != nil {
				return resp, errors.Wrap(err, "filter topics")
			}
			topics = append(topics, string(topic))
		}
		topicsFilter = append(topicsFilter, topics)
	}

	return utils.EthFilter{
		FromBlock: string(filter.FromBlock),
		ToBlock:   string(filter.ToBlock),
		EthBlockFilter: utils.EthBlockFilter{
			Addresses: addresses,
			Topics:    topicsFilter,
		},
	}, nil
}

func DecQuantityToInt(value Quantity) (int64, error) {
	if len(value) <= 2 || value[0:2] != "0x" {
		return 0, errors.Errorf("Invalid quantity format: %v", value)
	}
	return strconv.ParseInt(string(value), 0, 64)
}

func DecQuantityToUint(value Quantity) (uint64, error) {
	if len(value) <= 2 || value[0:2] != "0x" {
		return 0, errors.Errorf("Invalid quantity format: %v", value)
	}
	return strconv.ParseUint(string(value), 0, 64)
}

func DecDataToBytes(value Data) ([]byte, error) {
	if len(value) <= 2 || value[0:2] != "0x" {
		return []byte{}, errors.Errorf("Invalid data format: %v", value)
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
