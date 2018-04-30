package main

import (
	"errors"
	"math/big"

	loom "github.com/loomnetwork/go-loom"
	types "github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/util"
)

var economyKey = []byte("economy")
var decimals = 18

func UnmarshalBigUIntPB(b *types.BigUInt) *big.Int {
	return new(big.Int).SetBytes(b.Value)
}

func MarshalBigIntPB(b *big.Int) *types.BigUInt {
	return &types.BigUInt{
		Value: b.Bytes(),
	}
}

func accountKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte("account"), addr.Bytes())
}

type Coin struct {
}

func (c *Coin) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "coin",
		Version: "1.0.0",
	}, nil
}

func (c *Coin) Init(ctx contract.Context, req *types.InitRequest) error {
	for _, acct := range req.Accounts {
		owner := loom.UnmarshalAddressPB(acct.Owner)
		err := ctx.Set(accountKey(owner), acct)
		if err != nil {
			return err
		}
	}

	return nil
}

// ERC20 methods

func (c *Coin) TotalSupply(ctx contract.StaticContext, req *types.TotalSupplyRequest) (*types.TotalSupplyResponse, error) {
	var econ types.Economy
	err := ctx.Get(economyKey, &econ)
	if err != nil {
		return nil, err
	}
	return &types.TotalSupplyResponse{
		TotalSupply: econ.TotalSupply,
	}, nil
}

func (c *Coin) BalanceOf(ctx contract.StaticContext, req *types.BalanceOfRequest) (*types.BalanceOfResponse, error) {
	owner := loom.UnmarshalAddressPB(req.Owner)
	acct, err := loadAccount(ctx, owner)
	if err != nil {
		return nil, err
	}
	return &types.BalanceOfResponse{
		Balance: acct.Balance,
	}, nil
}

func (c *Coin) Transfer(ctx contract.Context, req *types.TransferRequest) error {
	from := ctx.Message().Sender
	to := loom.UnmarshalAddressPB(req.To)

	fromAccount, err := loadAccount(ctx, from)
	if err != nil {
		return err
	}

	toAccount, err := loadAccount(ctx, to)
	if err != nil {
		return err
	}

	amount := UnmarshalBigUIntPB(req.Amount)
	fromBalance := UnmarshalBigUIntPB(fromAccount.Balance)
	toBalance := UnmarshalBigUIntPB(toAccount.Balance)

	if fromBalance.Cmp(amount) < 0 {
		return errors.New("sender balance is too low")
	}

	fromBalance.Sub(fromBalance, amount)
	toBalance.Add(toBalance, amount)

	fromAccount.Balance = MarshalBigIntPB(fromBalance)
	toAccount.Balance = MarshalBigIntPB(toBalance)
	saveAccount(ctx, fromAccount)
	saveAccount(ctx, toAccount)
	return nil
}

func loadAccount(ctx contract.StaticContext, owner loom.Address) (*types.Account, error) {
	acct := &types.Account{Owner: owner.MarshalPB()}
	err := ctx.Get(accountKey(owner), acct)
	if err != nil && err != contract.ErrNotFound {
		return nil, err
	}

	return acct, nil
}

func saveAccount(ctx contract.Context, acct *types.Account) error {
	owner := loom.UnmarshalAddressPB(acct.Owner)
	return ctx.Set(accountKey(owner), acct)
}

var Contract plugin.Contract = contract.MakePluginContract(&Coin{})

func main() {
	plugin.Serve(Contract)
}
