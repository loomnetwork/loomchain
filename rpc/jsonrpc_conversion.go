package rpc

import (
	"encoding/hex"
	"strconv"

	"github.com/loomnetwork/go-loom/plugin/types"
	ltypes "github.com/loomnetwork/go-loom/types"
)

type quantity string
type data string

type JsonLog struct {
	Removed          bool     `json:"removed,omitempty"`
	LogIndex         quantity `json:"logIndex,omitempty"`
	TransactionIndex quantity `json:"transactionIndex,omitempty"`
	TransactionHash  data     `json:"transactionHash,omitempty"`
	BlockHash        data     `json:"blockHash,omitempty"`
	BlockNumber      quantity `json:"blockNumber,omitempty"`
	Address          data     `json:"address,omitempty"`
	Data             data     `json:"data,omitempty"`
	Topics           []data   `json:"topics,omitempty"`
}

type JsonTxReceipt struct {
	TransactionIndex  quantity  `json:"transactionIndex,omitempty"`
	BlockHash         data      `json:"blockHash,omitempty"`
	BlockNumber       quantity  `json:"blockumber,omitempty"`
	CumulativeGasUsed quantity  `json:"cumulativeGasUsed,omitempty"`
	GasUsed           quantity  `json:"gasUsed,omitempty"`
	ContractAddress   data      `json:"contractAddress,omitempty"`
	Logs              []JsonLog `json:"logs,omitempty"`
	LogsBloom         data      `json:"logsBloom,omitempty"`
	Status            quantity  `json:"status,omitempty"`
	TxHash            data      `json:"txHash,omitempty"`
	CallerAddress     data      `json:"callerAddress,omitempty"`
}

func encTxReceipt(receipt types.EvmTxReceipt) JsonTxReceipt {
	return JsonTxReceipt{
		TransactionIndex:  encInt(int64(receipt.TransactionIndex)),
		BlockHash:         encBytes(receipt.BlockHash),
		BlockNumber:       encInt(receipt.BlockNumber),
		CumulativeGasUsed: encInt(int64(receipt.CumulativeGasUsed)),
		GasUsed:           encInt(int64(receipt.GasUsed)),
		ContractAddress:   encBytes(receipt.ContractAddress),
		Logs:              encLogs(receipt.Logs),
		LogsBloom:         encBytes(receipt.LogsBloom),
		Status:            encInt(int64(receipt.Status)),
		TxHash:            encBytes(receipt.TxHash),
		CallerAddress:     encAddress(receipt.CallerAddress),
	}
}

func encLogs(logs []*types.EventData) []JsonLog {
	var jLogs []JsonLog
	for i, log := range logs {
		jLog := encLog(*log)
		jLog.LogIndex = encInt(int64(i))
		jLogs = append(jLogs, jLog)
	}
	return jLogs
}

func encLog(log types.EventData) JsonLog {
	jLog := JsonLog{
		TransactionHash: encBytes(log.TxHash),
		BlockNumber:     encUint(log.BlockHeight),
		Address:         encAddress(log.Caller),
		Data:            encBytes(log.EncodedBody),
	}
	for _, topic := range log.Topics {
		jLog.Topics = append(jLog.Topics, data(topic))
	}
	return jLog
}

func encInt(value int64) quantity {
	return quantity("0x" + strconv.FormatInt(value, 16))
}

func encUint(value uint64) quantity {
	return quantity("0x" + strconv.FormatUint(value, 16))
}

func encBytes(value []byte) data {
	return data("0x" + hex.EncodeToString(value))
}

func encAddress(value *ltypes.Address) data {
	return encBytes([]byte(value.Local))
}
