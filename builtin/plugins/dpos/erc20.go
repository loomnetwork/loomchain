package dpos

import (
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/builtin/types/coin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

type ERC20Static struct {
	StaticContext   contract.StaticContext
	ContractAddress loom.Address
}

func (c *ERC20Static) TotalSupply() (*loom.BigUInt, error) {
	req := &coin.TotalSupplyRequest{}
	var resp coin.TotalSupplyResponse

	err := contract.StaticCallMethod(c.StaticContext, c.ContractAddress, "TotalSupply", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.TotalSupply.Value, nil
}

func (c *ERC20Static) BalanceOf(addr loom.Address) (*loom.BigUInt, error) {
	req := &coin.BalanceOfRequest{
		Owner: addr.MarshalPB(),
	}
	var resp coin.BalanceOfResponse
	err := contract.StaticCallMethod(c.StaticContext, c.ContractAddress, "BalanceOf", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Balance.Value, nil
}
