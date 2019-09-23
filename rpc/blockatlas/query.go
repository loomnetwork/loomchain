package blockatlas

import (
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

var (
	searchBlockSize = uint64(20)
)

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

	// We ignore the error here becuase if the block results can't be loaded for any reason
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

	var input []byte
	switch txTx.Id {
	case DeployId:
		{
			var deployTx vm.DeployTx
			if err := proto.Unmarshal(msg.Data, &deployTx); err != nil {
				return GetEmptyTxObject(), nil, err
			}
			fmt.Printf("DEPLOY-TX : %+v\n", deployTx)
			input = deployTx.Code
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
			// if deployTx.Value != nil {
			// 	txObj.Value = EncBigInt(*deployTx.Value.Value.Int)
			// }
			txObj.TransactionType = TransactionType[DeployId]
		}
	case CallId:
		{
			var callTx vm.CallTx
			if err := proto.Unmarshal(msg.Data, &callTx); err != nil {
				return GetEmptyTxObject(), nil, err
			}

			input = callTx.Input
			txObj.To = msg.To.Local.String()
			if callTx.VmType == vm.VMType_EVM && len(txResultData) > 0 {
				txObj.Hash = EncBytes(txResultData)
			}
			if callTx.Value != nil {
				txObj.Value = EncBigInt(*callTx.Value.Value.Int)
			}

			var req gplugin.Request
			if err := proto.Unmarshal(input, &req); err != nil {
				return GetEmptyTxObject(), nil, err
			}

			var methodcall gplugin.ContractMethodCall
			if err := proto.Unmarshal(req.Body, &methodcall); err != nil {
				return GetEmptyTxObject(), nil, err
			}

			txObj.ContractMethod = methodcall.GetMethod()
			switch methodcall.GetMethod() {
			case "Transfer":
				var transfer cointypes.TransferRequest
				if err := proto.Unmarshal(methodcall.Args, &transfer); err != nil {
					return GetEmptyTxObject(), nil, err
				}
			case "Delegate":
				var delegate dpos3types.DelegateRequest
				if err := proto.Unmarshal(methodcall.Args, &delegate); err != nil {
					return GetEmptyTxObject(), nil, err
				}
				txObj.Value = DelegateValue{
					ValidatorAddress: EncAddress(delegate.GetValidatorAddress()),
					Amount:           EncUint(delegate.Amount.Value.Uint64()),
					LockTimeTier:     EncUint(delegate.LocktimeTier),
					Referrer:         Data(delegate.GetReferrer()),
				}
			case "Redelegate":
				var redelegate dpos3types.RedelegateRequest
				if err := proto.Unmarshal(methodcall.Args, &redelegate); err != nil {
					return GetEmptyTxObject(), nil, err
				}
				txObj.Value = ReDelegateValue{
					ValidatorAddress:       EncAddress(redelegate.GetValidatorAddress()),
					FormerValidatorAddress: EncAddress(redelegate.GetFormerValidatorAddress()),
					Index:                  EncUint(redelegate.Index),
					Amount:                 EncUint(redelegate.Amount.Value.Uint64()),
					NewLockTimeTier:        EncUint(redelegate.NewLocktimeTier),
					Referrer:               Data(redelegate.GetReferrer()),
				}
			default:
				fmt.Printf("Some others method %s\n", methodcall.GetMethod())
			}
			txObj.TransactionType = TransactionType[CallId]
		}
	case MigrationTx:
		txObj.To = msg.To.Local.String()
		txObj.TransactionType = TransactionType[MigrationTx]
	default:
		return GetEmptyTxObject(), nil, fmt.Errorf("unrecognised tx type %v", txTx.Id)
	}
	return txObj, contractAddress, nil
}
