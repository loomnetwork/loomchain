package blockatlas

import (
	"encoding/json"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/loomnetwork/go-loom/auth"
	cointypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	dpos3types "github.com/loomnetwork/go-loom/builtin/types/dposv3"
	gplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
)

const (
	DeployId    = uint32(1)
	CallId      = uint32(2)
	MigrationTx = uint32(3)
)

var TransactionType = map[uint32]string{
	DeployId:    "DeployTx",
	CallId:      "ContractCall",
	MigrationTx: "MigrationTx",
}

func GetBlockByNumber(
	blockStore store.BlockStore,
	state loomchain.ReadOnlyState,
	height int64,
	evmAuxStore *evmaux.EvmAuxStore,
) (resp JsonBlockObject, err error) {

	if height > state.Block().Height {
		return resp, errors.New("get block information for pending blocks not implemented yet")
	}

	var blockResult *ctypes.ResultBlock
	blockResult, err = blockStore.GetBlockByHeight(&height)
	if err != nil {
		return resp, errors.Wrapf(err, "GetBlockByNumber failed to get block %d", height)
	}

	blockInfo := JsonBlockObject{
		Timestamp:        EncInt(int64(blockResult.Block.Header.Time.Unix())),
		GasLimit:         EncInt(0),
		GasUsed:          EncInt(0),
		Size:             EncInt(0),
		Nonce:            ZeroedData8Bytes,
		TransactionsRoot: ZeroedData32Bytes,
	}

	// These three fields are null for pending blocks.
	blockInfo.Hash = EncBytes(blockResult.BlockMeta.BlockID.Hash)
	blockInfo.Number = EncInt(height)

	var blockResults *ctypes.ResultBlockResults

	// We ignore the error here because if the block results can't be loaded for any reason
	// we'll try to load the data we need from tx_index.db instead.
	// TODO: Log the error returned by GetBlockResults.
	blockResults, _ = blockStore.GetBlockResults(&height)
	for index, tx := range blockResult.Block.Data.Txs {
		var blockResultBytes []byte
		if blockResults == nil {
			// Retrieve tx result from tx_index.db
			txResult, err := blockStore.GetTxResult(tx.Hash())
			if err != nil {
				return resp, errors.Wrapf(err, "cant find tx details, hash %X", tx.Hash())
			}
			blockResultBytes = txResult.TxResult.Data
		} else {
			blockResultBytes = blockResults.Results.DeliverTx[index].Data
		}

		txObj, _, err := GetTxObjectFromBlockResult(blockResult, blockResultBytes, int64(index))
		if err != nil {
			return resp, errors.Wrapf(err, "cant resolve tx, hash %X", tx.Hash())
		}
		blockInfo.Transactions = append(blockInfo.Transactions, txObj)
	}

	if len(blockInfo.Transactions) == 0 {
		blockInfo.Transactions = make([]JsonTxObject, 0)
	}
	return blockInfo, nil
}

func GetTxObjectFromBlockResult(
	blockResult *ctypes.ResultBlock, txResultData []byte, index int64,
) (JsonTxObject, *Data, error) {
	tx := blockResult.Block.Data.Txs[index]
	var contractAddress *Data
	txObj := JsonTxObject{
		BlockHash:        EncBytes(blockResult.BlockMeta.BlockID.Hash),
		BlockNumber:      EncInt(blockResult.Block.Header.Height),
		TransactionIndex: EncInt(int64(index)),
		GasPrice:         EncInt(0),
		Gas:              EncInt(0),
		Hash:             EncBytes(tx.Hash()),
	}

	var signedTx auth.SignedTx
	if err := proto.Unmarshal([]byte(tx), &signedTx); err != nil {
		return GetEmptyTxObject(), nil, err
	}

	var nonceTx auth.NonceTx
	if err := proto.Unmarshal(signedTx.Inner, &nonceTx); err != nil {
		return GetEmptyTxObject(), nil, err
	}
	txObj.Nonce = EncInt(int64(nonceTx.Sequence))

	var txTx loomchain.Transaction
	if err := proto.Unmarshal(nonceTx.Inner, &txTx); err != nil {
		return GetEmptyTxObject(), nil, err
	}

	var msg vm.MessageTx
	if err := proto.Unmarshal(txTx.Data, &msg); err != nil {
		return GetEmptyTxObject(), nil, err
	}
	txObj.From = msg.From.Local.String()

	switch txTx.Id {
	case DeployId:
		{
			var deployTx vm.DeployTx
			if err := proto.Unmarshal(msg.Data, &deployTx); err != nil {
				return GetEmptyTxObject(), nil, err
			}
			if deployTx.VmType == vm.VMType_EVM {
				var resp vm.DeployResponse
				if err := proto.Unmarshal(txResultData, &resp); err != nil {
					return GetEmptyTxObject(), nil, err
				}
				var respData vm.DeployResponseData
				if err := proto.Unmarshal(resp.Output, &respData); err != nil {
					return GetEmptyTxObject(), nil, err
				}
				contractAddress = EncPtrAddress(resp.Contract)
				if len(respData.TxHash) > 0 {
					txObj.Hash = EncBytes(respData.TxHash)
				}
			}
			if deployTx.Value != nil {
				txObj.Value = deployTx.Value.Value.Bytes()
			}
			txObj.TransactionType = TransactionType[DeployId]
		}
	case CallId:
		{
			var callTx vm.CallTx
			if err := proto.Unmarshal(msg.Data, &callTx); err != nil {
				return GetEmptyTxObject(), nil, err
			}

			txObj.To = msg.To.Local.String()
			if callTx.VmType == vm.VMType_EVM && len(txResultData) > 0 {
				txObj.Hash = EncBytes(txResultData)
			}

			var req gplugin.Request
			if err := proto.Unmarshal(callTx.Input, &req); err != nil {
				return GetEmptyTxObject(), nil, err
			}

			var methodcall gplugin.ContractMethodCall
			if err := proto.Unmarshal(req.Body, &methodcall); err != nil {
				return GetEmptyTxObject(), nil, err
			}

			txObj.ContractMethod = methodcall.GetMethod()
			var val []byte
			var err error
			switch methodcall.GetMethod() {
			case "Transfer":
				var transfer cointypes.TransferRequest
				if err := proto.Unmarshal(methodcall.Args, &transfer); err != nil {
					return GetEmptyTxObject(), nil, err
				}
				var toAddr, amount string
				if transfer.To != nil {
					toAddr = transfer.To.Local.String()
				}
				if transfer.Amount != nil {
					amount = transfer.Amount.Value.String()
				}
				val, err = json.Marshal(TransferValue{
					To:     toAddr,
					Amount: amount,
				})
				if err != nil {
					return GetEmptyTxObject(), nil, err
				}
			case "Approve":
				var approve cointypes.ApproveRequest
				if err := proto.Unmarshal(methodcall.Args, &approve); err != nil {
					return GetEmptyTxObject(), nil, err
				}
				var spender, amount string
				if approve.Spender != nil {
					spender = approve.Spender.Local.String()
				}
				if approve.Amount != nil {
					amount = approve.Amount.Value.String()
				}
				val, err = json.Marshal(ApproveValue{
					Spender: spender,
					Amount:  amount,
				})
				if err != nil {
					return GetEmptyTxObject(), nil, err
				}
			case "Delegate":
				var delegate dpos3types.DelegateRequest
				if err := proto.Unmarshal(methodcall.Args, &delegate); err != nil {
					return GetEmptyTxObject(), nil, err
				}
				var validatorAddr, amount string
				if delegate.ValidatorAddress != nil {
					validatorAddr = delegate.ValidatorAddress.Local.String()
				}
				if delegate.Amount != nil {
					amount = delegate.Amount.Value.Int.String()
				}
				val, err = json.Marshal(DelegateValue{
					ValidatorAddress: validatorAddr,
					Amount:           amount,
					LockTimeTier:     delegate.LocktimeTier,
					Referrer:         delegate.GetReferrer(),
				})
				if err != nil {
					return GetEmptyTxObject(), nil, err
				}
			case "Redelegate":
				var redelegate dpos3types.RedelegateRequest
				if err := proto.Unmarshal(methodcall.Args, &redelegate); err != nil {
					return GetEmptyTxObject(), nil, err
				}
				var validatorAddr, formerAddr, amount string
				if redelegate.ValidatorAddress != nil {
					validatorAddr = redelegate.ValidatorAddress.Local.String()
				}
				if redelegate.FormerValidatorAddress != nil {
					formerAddr = redelegate.FormerValidatorAddress.Local.String()
				}
				if redelegate.Amount == nil {
					amount = "amount" //Equal to Previous delegation amount
				}
				amount = redelegate.Amount.Value.String()
				val, err = json.Marshal(ReDelegateValue{
					ValidatorAddress:       validatorAddr,
					FormerValidatorAddress: formerAddr,
					Index:                  redelegate.Index,
					Amount:                 amount,
					NewLockTimeTier:        redelegate.NewLocktimeTier,
					Referrer:               redelegate.GetReferrer(),
				})
				if err != nil {
					return GetEmptyTxObject(), nil, err
				}
			case "Unbond":
				var unbond dpos3types.UnbondRequest
				if err := proto.Unmarshal(methodcall.Args, &unbond); err != nil {
					return GetEmptyTxObject(), nil, err
				}
				var validatorAddr, amount string
				if unbond.ValidatorAddress != nil {
					validatorAddr = unbond.ValidatorAddress.Local.String()
				}
				if unbond.Amount != nil {
					amount = unbond.Amount.Value.String()
				}
				val, err = json.Marshal(UnbondValue{
					ValidatorAddress: validatorAddr,
					Index:            unbond.Index,
					Amount:           amount,
				})
				if err != nil {
					return GetEmptyTxObject(), nil, err
				}
			}
			txObj.TransactionType = TransactionType[CallId]
			txObj.Value = val
		}
	case MigrationTx:
		txObj.To = msg.To.Local.String()
		txObj.TransactionType = TransactionType[MigrationTx]
	default:
		return GetEmptyTxObject(), nil, fmt.Errorf("unrecognised tx type %v", txTx.Id)
	}
	return txObj, contractAddress, nil
}
