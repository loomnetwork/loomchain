package ethcoin

import (
	"github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	ectypes "github.com/loomnetwork/go-loom/builtin/types/ethcoin"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/pkg/errors"
)

type (
	InitRequest          = ectypes.ETHCoinInitRequest
	MintToGatewayRequest = ectypes.ETHCoinMintToGatewayRequest
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
	// Store Keys
	economyKey         = []byte("economy")
	accountKeyPrefix   = []byte("account")
	allowanceKeyPrefix = []byte("allowance")
)

func accountKey(addr loom.Address) []byte {
	return util.PrefixKey(accountKeyPrefix, addr.Bytes())
}

func allowanceKey(owner, spender loom.Address) []byte {
	return util.PrefixKey(allowanceKeyPrefix, owner.Bytes(), spender.Bytes())
}

// ETHCoin is an ERC20-like contract that's used to store & transfer ETH on the DAppChain.
// Its initial total supply is zero, and the Gateway contract is the only entity that's allowed
// to mint new ETH.
type ETHCoin struct {
}

func (c *ETHCoin) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "ethcoin",
		Version: "1.0.0",
	}, nil
}

func (c *ETHCoin) Init(ctx contract.Context, req *InitRequest) error {
	div := loom.NewBigUIntFromInt(10)
	div.Exp(div, loom.NewBigUIntFromInt(18), nil)

	supply := loom.NewBigUIntFromInt(0)
	for _, initAcct := range req.Accounts {
		owner := loom.UnmarshalAddressPB(initAcct.Owner)
		balance := loom.NewBigUIntFromInt(int64(initAcct.Balance))
		balance.Mul(balance, div)

		acct := &Account{
			Owner: owner.MarshalPB(),
			Balance: &types.BigUInt{
				Value: *balance,
			},
		}
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

// MintToGateway adds ETH to the Gateway contract balance, and updates the total supply.
func (c *ETHCoin) MintToGateway(ctx contract.Context, req *MintToGatewayRequest) error {
	gatewayAddr, err := ctx.Resolve("gateway")
	if err != nil {
		return errors.Wrap(err, "failed to mint ETH")
	}

	if ctx.Message().Sender.Compare(gatewayAddr) != 0 {
		return errors.New("not authorized to mint ETH")
	}

	return Mint(ctx, gatewayAddr, &req.Amount.Value)
}

// NOTE: This is only exported for tests, don't call it from anywhere else!
func Mint(ctx contract.Context, to loom.Address, amount *loom.BigUInt) error {
	account, err := loadAccount(ctx, to)
	if err != nil {
		return err
	}

	econ := &Economy{
		TotalSupply: &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
	}
	err = ctx.Get(economyKey, econ)
	if err != nil && err != contract.ErrNotFound {
		return err
	}

	bal := account.Balance.Value
	supply := econ.TotalSupply.Value

	bal.Add(&bal, amount)
	supply.Add(&supply, amount)

	account.Balance.Value = bal
	econ.TotalSupply.Value = supply

	if err := saveAccount(ctx, account); err != nil {
		return err
	}
	return ctx.Set(economyKey, econ)
}

// ERC20 methods

func (c *ETHCoin) TotalSupply(ctx contract.StaticContext, req *TotalSupplyRequest) (*TotalSupplyResponse, error) {
	var econ Economy
	err := ctx.Get(economyKey, &econ)
	if err != nil {
		return nil, err
	}
	return &TotalSupplyResponse{
		TotalSupply: econ.TotalSupply,
	}, nil
}

func (c *ETHCoin) BalanceOf(ctx contract.StaticContext, req *BalanceOfRequest) (*BalanceOfResponse, error) {
	owner := loom.UnmarshalAddressPB(req.Owner)
	balance, err := BalanceOf(ctx, owner)
	if err != nil {
		return nil, err
	}
	return &BalanceOfResponse{
		Balance: &types.BigUInt{Value: *balance},
	}, nil
}

func BalanceOf(ctx contract.StaticContext, owner loom.Address) (*loom.BigUInt, error) {
	acct, err := loadAccount(ctx, owner)
	if err != nil {
		return nil, err
	}
	return &acct.Balance.Value, nil
}

func AddBalance(ctx contract.Context, addr loom.Address, amount *loom.BigUInt) error {
	account, err := loadAccount(ctx, addr)
	if err != nil {
		return err
	}

	balance := account.Balance.Value
	balance.Add(&balance, amount)
	account.Balance.Value = balance
	err = saveAccount(ctx, account)
	if err != nil {
		return err
	}
	return nil
}

func SubBalance(ctx contract.Context, addr loom.Address, amount *loom.BigUInt) error {
	account, err := loadAccount(ctx, addr)
	if err != nil {
		return err
	}

	balance := account.Balance.Value
	balance.Sub(&balance, amount)
	account.Balance.Value = balance
	err = saveAccount(ctx, account)
	if err != nil {
		return err
	}
	return nil
}

func SetBalance(ctx contract.Context, addr loom.Address, amount *loom.BigUInt) error {
	account, err := loadAccount(ctx, addr)
	if err != nil {
		return err
	}
	account.Balance.Value = *loom.NewBigUInt(amount.Int)
	err = saveAccount(ctx, account)
	if err != nil {
		return err
	}
	return nil
}

func (c *ETHCoin) Transfer(ctx contract.Context, req *TransferRequest) error {
	from := ctx.Message().Sender
	to := loom.UnmarshalAddressPB(req.To)
	amount := req.Amount.Value

	return Transfer(ctx, from, to, &amount)
}

func Transfer(ctx contract.Context, from, to loom.Address, amount *loom.BigUInt) error {
	fromAccount, err := loadAccount(ctx, from)
	if err != nil {
		return err
	}

	toAccount, err := loadAccount(ctx, to)
	if err != nil {
		return err
	}

	fromBalance := fromAccount.Balance.Value
	toBalance := toAccount.Balance.Value

	if fromBalance.Cmp(amount) < 0 {
		return errors.New("sender balance is too low")
	}

	fromBalance.Sub(&fromBalance, amount)
	toBalance.Add(&toBalance, amount)

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

func (c *ETHCoin) Approve(ctx contract.Context, req *ApproveRequest) error {
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

func (c *ETHCoin) Allowance(ctx contract.StaticContext, req *AllowanceRequest) (*AllowanceResponse, error) {
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

func (c *ETHCoin) TransferFrom(ctx contract.Context, req *TransferFromRequest) error {
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

func loadAccount(ctx contract.StaticContext, owner loom.Address) (*Account, error) {
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

func loadAllowance(ctx contract.StaticContext, owner, spender loom.Address) (*Allowance, error) {
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

var Contract plugin.Contract = contract.MakePluginContract(&ETHCoin{})
