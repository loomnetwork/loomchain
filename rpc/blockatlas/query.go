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
	deployId    = uint32(1)
	callId      = uint32(2)
	migrationTx = uint32(3)
)

var (
	searchBlockSize = uint64(20)
)

func GetBlockByNumber(
	blockStore store.BlockStore,
	state loomchain.ReadOnlyState,
	height int64,
	evmAuxStore *evmaux.EvmAuxStore,
) (resp *JsonBlockObject, err error) {

	if height > state.Block().Height {
		return resp, errors.New("get block information for pending blocks not implemented yet")
	}

	var blockResult *ctypes.ResultBlock
	blockResult, err = blockStore.GetBlockByHeight(&height)
	if err != nil {
		return resp, errors.Wrapf(err, "GetBlockByNumber failed to get block %d", height)
	}

	blockInfo := &JsonBlockObject{
		Transactions:     nil,
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
		Value:            EncInt(0),
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
	case deployId:
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
			if deployTx.Value != nil {
				txObj.Value = EncBigInt(*deployTx.Value.Value.Int)
			}
		}
	case callId:
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
			fmt.Printf("CALL-TX 1: %+v\n", req)

			var methodcall gplugin.ContractMethodCall
			if err := proto.Unmarshal(req.Body, &methodcall); err != nil {
				return GetEmptyTxObject(), nil, err
			}
			fmt.Printf("CALL-TX 2: %+v\n", methodcall)

			txObj.ContractMethod = methodcall.GetMethod()
			switch methodcall.GetMethod() {
			case "Transfer":
				var transfer cointypes.TransferRequest
				if err := proto.Unmarshal(methodcall.Args, &transfer); err != nil {
					return GetEmptyTxObject(), nil, err
				}
				fmt.Printf("CALL-TX 3 transfer: %+v\n", transfer)
			case "Delegate":
				var delegate dpos3types.DelegateRequest
				if err := proto.Unmarshal(methodcall.Args, &delegate); err != nil {
					return GetEmptyTxObject(), nil, err
				}
				fmt.Printf("CALL-TX 3 delegate : %+v\n", delegate)
			case "Redelegate":
				var redelegate dpos3types.RedelegateRequest
				if err := proto.Unmarshal(methodcall.Args, &redelegate); err != nil {
					return GetEmptyTxObject(), nil, err
				}
				fmt.Printf("CALL-TX 3 redelegate : %+v\n", redelegate)
			default:
				fmt.Printf("Something else %s\n", methodcall.GetMethod())
			}
		}
	case migrationTx:
		txObj.To = msg.To.Local.String()
		input = msg.Data
	default:
		return GetEmptyTxObject(), nil, fmt.Errorf("unrecognised tx type %v", txTx.Id)
	}
	txObj.Input = EncBytes(input)

	return txObj, contractAddress, nil
}

func GetNumTxBlock(blockStore store.BlockStore, state loomchain.ReadOnlyState, height int64) (uint64, error) {
	// todo make information about pending block available.
	// Should be able to get transaction count from receipt object.
	if height > state.Block().Height {
		return 0, errors.New("get number of transactions for pending blocks, not implemented yet")
	}

	var blockResults *ctypes.ResultBlockResults
	blockResults, err := blockStore.GetBlockResults(&height)
	if err != nil {
		return 0, errors.Wrapf(err, "results for block %v", height)
	}
	return uint64(len(blockResults.Results.DeliverTx)), nil
}
