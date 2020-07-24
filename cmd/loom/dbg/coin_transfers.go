package dbg

import (
	"encoding/json"
	"fmt"
	"math/big"
	"path"
	"time"

	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/blockchain"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/state/txindex/kv"
	tmtypes "github.com/tendermint/tendermint/types"
)

func findCoinTransfers(
	coinContracts []loom.Address, chaindataPath string, startHeight, endHeight int64, recipient *loom.Address,
) error {
	var err error
	blockStoreDB, err := dbm.NewGoLevelDBWithOpts(
		"blockstore", path.Join(chaindataPath, "data"),
		&opt.Options{
			ReadOnly: true,
		},
	)
	if err != nil {
		return errors.New("failed to load block store")
	}
	defer blockStoreDB.Close()

	txIndexDB, err := dbm.NewGoLevelDBWithOpts(
		"tx_index", path.Join(chaindataPath, "data"),
		&opt.Options{
			ReadOnly: true,
		},
	)
	if err != nil {
		return errors.New("failed to load tx index store")
	}
	defer txIndexDB.Close()

	blockStore := blockchain.NewBlockStore(blockStoreDB)
	if startHeight == 0 {
		startHeight = 1
	}
	if endHeight == 0 {
		endHeight = blockchain.LoadBlockStoreStateJSON(blockStoreDB).Height
	}

	txIndexer := kv.NewTxIndex(txIndexDB)

	fmt.Printf("Searching from block %v to block %v...\n", startHeight, endHeight)

	var recipientAddr string
	if recipient != nil {
		recipientAddr = recipient.String()
	}

	var totalTxCount, matchingTxCount int
	for h := startHeight; h <= endHeight; h++ {
		block := blockStore.LoadBlock(h)
		if block == nil {
			fmt.Printf("missing block at height %v\n", h)
			continue
		}
		if len(block.Data.Txs) > 0 {
			for ti, tx := range block.Data.Txs {
				txr, err := txIndexer.Get(tx.Hash())
				if err != nil {
					return err
				}
				if txr != nil { // means no result was found
					// Skip failed txs since they don't modify state, only look at calls to Go contracts
					if txr.Result.Code != abci.CodeTypeOK {
						continue
					}
					if txr.Result.Info != utils.CallPlugin {
						if txr.Result.Info != "" {
							continue
						} else {
							fmt.Printf("warning: unknown tx type at height %v, index %v\n", h, ti)
						}
					}
				} else {
					fmt.Printf("missing tx result at height %v, index %v\n", h, ti)
					continue
				}
				if info, err := decodeCoinTransferTx(tx, coinContracts); err == nil {
					if recipient != nil && info.Recipient != recipientAddr {
						continue
					}
					info.Time = block.Header.Time
					info.Height = block.Header.Height
					output, err := json.MarshalIndent(info, "", "  ")
					if err != nil {
						return err
					}
					fmt.Printf("%s,\n", string(output))
					matchingTxCount++
				}
			}
			totalTxCount += len(block.Data.Txs)
		}
	}

	fmt.Printf("Examined %v txs, found %v matches\n", totalTxCount, matchingTxCount)
	return nil
}

type coinTxInfo struct {
	Time      time.Time
	Height    int64
	TxHash    string
	Sender    string
	Recipient string
	Amount    string
	Contract  string
	Method    string
}

func decodeCoinTransferTx(tx tmtypes.Tx, coinContracts []loom.Address) (coinTxInfo, error) {
	var def coinTxInfo
	var signedTx auth.SignedTx
	if err := proto.Unmarshal(tx, &signedTx); err != nil {
		return def, errors.Wrap(err, "failed to unmarshal SignedTx")
	}

	var nonceTx auth.NonceTx
	if err := proto.Unmarshal(signedTx.Inner, &nonceTx); err != nil {
		return def, errors.Wrap(err, "failed to unmarshal NonceTx")
	}

	var loomTx types.Transaction
	if err := proto.Unmarshal(nonceTx.Inner, &loomTx); err != nil {
		return def, errors.Wrap(err, "failed to unmarshal Transaction")
	}

	var msgTx vm.MessageTx
	if err := proto.Unmarshal(loomTx.Data, &msgTx); err != nil {
		return def, errors.Wrap(err, "failed to unmarshal MessageTx")
	}

	if msgTx.To == nil {
		return def, errors.New("MessageTx.To not set")
	}

	if msgTx.From == nil {
		return def, errors.New("MessageTx.From not set")
	}

	var coinAddr loom.Address
	for i := range coinContracts {
		if coinContracts[i].Compare(loom.UnmarshalAddressPB(msgTx.To)) == 0 {
			coinAddr = coinContracts[i]

		}
	}
	if coinAddr.IsEmpty() {
		return def, errors.New("not a coin contract call")
	}

	if loomTx.Id != 2 {
		return def, errors.New("not a CallTx")
	}

	var callTx vm.CallTx
	if err := proto.Unmarshal(msgTx.Data, &callTx); err != nil {
		return def, errors.Wrap(err, "failed to unmarshal CallTx")
	}

	var req plugin.Request
	if err := proto.Unmarshal(callTx.Input, &req); err != nil {
		return def, errors.Wrap(err, "failed to unmarshal Request")
	}

	var methodCall plugin.ContractMethodCall
	if err := proto.Unmarshal(req.Body, &methodCall); err != nil {
		return def, errors.Wrap(err, "failed to unmarshal ContractMethodCall")
	}

	if methodCall.Method == "Transfer" {
		var args ctypes.TransferRequest
		if err := proto.Unmarshal(methodCall.Args, &args); err != nil {
			return def, errors.Wrap(err, "failed to unmarshal TransferRequest")
		}
		if args.To == nil {
			return def, errors.New("TransferRequest.To not set")
		}

		amount := new(big.Int)
		if args.Amount != nil {
			amount = args.Amount.Value.Int
		}

		return coinTxInfo{
			TxHash:    fmt.Sprintf("%X", tx.Hash()),
			Sender:    loom.UnmarshalAddressPB(msgTx.From).String(),
			Recipient: loom.UnmarshalAddressPB(args.To).String(),
			Amount:    amount.String(),
			Contract:  coinAddr.String(),
			Method:    "Transfer",
		}, nil
	} else if methodCall.Method == "TransferFrom" {
		var args ctypes.TransferFromRequest
		if err := proto.Unmarshal(methodCall.Args, &args); err != nil {
			return def, errors.Wrap(err, "failed to unmarshal TransferFromRequest")
		}
		if args.From == nil || args.To == nil {
			return def, errors.New("TransferFromRequest missing From or To")
		}

		amount := new(big.Int)
		if args.Amount != nil {
			amount = args.Amount.Value.Int
		}

		return coinTxInfo{
			TxHash:    fmt.Sprintf("%X", tx.Hash()),
			Sender:    loom.UnmarshalAddressPB(args.From).String(),
			Recipient: loom.UnmarshalAddressPB(args.To).String(),
			Amount:    amount.String(),
			Contract:  coinAddr.String(),
			Method:    "TransferFrom",
		}, nil
	}
	return def, errors.New("not a coin transfer")
}
