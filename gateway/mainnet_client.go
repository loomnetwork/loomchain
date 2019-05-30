// +build evm

package gateway

import (
	"bytes"
	"context"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/pkg/errors"
)

// MainnetClient defines typed wrappers for the Ethereum RPC API.
type MainnetClient struct {
	*ethclient.Client
	rpcClient *rpc.Client
	erc20ABI  abi.ABI
}

type ERC20DepositInfo struct {
	ERC20Contract common.Address
	From          common.Address
	To            common.Address
	Amount        *big.Int
	TxIndex       uint
	BlockNumber   uint64
}

// ConnectToMainnet connects a client to the given URL.
func ConnectToMainnet(url string, erc20ABI abi.ABI) (*MainnetClient, error) {
	rpcClient, err := rpc.DialContext(context.Background(), url)
	if err != nil {
		return nil, err
	}
	return &MainnetClient{ethclient.NewClient(rpcClient), rpcClient, erc20ABI}, nil
}

type MainnetContractCreationTx struct {
	CreatorAddress  common.Address
	ContractAddress common.Address
}

func (c *MainnetClient) GetERC20DepositByTxHash(ctx context.Context, gatewayAddress common.Address, txHash common.Hash) ([]*ERC20DepositInfo, error) {
	r := &struct {
		Logs []types.Log `json:"logs"`
	}{}

	err := c.rpcClient.CallContext(ctx, &r, "eth_getTransactionReceipt", txHash)
	if err != nil {
		return nil, err
	} else if r == nil {
		return nil, ethereum.NotFound
	}

	transferEvent, ok := c.erc20ABI.Events["Transfer"]
	if !ok {
		return nil, errors.New("incompatible erc20 abi, no transfer event exists")
	}
	transferEventHash := transferEvent.Id()

	erc20Deposits := make([]*ERC20DepositInfo, 0, len(r.Logs))
	for _, log := range r.Logs {
		if !bytes.Equal(transferEventHash.Bytes(), log.Topics[0].Bytes()) {
			continue
		}

		valueHexStr := common.Bytes2Hex(log.Data)
		result, err := evmcompat.SolidityUnpackString(valueHexStr, []string{"uint256"})
		if err != nil {
			return nil, err
		}
		value := result[0].(*big.Int)

		result, err = evmcompat.SolidityUnpackString(common.Bytes2Hex(log.Topics[1].Bytes()), []string{"address"})
		if err != nil {
			return nil, err
		}
		from := common.HexToAddress(result[0].(string))

		result, err = evmcompat.SolidityUnpackString(common.Bytes2Hex(log.Topics[2].Bytes()), []string{"address"})
		if err != nil {
			return nil, err
		}
		to := common.HexToAddress(result[0].(string))

		// We are going to capture deposits made only to our gateway contract
		if gatewayAddress.Hex() != to.Hex() {
			continue
		}

		erc20Deposits = append(erc20Deposits, &ERC20DepositInfo{
			From:          from,
			To:            to,
			Amount:        value,
			ERC20Contract: log.Address,
			TxIndex:       log.TxIndex,
			BlockNumber:   log.BlockNumber,
		})
	}

	return erc20Deposits, nil
}

// ContractCreationTxByHash attempt to retrieve the address of the contract, and its creator from
// the tx matching the given tx hash.
// NOTE: This probably only works for contracts that are created by an owned account, contracts that
//       are created by other contracts don't have a distinct tx that can be looked up.
func (c *MainnetClient) ContractCreationTxByHash(ctx context.Context, txHash common.Hash) (*MainnetContractCreationTx, error) {
	tx := &struct {
		// Ganache doesn't format the zero address the way go-ethereum expects, so have to unmarshal
		// it manually
		To        string         `json:"to"`
		From      common.Address `json:"from"`
		BlockHash common.Hash    `json:"blockHash"`
	}{}

	err := c.rpcClient.CallContext(ctx, &tx, "eth_getTransactionByHash", txHash)
	if err != nil {
		return nil, err
	} else if tx == nil {
		return nil, ethereum.NotFound
	}

	// A contract creation tx must be sent to 0x0
	if tx.To != "0x0" && tx.To != "0x" && len(tx.To) > 0 {
		return nil, ethereum.NotFound
	}

	// TODO: check for empty block hash - means tx is still pending and we should error out

	r := &struct {
		ContractAddress common.Address `json:"contractAddress"`
	}{}

	err = c.rpcClient.CallContext(ctx, &r, "eth_getTransactionReceipt", txHash)
	if err != nil {
		return nil, err
	} else if r == nil {
		return nil, ethereum.NotFound
	}

	return &MainnetContractCreationTx{
		CreatorAddress:  tx.From,
		ContractAddress: r.ContractAddress,
	}, nil
}
