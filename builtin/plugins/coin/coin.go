package coin

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/features"
	"github.com/pkg/errors"
)

const (
	TransferEventTopic = "coin:transfer"
	ApprovalEventTopic = "coin:approval"
)

type (
	InitRequest          = ctypes.InitRequest
	MintToGatewayRequest = ctypes.MintToGatewayRequest
	TotalSupplyRequest   = ctypes.TotalSupplyRequest
	TotalSupplyResponse  = ctypes.TotalSupplyResponse
	BalanceOfRequest     = ctypes.BalanceOfRequest
	BalanceOfResponse    = ctypes.BalanceOfResponse
	TransferRequest      = ctypes.TransferRequest
	TransferResponse     = ctypes.TransferResponse
	TransferEvent        = ctypes.TransferEvent
	ApproveRequest       = ctypes.ApproveRequest
	ApproveResponse      = ctypes.ApproveResponse
	ApprovalEvent        = ctypes.ApprovalEvent
	AllowanceRequest     = ctypes.AllowanceRequest
	AllowanceResponse    = ctypes.AllowanceResponse
	TransferFromRequest  = ctypes.TransferFromRequest
	TransferFromResponse = ctypes.TransferFromResponse
	Allowance            = ctypes.Allowance
	Account              = ctypes.Account
	InitialAccount       = ctypes.InitialAccount
	Economy              = ctypes.Economy
	BurnRequest          = ctypes.BurnRequest
	BalancesRequest      = ctypes.BalancesRequest
	BalancesResponse     = ctypes.BalancesResponse
)

var (
	ErrSenderBalanceTooLow = errors.New("sender balance is too low")
	ErrInvalidRequest      = errors.New("[Coin Contract] invalid request")
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

// MintToGateway adds LOOM to the Gateway contract balance and updates the total supply.
func (c *Coin) MintToGateway(ctx contract.Context, req *MintToGatewayRequest) error {
	if ctx.FeatureEnabled(features.CoinVersion1_2Feature, false) && req.Amount == nil {
		return ErrInvalidRequest
	}

	if err := validateMinter(ctx, ctx.Message().Sender); err != nil {
		return errors.Wrap(err, "failed to mint LOOM")
	}

	return mint(ctx, ctx.Message().Sender, &req.Amount.Value)
}

// Burn removes LOOM from the Gateway contract balance and updates the total supply.
func (c *Coin) Burn(ctx contract.Context, req *BurnRequest) error {
	if req.Owner == nil || req.Amount == nil {
		return errors.New("owner or amount is nil")
	}

	if err := validateMinter(ctx, ctx.Message().Sender); err != nil {
		return errors.Wrap(err, "failed to burn LOOM")
	}

	return burn(ctx, loom.UnmarshalAddressPB(req.Owner), &req.Amount.Value)
}

// Returns nil if the given address is allowed to mint & burn LOOM, and an error otherwise.
func validateMinter(ctx contract.StaticContext, minter loom.Address) error {
	gatewayAddr, err := ctx.Resolve("loomcoin-gateway")
	if err == nil && minter.Compare(gatewayAddr) == 0 {
		return nil
	}

	if ctx.FeatureEnabled(features.CoinVersion1_3Feature, false) {
		gatewayAddr, err = ctx.Resolve("binance-gateway")
		if err == nil && minter.Compare(gatewayAddr) == 0 {
			return nil
		}
	}

	return errors.New("not authorized")
}

func burn(ctx contract.Context, from loom.Address, amount *loom.BigUInt) error {
	account, err := loadAccount(ctx, from)
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

	// Being extra cautious won't hurt.
	if bal.Cmp(amount) < 0 || supply.Cmp(amount) < 0 {
		return fmt.Errorf("can't burn more coins than the available balance: %s", bal.String())
	}

	bal.Sub(&bal, amount)
	supply.Sub(&supply, amount)

	account.Balance.Value = bal
	econ.TotalSupply.Value = supply

	if err := saveAccount(ctx, account); err != nil {
		return err
	}

	// Burn events are transfer events with the empty address as `to`
	burnAddress := loom.RootAddress(ctx.Block().ChainID)
	if err := emitTransferEvent(ctx, from, burnAddress, amount); err != nil {
		return err
	}

	return ctx.Set(economyKey, econ)
}

func mint(ctx contract.Context, to loom.Address, amount *loom.BigUInt) error {
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

	// Minting events are transfer events with the empty address as `from`
	mintAddress := loom.RootAddress(ctx.Block().ChainID)
	if err := emitTransferEvent(ctx, mintAddress, to, amount); err != nil {
		return err
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
	if req.Owner == nil {
		return nil, ErrInvalidRequest
	}
	owner := loom.UnmarshalAddressPB(req.Owner)
	acct, err := loadAccount(ctx, owner)
	if err != nil {
		return nil, err
	}
	return &BalanceOfResponse{
		Balance: acct.Balance,
	}, nil
}

func (c *Coin) Balances(
	ctx contract.StaticContext,
	req *BalancesRequest,
) (*BalancesResponse, error) {
	accountRange := ctx.Range([]byte("account"))
	accounts := []*Account{}
	for _, m := range accountRange {
		var account Account
		if err := proto.Unmarshal(m.Value, &account); err != nil {
			return nil, errors.Wrapf(err, "unmarshal account %x", m.Key)
		}
		accounts = append(accounts, &account)
	}
	return &BalancesResponse{
		Accounts: accounts,
	}, nil
}

func (c *Coin) Transfer(ctx contract.Context, req *TransferRequest) error {
	if ctx.FeatureEnabled(features.CoinVersion1_2Feature, false) && (req.To == nil || req.Amount == nil) {
		return ErrInvalidRequest
	}
	if ctx.FeatureEnabled(features.CoinVersion1_1Feature, false) {
		return c.transfer(ctx, req)
	}
	return c.legacyTransfer(ctx, req)
}

func (c *Coin) transfer(ctx contract.Context, req *TransferRequest) error {
	from := ctx.Message().Sender

	fromAccount, err := loadAccount(ctx, from)
	if err != nil {
		return err
	}
	amount := req.Amount.Value
	fromBalance := fromAccount.Balance.Value

	if fromBalance.Cmp(&amount) < 0 {
		return ErrSenderBalanceTooLow
	}

	fromBalance.Sub(&fromBalance, &amount)
	fromAccount.Balance.Value = fromBalance

	err = saveAccount(ctx, fromAccount)
	if err != nil {
		return err
	}

	to := loom.UnmarshalAddressPB(req.To)
	toAccount, err := loadAccount(ctx, to)
	if err != nil {
		return err
	}

	toBalance := toAccount.Balance.Value
	toBalance.Add(&toBalance, &amount)
	toAccount.Balance.Value = toBalance

	err = saveAccount(ctx, toAccount)
	if err != nil {
		return err
	}

	return emitTransferEvent(ctx, from, to, &amount)
}

func (c *Coin) Approve(ctx contract.Context, req *ApproveRequest) error {
	if ctx.FeatureEnabled(features.CoinVersion1_2Feature, false) && (req.Spender == nil || req.Amount == nil) {
		return ErrInvalidRequest
	}

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

	// req.Amount may be nil at this point, so have to be careful when dereferencing it
	var amount *loom.BigUInt
	if req.Amount != nil {
		amount = &req.Amount.Value
	}
	return emitApprovalEvent(ctx, owner, spender, amount)
}

func (c *Coin) Allowance(
	ctx contract.StaticContext,
	req *AllowanceRequest,
) (*AllowanceResponse, error) {
	if req.Spender == nil || req.Owner == nil {
		return nil, ErrInvalidRequest
	}
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
	if ctx.FeatureEnabled(features.CoinVersion1_2Feature, false) {
		if req.Amount == nil || req.From == nil || req.To == nil {
			return ErrInvalidRequest
		}
	}
	if ctx.FeatureEnabled(features.CoinVersion1_1Feature, false) {
		return c.transferFrom(ctx, req)
	}

	return c.legacyTransferFrom(ctx, req)
}

func (c *Coin) transferFrom(ctx contract.Context, req *TransferFromRequest) error {
	from := loom.UnmarshalAddressPB(req.From)
	fromAccount, err := loadAccount(ctx, from)
	if err != nil {
		return err
	}
	spender := ctx.Message().Sender
	allow, err := loadAllowance(ctx, from, spender)
	if err != nil {
		return err
	}

	allowAmount := allow.Amount.Value
	amount := req.Amount.Value
	fromBalance := fromAccount.Balance.Value

	if allowAmount.Cmp(&amount) < 0 {
		return errors.New("amount is over spender's limit")
	}

	if fromBalance.Cmp(&amount) < 0 {
		return ErrSenderBalanceTooLow
	}

	fromBalance.Sub(&fromBalance, &amount)
	fromAccount.Balance.Value = fromBalance

	err = saveAccount(ctx, fromAccount)
	if err != nil {
		return err
	}
	to := loom.UnmarshalAddressPB(req.To)
	toAccount, err := loadAccount(ctx, to)
	if err != nil {
		return err
	}

	toBalance := toAccount.Balance.Value
	toBalance.Add(&toBalance, &amount)
	toAccount.Balance.Value = toBalance

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

	return emitTransferEvent(ctx, from, to, &amount)
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

// Legacy methods from v1.0.0

func (c *Coin) legacyTransfer(ctx contract.Context, req *TransferRequest) error {
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

	return emitTransferEvent(ctx, from, to, &amount)
}

func (c *Coin) legacyTransferFrom(ctx contract.Context, req *TransferFromRequest) error {
	from := loom.UnmarshalAddressPB(req.From)
	fromAccount, err := loadAccount(ctx, from)
	if err != nil {
		return err
	}
	to := loom.UnmarshalAddressPB(req.To)
	toAccount, err := loadAccount(ctx, to)
	if err != nil {
		return err
	}
	spender := ctx.Message().Sender
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

	return emitTransferEvent(ctx, from, to, &amount)
}

// Events

func emitTransferEvent(ctx contract.Context, from, spender loom.Address, amount *loom.BigUInt) error {
	var safeAmount *types.BigUInt
	if amount == nil {
		safeAmount = loom.BigZeroPB()
	} else {
		safeAmount = &types.BigUInt{Value: *amount}
	}
	marshalled, err := proto.Marshal(&TransferEvent{
		From:   from.MarshalPB(),
		To:     spender.MarshalPB(),
		Amount: safeAmount,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, TransferEventTopic)
	return nil
}

func emitApprovalEvent(ctx contract.Context, from, to loom.Address, amount *loom.BigUInt) error {
	var safeAmount *types.BigUInt
	if amount == nil {
		safeAmount = loom.BigZeroPB()
	} else {
		safeAmount = &types.BigUInt{Value: *amount}
	}
	marshalled, err := proto.Marshal(&ApprovalEvent{
		From:    from.MarshalPB(),
		Spender: to.MarshalPB(),
		Amount:  safeAmount,
	})
	if err != nil {
		return err
	}

	ctx.EmitTopics(marshalled, ApprovalEventTopic)
	return nil
}
