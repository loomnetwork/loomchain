package gateway

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	loom "github.com/loomnetwork/go-loom"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

type ERC20Context struct {
	contract.Context
	TokenAddress loom.Address
}

func NewERC20Context(ctx contract.Context, tokenAddr loom.Address) *ERC20Context {
	return &ERC20Context{
		Context:      ctx,
		TokenAddress: tokenAddr,
	}
}

func (c *ERC20Context) transferFrom(from, to loom.Address, amount *big.Int) error {
	fromAddr := common.BytesToAddress(from.Local)
	toAddr := common.BytesToAddress(to.Local)
	_, err := callEVM(c, c.TokenAddress, "transferFrom", fromAddr, toAddr, amount)
	return err
}
