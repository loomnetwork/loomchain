// +build evm

package gateway

import (
	"context"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// MainnetClient defines typed wrappers for the Ethereum RPC API.
type MainnetClient struct {
	*ethclient.Client
	rpcClient *rpc.Client
}

// ConnectToMainnet connects a client to the given URL.
func ConnectToMainnet(url string) (*MainnetClient, error) {
	rpcClient, err := rpc.DialContext(context.Background(), url)
	if err != nil {
		return nil, err
	}
	return &MainnetClient{ethclient.NewClient(rpcClient), rpcClient}, nil
}

type MainnetContractCreationTx struct {
	CreatorAddress  common.Address
	ContractAddress common.Address
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
