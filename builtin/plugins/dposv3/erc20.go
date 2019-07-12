package dposv3

import (
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/builtin/types/coin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
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

type ERC20 struct {
	Context         contract.Context
	ContractAddress loom.Address
}

func (c *ERC20) Transfer(to loom.Address, amount *loom.BigUInt) error {
	req := &coin.TransferRequest{
		To: to.MarshalPB(),
		Amount: &types.BigUInt{
			Value: *amount,
		},
	}

	err := contract.CallMethod(c.Context, c.ContractAddress, "Transfer", req, nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *ERC20) TransferFrom(from, to loom.Address, amount *loom.BigUInt) error {
	req := &coin.TransferFromRequest{
		From: from.MarshalPB(),
		To:   to.MarshalPB(),
		Amount: &types.BigUInt{
			Value: *amount,
		},
	}

	err := contract.CallMethod(c.Context, c.ContractAddress, "TransferFrom", req, nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *ERC20) Approve(spender loom.Address, amount *loom.BigUInt) error {
	req := &coin.ApproveRequest{
		Spender: spender.MarshalPB(),
		Amount: &types.BigUInt{
			Value: *amount,
		},
	}

	err := contract.CallMethod(c.Context, c.ContractAddress, "Approve", req, nil)
	if err != nil {
		return err
	}

	return nil
}

// MintToDPOS adds loom coins to the loom coin DPOS contract balance, and updates the total supply.
func (c *ERC20) MintToDPOS(amount *loom.BigUInt) error {
	req := &coin.MintToDPOSRequest{
		Amount: &types.BigUInt{
			Value: *amount,
		},
	}
	err := contract.CallMethod(c.Context, c.ContractAddress, "MintToDPOS", req, nil)
	if err != nil {
		return err
	}

	return nil
}

