package coin

import (
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"

	errUtil "github.com/pkg/errors"
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
	Policy               = ctypes.Policy
	BurnRequest          = ctypes.BurnRequest
)

var (
	ErrSenderBalanceTooLow   = errors.New("sender balance is too low")
	InvalidBaseMintingAmount = errors.New("Base Minting Amount should be greater than zero")
)

var (
	economyKey = []byte("economy")
	policyKey  = []byte("policy")
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
	//Checks if Coin Policy Feature is enabled before loading Coin Monetary Supply Policy
	if ctx.FeatureEnabled(loomchain.CoinPolicyFeature, false) {
		if req.Policy == nil {
			return errors.New("Policy is not specified")
		}
		if req.Policy.DeflationFactorNumerator <= 0 {
			return errors.New("DeflationFactorNumerator should be greater than zero")
		}
		if req.Policy.DeflationFactorDenominator <= 0 {
			return errors.New("DeflationFactorDenominator should be greater than zero")
		}
		if req.BaseMintingAmount <= 0 {
			return InvalidBaseMintingAmount
		}
		if req.Policy.MintingAccount == nil {
			return errors.New("Invalid Minting Account Address")
		}
		addr := loom.UnmarshalAddressPB(req.Policy.MintingAccount)
		if addr.Compare(loom.RootAddress(addr.ChainID)) == 0 {
			return errors.New("Minting Account Address cannot be Root Address")
		}
		deflationFactorNumerator := req.Policy.DeflationFactorNumerator
		deflationFactorDenominator := req.Policy.DeflationFactorDenominator
		baseMintingAmount := loom.NewBigUIntFromInt(int64(req.BaseMintingAmount))
		baseMintingAmount.Mul(baseMintingAmount, div)
		policy := &Policy{
			DeflationFactorNumerator:   deflationFactorNumerator,
			DeflationFactorDenominator: deflationFactorDenominator,
			BaseMintingAmount: &types.BigUInt{
				Value: *baseMintingAmount,
			},
			MintingAccount: req.Policy.MintingAccount,
		}
		err := ctx.Set(policyKey, policy)
		if err != nil {
			return err
		}
	}
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

// MintToGateway adds loom coins to the loom coin Gateway contract balance, and updates the total supply.
func (c *Coin) MintToGateway(ctx contract.Context, req *MintToGatewayRequest) error {
	gatewayAddr, err := ctx.Resolve("loomcoin-gateway")
	if err != nil {
		return errUtil.Wrap(err, "failed to mint Loom coin")
	}

	if ctx.Message().Sender.Compare(gatewayAddr) != 0 {
		return errors.New("not authorized to mint Loom coin")
	}

	return mint(ctx, gatewayAddr, &req.Amount.Value)
}

func (c *Coin) Burn(ctx contract.Context, req *BurnRequest) error {
	if req.Owner == nil || req.Amount == nil {
		return errors.New("owner or amount is nil")
	}

	gatewayAddr, err := ctx.Resolve("loomcoin-gateway")
	if err != nil {
		return errUtil.Wrap(err, "failed to burn Loom coin")
	}

	if ctx.Message().Sender.Compare(gatewayAddr) != 0 {
		return errors.New("not authorized to burn Loom coin")
	}

	return burn(ctx, loom.UnmarshalAddressPB(req.Owner), &req.Amount.Value)
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

	// Being extra cautious wont hurt.
	if bal.Cmp(amount) < 0 || supply.Cmp(amount) < 0 {
		return fmt.Errorf("cant burn coins more than available balance: %s", bal.String())
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

//Mint : to be called by CoinDeflationManager Responsible to mint coins as per various parameter defined
func Mint(ctx contract.Context) error {
	var policy Policy
	err := ctx.Get(policyKey, &policy)
	if err != nil {
		return errUtil.Wrap(err, "Failed to Get PolicyInfo")
	}
	var depreciation *common.BigUInt
	policyNumerator := loom.NewBigUIntFromInt(int64(policy.DeflationFactorNumerator))
	policyDenominator := loom.NewBigUIntFromInt(int64(policy.DeflationFactorDenominator))
	blockHeight := loom.NewBigUIntFromInt(ctx.Block().Height)
	depreciation.Mul(policyNumerator, blockHeight)
	depreciation.Div(depreciation, policyDenominator)
	amount := policy.BaseMintingAmount.Value
	amount.Sub(&amount, depreciation)
	return mint(ctx, loom.UnmarshalAddressPB(policy.MintingAccount), &amount)
}

//ModifyMintParameter Method to modify deflation parameter, only callable by manager,
// when feature flag is going to be enabled
func ModifyMintParameter(ctx contract.Context, policy *ctypes.Policy) error {
	if policy.MintingAccount == nil || policy.BaseMintingAmount == nil {
		return errors.New("MintingAccount or BaseMintingAmount is not given")
	}
	err := ctx.Set(policyKey, policy)
	if err != nil {
		return err
	}
	return nil

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
	if ctx.FeatureEnabled(loomchain.CoinVersion1_1Feature, false) {
		return c.transfer(ctx, req)
	}

	return c.legacyTransfer(ctx, req)
}

func (c *Coin) transfer(ctx contract.Context, req *TransferRequest) error {
	from := ctx.Message().Sender
	to := loom.UnmarshalAddressPB(req.To)

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
	if ctx.FeatureEnabled(loomchain.CoinVersion1_1Feature, false) {
		return c.transferFrom(ctx, req)
	}

	return c.legacyTransferFrom(ctx, req)
}

func (c *Coin) transferFrom(ctx contract.Context, req *TransferFromRequest) error {
	spender := ctx.Message().Sender
	from := loom.UnmarshalAddressPB(req.From)
	to := loom.UnmarshalAddressPB(req.To)

	fromAccount, err := loadAccount(ctx, from)
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
