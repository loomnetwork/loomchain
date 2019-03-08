// +build evm

package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/loomnetwork/go-loom"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/pkg/errors"
)

// The Mainnet Event Simulator collects events from a set of Ethereum txs sent to the Ethereum
// Gateway, and resets the block number on the events. Currently only supports simulation of LOOM
// deposits.
//
// Useful for writing tests using a known set of events from a deployed Ethereum Gateway.
type mainnetEventSimulator struct {
	sourceTxs       []common.Hash
	simulatedEvents []*mainnetEventInfo
	logger          *loom.Logger
	oracle          *Oracle
}

// Creates a new instance of the simulator, the sourceTxPath should be the path to a JSON file
// containing an array of hex-encoded strings reprsenting Ethereum tx hashes.
func newMainnetEventSimulator(oracle *Oracle, sourceTxsPath string) (*mainnetEventSimulator, error) {
	var sourceTxs []common.Hash
	content, err := ioutil.ReadFile(sourceTxsPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read %s", sourceTxsPath)
	}
	if err := json.Unmarshal(content, &sourceTxs); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal %s", sourceTxsPath)
	}

	return &mainnetEventSimulator{
		sourceTxs: sourceTxs,
		logger:    oracle.logger,
		oracle:    oracle,
	}, nil
}

func (mes *mainnetEventSimulator) simulateEvents(startBlock, endBlock uint64) ([]*MainnetEvent, error) {
	eventInfos, err := mes.fetchEventsFromSourceTxs()
	if err != nil {
		return nil, err
	}

	events := make([]*MainnetEvent, len(eventInfos))
	for i, eventInfo := range eventInfos {
		var eventName string
		var tokenKind string

		events[i] = eventInfo.Event
		// Reset block number so all events end up in the same block, they're already sorted by
		// block number & tx index.
		events[i].EthBlock = startBlock

		switch payload := events[i].Payload.(type) {
		case *tgtypes.TransferGatewayMainnetEvent_Deposit:
			eventName = "deposit"
			tokenKind = payload.Deposit.TokenKind.String()
		case *tgtypes.TransferGatewayMainnetEvent_Withdrawal:
			eventName = "withdrawal"
			tokenKind = payload.Withdrawal.TokenKind.String()
		}

		mes.logger.Info("Simulated Ethereum Gateway event",
			"txHash", eventInfo.TxHash.String(),
			"srcBlock", eventInfo.BlockNum,
			"destBlock", startBlock,
			"event", eventName,
			"token", tokenKind,
		)
	}

	if len(events) > 0 {
		mes.logger.Info("Simulated Ethereum events from source txs",
			"startBlock", startBlock,
			"endBlock", endBlock,
			"numEvents", len(events),
		)
	}
	return events, nil
}

func (mes *mainnetEventSimulator) fetchEventsFromSourceTxs() ([]*mainnetEventInfo, error) {
	// Events are only fetched once
	if mes.simulatedEvents != nil {
		return mes.simulatedEvents, nil
	}

	// Find all the blocks containing the source txs
	blocks := map[uint64]bool{}
	txHashes := map[common.Hash]bool{}
	for _, txHash := range mes.sourceTxs {
		receipt, err := mes.oracle.ethClient.TransactionReceipt(context.TODO(), txHash)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to fetch tx for hash %s", txHash.String())
		}
		for _, eventLog := range receipt.Logs {
			blocks[eventLog.BlockNumber] = true
		}
		txHashes[txHash] = true
	}

	// Fetch all the events from the relevant blocks
	var loomcoinDeposits []*mainnetEventInfo
	if mes.oracle.isLoomCoinOracle {
		for h := range blocks {
			events, err := mes.oracle.fetchLoomCoinDeposits(&bind.FilterOpts{
				Start: h,
				End:   &h,
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to fetch LOOM deposits from block %d", h)
			}
			// Filter out any events that didn't originate from the source txs
			for i := range events {
				if txHashes[events[i].TxHash] {
					loomcoinDeposits = append(loomcoinDeposits, events[i])
				}
			}
		}
	}

	// Sort events by block number & tx index
	sortedEvents := loomcoinDeposits
	sortMainnetEvents(sortedEvents)

	// There should only be one Gateway event per source tx
	if len(sortedEvents) != len(mes.sourceTxs) {
		return nil, fmt.Errorf(
			"number of Mainnet events (%d) doesn't match number of source txs (%d)",
			len(mes.sourceTxs), len(sortedEvents),
		)
	}

	if len(sortedEvents) > 0 {
		mes.logger.Info(
			"Fetched Mainnet events from source txs",
			"numTxs", len(mes.sourceTxs),
			"loomDeposits", len(loomcoinDeposits),
		)
	}

	mes.sourceTxs = nil
	mes.simulatedEvents = sortedEvents
	return sortedEvents, nil
}
