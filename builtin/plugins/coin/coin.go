package coin

import (
	"errors"

	loom "github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
)

type (
	InitRequest          = ctypes.InitRequest
	TotalSupplyRequest   = ctypes.TotalSupplyRequest
	TotalSupplyResponse  = ctypes.TotalSupplyResponse
	BalanceOfRequest     = ctypes.BalanceOfRequest
	BalanceOfResponse    = ctypes.BalanceOfResponse
	TransferRequest      = ctypes.TransferRequest
	TransferResponse     = ctypes.TransferResponse
	ApproveRequest       = ctypes.ApproveRequest
	ApproveResponse      = ctypes.ApproveResponse
	AllowanceRequest     = ctypes.AllowanceRequest
	AllowanceResponse    = ctypes.AllowanceResponse
	TransferFromRequest  = ctypes.TransferFromRequest
	TransferFromResponse = ctypes.TransferFromResponse
	Allowance            = ctypes.Allowance
	Account              = ctypes.Account
	Economy              = ctypes.Economy
)

var (
	economyKey = []byte("economy")
	decimals   = 18
)

func accountKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte("account"), addr.Bytes())
}

func allowanceKey(owner, spender loom.Address) []byte {
	return util.PrefixKey([]byte("allowance"), owner.Bytes(), spender.Bytes())
}

type Coin struct {
}

func (c *Coin) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "coin",
		Version: "1.0.0",
	}, nil
}

func (c *Coin) Init(ctx contract.Context, req *InitRequest) error {
	supply := loom.NewBigUIntFromInt(0)
	for _, acct := range req.Accounts {
		owner := loom.UnmarshalAddressPB(acct.Owner)
		err := ctx.Set(accountKey(owner), acct)
		if err != nil {
			return err
		}

		supply.Add(supply, &acct.Balance.Value)
	}

	econ := &Economy{
		TotalSupply: &types.BigUInt{
			Value: *supply,
		},
	}
	return ctx.Set(economyKey, econ)
}

// ERC20 methods

func (c *Coin) TotalSupply(
	ctx contract.StaticContext,
	req *TotalSupplyRequest,
) (*TotalSupplyResponse, error) {
	var econ Economy
	err := ctx.Get(economyKey, &econ)
	if err != nil {
		return nil, err
	}
	return &TotalSupplyResponse{
		TotalSupply: econ.TotalSupply,
	}, nil
}

func (c *Coin) BalanceOf(
	ctx contract.StaticContext,
	req *BalanceOfRequest,
) (*BalanceOfResponse, error) {
	owner := loom.UnmarshalAddressPB(req.Owner)
	acct, err := loadAccount(ctx, owner)
	if err != nil {
		return nil, err
	}
	return &BalanceOfResponse{
		Balance: acct.Balance,
	}, nil
}

func (c *Coin) Transfer(ctx contract.Context, req *TransferRequest) error {
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

	amount := req.Amount.Value
	fromBalance := fromAccount.Balance.Value
	toBalance := toAccount.Balance.Value

	if fromBalance.Cmp(&amount) < 0 {
		return errors.New("sender balance is too low")
	}

	fromBalance.Sub(&fromBalance, &amount)
	toBalance.Add(&toBalance, &amount)

	fromAccount.Balance.Value = fromBalance
	toAccount.Balance.Value = toBalance
	err = saveAccount(ctx, fromAccount)
	if err != nil {
		return err
	}
	err = saveAccount(ctx, toAccount)
	if err != nil {
		return err
	}
	return nil
}

func (c *Coin) Approve(ctx contract.Context, req *ApproveRequest) error {
	owner := ctx.Message().Sender
	spender := loom.UnmarshalAddressPB(req.Spender)

	allow, err := loadAllowance(ctx, owner, spender)
	if err != nil {
		return err
	}

	allow.Amount = req.Amount
	err = saveAllowance(ctx, allow)
	if err != nil {
		return err
	}

	return nil
}

func (c *Coin) Allowance(
	ctx contract.Context,
	req *AllowanceRequest,
) (*AllowanceResponse, error) {
	owner := loom.UnmarshalAddressPB(req.Owner)
	spender := loom.UnmarshalAddressPB(req.Spender)

	allow, err := loadAllowance(ctx, owner, spender)
	if err != nil {
		return nil, err
	}

	return &AllowanceResponse{
		Amount: allow.Amount,
	}, nil
}

func (c *Coin) TransferFrom(ctx contract.Context, req *TransferFromRequest) error {
	spender := ctx.Message().Sender
	from := loom.UnmarshalAddressPB(req.From)
	to := loom.UnmarshalAddressPB(req.To)

	fromAccount, err := loadAccount(ctx, from)
	if err != nil {
		return err
	}

	toAccount, err := loadAccount(ctx, to)
	if err != nil {
		return err
	}

	allow, err := loadAllowance(ctx, from, spender)
	if err != nil {
		return err
	}

	allowAmount := allow.Amount.Value
	amount := req.Amount.Value
	fromBalance := fromAccount.Balance.Value
	toBalance := toAccount.Balance.Value

	if allowAmount.Cmp(&amount) < 0 {
		return errors.New("amount is over spender's limit")
	}

	if fromBalance.Cmp(&amount) < 0 {
		return errors.New("sender balance is too low")
	}

	fromBalance.Sub(&fromBalance, &amount)
	toBalance.Add(&toBalance, &amount)

	fromAccount.Balance.Value = fromBalance
	toAccount.Balance.Value = toBalance
	err = saveAccount(ctx, fromAccount)
	if err != nil {
		return err
	}
	err = saveAccount(ctx, toAccount)
	if err != nil {
		return err
	}

	allowAmount.Sub(&allowAmount, &amount)
	allow.Amount.Value = allowAmount
	err = saveAllowance(ctx, allow)
	if err != nil {
		return err
	}
	return nil
}

func loadAccount(
	ctx contract.StaticContext,
	owner loom.Address,
) (*Account, error) {
	acct := &Account{
		Owner: owner.MarshalPB(),
		Balance: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(0),
		},
	}
	err := ctx.Get(accountKey(owner), acct)
	if err != nil && err != contract.ErrNotFound {
		return nil, err
	}

	return acct, nil
}

func saveAccount(ctx contract.Context, acct *Account) error {
	owner := loom.UnmarshalAddressPB(acct.Owner)
	return ctx.Set(accountKey(owner), acct)
}

func loadAllowance(
	ctx contract.StaticContext,
	owner, spender loom.Address,
) (*Allowance, error) {
	allow := &Allowance{
		Owner:   owner.MarshalPB(),
		Spender: spender.MarshalPB(),
		Amount: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(0),
		},
	}
	err := ctx.Get(allowanceKey(owner, spender), allow)
	if err != nil && err != contract.ErrNotFound {
		return nil, err
	}

	return allow, nil
}

func saveAllowance(ctx contract.Context, allow *Allowance) error {
	owner := loom.UnmarshalAddressPB(allow.Owner)
	spender := loom.UnmarshalAddressPB(allow.Spender)
	return ctx.Set(allowanceKey(owner, spender), allow)
}

var Contract plugin.Contract = contract.MakePluginContract(&Coin{})
