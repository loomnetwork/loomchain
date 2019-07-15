package coin

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

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
	ErrSenderBalanceTooLow = errors.New("sender balance is too low")
	ErrInvalidTotalSupply  = errors.New("Total Supply should be greater than zero")
)

var (
	economyKey       = []byte("economy")
	policyKey        = []byte("policy")
	mintingAmountKey = []byte("mintingAmount")
	mintingHeightKey = []byte("mintingHeight")
	decimals         = 18
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
	if req.Policy != nil {
		if req.Policy.ChangeRatioNumerator == 0 {
			return errors.New("ChangeRatioNumerator should be greater than zero")
		}
		if req.Policy.ChangeRatioDenominator == 0 {
			return errors.New("ChangeRatioDenominator should be greater than zero")
		}
		if req.Policy.BasePercentage == 0 {
			return errors.New("BasePercentage should be greater than zero")
		}
		if req.Policy.TotalSupply == 0 {
			return ErrInvalidTotalSupply
		}
		if req.Policy.MintingAccount == nil {
			return errors.New("Invalid Minting Account Address")
		}
		if req.Policy.BlocksGeneratedPerYear == 0 {
			return errors.New("Blocks Generated Per Year should be greater than zero")
		}
		if !strings.EqualFold("div", req.Policy.Operator) && !strings.EqualFold("exp", req.Policy.Operator) {
			return errors.New("Invalid operator - Operator should be div or exp")
		}
		addr := loom.UnmarshalAddressPB(req.Policy.MintingAccount)
		if addr.Compare(loom.RootAddress(addr.ChainID)) == 0 {
			return errors.New("Minting Account Address cannot be Root Address")
		}
		policy := &Policy{
			ChangeRatioNumerator:   req.Policy.ChangeRatioNumerator,
			ChangeRatioDenominator: req.Policy.ChangeRatioDenominator,
			TotalSupply:            req.Policy.TotalSupply,
			MintingAccount:         req.Policy.MintingAccount,
			BlocksGeneratedPerYear: req.Policy.BlocksGeneratedPerYear,
			BasePercentage:         req.Policy.BasePercentage,
			Operator:               req.Policy.Operator,
		}
		if err := ctx.Set(policyKey, policy); err != nil {
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

//Mint : to be called by CoinPolicyManager Responsible to mint coins as per various parameter defined
func Mint(ctx contract.Context) error {
	div := loom.NewBigUIntFromInt(10)
	div.Exp(div, loom.NewBigUIntFromInt(18), nil)
	var policy Policy
	var amount *common.BigUInt
	err := ctx.Get(policyKey, &policy)
	if err != nil {
		return errUtil.Wrap(err, "Failed to Get PolicyInfo")
	}
	changeRatioNumerator := loom.NewBigUIntFromInt(int64(policy.ChangeRatioNumerator))
	changeRatioDenominator := loom.NewBigUIntFromInt(int64(policy.ChangeRatioDenominator))
	blockHeight := loom.NewBigUIntFromInt(ctx.Block().Height)
	baseAmount := loom.NewBigUIntFromInt(int64(policy.TotalSupply))
	baseAmount.Mul(baseAmount, div)
	blocksGeneratedPerYear := loom.NewBigUIntFromInt(int64(policy.BlocksGeneratedPerYear))
	basePercentage := loom.NewBigUIntFromInt(int64(policy.BasePercentage))
	operator := policy.Operator
	//Checks if minting amount is computed before or it is being computed for first time
	var checkMintAmount = &types.BigUInt{
		Value: *loom.NewBigUIntFromInt(0),
	}
	err = ctx.Get(mintingAmountKey, checkMintAmount)
	if err != nil && err != contract.ErrNotFound {
		return err
	}
	if err == contract.ErrNotFound {
		//Sets Minting Height - Height at which minting started
		err := ctx.Set(mintingHeightKey, &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(ctx.Block().Height),
		})
		if err != nil {
			return err
		}
	}
	var mintingHeight = &types.BigUInt{
		Value: *loom.NewBigUIntFromInt(0),
	}
	err1 := ctx.Get(mintingHeightKey, mintingHeight)
	if err1 != nil {
		return err1
	}
	relativeHeight := big.NewInt(ctx.Block().Height).Sub(big.NewInt(ctx.Block().Height), big.NewInt(mintingHeight.Value.Int64()))
	_, modulus := big.NewInt(ctx.Block().Height).DivMod(big.NewInt(ctx.Block().Height-mintingHeight.Value.Int64()),
		big.NewInt(int64(policy.BlocksGeneratedPerYear)), big.NewInt(int64(policy.BlocksGeneratedPerYear)))
	year := blockHeight.Div(blockHeight, blocksGeneratedPerYear)
	//Computes relative Year by taking into Account minting Height and blocks Generated Per Year
	relativeYear := relativeHeight.Div(relativeHeight, blocksGeneratedPerYear.Int)
	//Minting Amount per block is set in ctx after minting amount is determined for year
	if year.Cmp(loom.NewBigUIntFromInt(0)) == 0 {
		//Determines minting amount at the beginning block of blockchain or at block height at which minting is enabled
		if modulus.Cmp(big.NewInt(1)) == 0 || err == contract.ErrNotFound {
			//Minting Amount Computation for starting year
			amount, err = ComputeforFirstYear(ctx, baseAmount, blocksGeneratedPerYear)
			if err != nil {
				return errUtil.Wrap(err, "Failed Minting Block for first year")
			}
		}

	} else {
		if modulus.Cmp(big.NewInt(1)) == 0 && err != contract.ErrNotFound {
			//Operator Support to support different inflation ratio patterns for different year
			amount, err = ComputeforConsecutiveYearBeginningWithOperator(ctx, baseAmount, changeRatioNumerator,
				changeRatioDenominator, basePercentage, blocksGeneratedPerYear, amount,
				loom.NewBigUIntFromInt(relativeYear.Int64()), operator)
			if err != nil {
				return errUtil.Wrap(err, "Failed Minting Block with operator for consecutive year")
			}
		} else if err == contract.ErrNotFound {
			amount, err = ComputeforFirstYearBlockHeightgreaterthanOneyear(ctx, baseAmount, basePercentage, blocksGeneratedPerYear)
			if err != nil {
				return errUtil.Wrap(err, "Failed Minting Block for First Year BlockHeight greater than One year")
			}
		} else {
			amount, err = ComputeforConsecutiveYearinMiddle(ctx)

		}
	}
	var mintingAmount = &types.BigUInt{
		Value: *loom.NewBigUIntFromInt(0),
	}
	err = ctx.Get(mintingAmountKey, mintingAmount)
	if err != nil {
		return err
	}
	//Boundary case when minting amount per block becomes zero
	if mintingAmount.Value.Cmp(loom.NewBigUIntFromInt(0)) == 0 {
		return nil // No more coins to be minted on block creation
	}
	return mint(ctx, loom.UnmarshalAddressPB(policy.MintingAccount), loom.NewBigUIntFromInt(mintingAmount.Value.Int64()))
}

//First year is started from the checkpoint at which minting started
//Computes minting Amount for first year if Blocks generated are less than a year, whether its year beginning or in the middle
//Divides BaseAmountforfirstyear/Blocks generated per year to get minting Amount per block
func ComputeforFirstYear(ctx contract.Context, baseAmount *common.BigUInt,
	blocksGeneratedPerYear *common.BigUInt) (*common.BigUInt, error) {
	amount := baseAmount.Div(baseAmount, blocksGeneratedPerYear)
	err := ctx.Set(mintingAmountKey, &types.BigUInt{
		Value: *amount,
	})
	if err != nil {
		return nil, err
	}
	return amount, nil
}

//Computes minting Amount for first year if Block Height is greater than blocks generated in a year, whether its year beginning or in the middle
//Computes base Amount by taking minting height into Account (Height at which minting started), applies base percentage to compute inflation for that year which is divided by blocks generated per year to get ==> minting Amount per block
func ComputeforFirstYearBlockHeightgreaterthanOneyear(ctx contract.Context, baseAmount *common.BigUInt,
	basePercentage *common.BigUInt, blocksGeneratedPerYear *common.BigUInt) (*common.BigUInt, error) {
	var econ Economy
	err := ctx.Get(economyKey, &econ)
	if err != nil {
		return nil, err
	}
	//Fetches Total Supply from coin contract and applies inflation ratio on total Supply
	baseAmount = &econ.TotalSupply.Value
	inflationforYear := baseAmount.Mul(baseAmount, basePercentage)
	inflationforYear = inflationforYear.Div(inflationforYear, loom.NewBigUIntFromInt(100))
	amount := inflationforYear.Div(inflationforYear, blocksGeneratedPerYear)
	err = ctx.Set(mintingAmountKey, &types.BigUInt{
		Value: *amount,
	})
	if err != nil {
		return nil, err
	}
	return amount, nil
}

func ComputeforConsecutiveYearBeginningWithOperator(ctx contract.Context, baseAmount *common.BigUInt,
	changeRatioNumerator *common.BigUInt,
	changeRatioDenominator *common.BigUInt, basePercentage *common.BigUInt, blocksGeneratedPerYear *common.BigUInt,
	amount *common.BigUInt, year *common.BigUInt, operator string) (*common.BigUInt, error) {
	var err error
	switch operator {
	//Computes minting amount per year, after applying change ratio to base percentage using div operator
	//Computes inflation by multiplying base Amount with base percentage. Base percentage is computed by multiplying base percentage with change ratio on which div operator is applied according to the year.
	case "div":
		changeRatioDenominator = changeRatioDenominator.Mul(changeRatioDenominator, year)
		basePercentage = basePercentage.Mul(basePercentage, changeRatioNumerator)
	//Computes minting amount per year, after applying change ratio to base percentage using exp operator
	//Computes inflation by multiplying base Amount with base percentage. Base percentage is computed by multiplying base percentage with change ratio on which exp operator is applied according to the year
	case "exp":
		changeRatioNumerator = changeRatioNumerator.Exp(changeRatioNumerator, year, nil)
		changeRatioDenominator = changeRatioDenominator.Exp(changeRatioDenominator, year, nil)
		basePercentage = basePercentage.Mul(basePercentage, changeRatioNumerator)
	default:
		err = fmt.Errorf("unrecognised operator %v", operator)
	}
	if err != nil {
		return nil, err
	}
	inflationforYear, err := ComputeInflationForYear(ctx, baseAmount, changeRatioDenominator, basePercentage)
	if err != nil {
		return nil, err
	}
	amount = inflationforYear.Div(inflationforYear, blocksGeneratedPerYear)
	//Minting Amount per block computed starting from beginning block of that year and saved to avoid re-computations
	err = ctx.Set(mintingAmountKey, &types.BigUInt{
		Value: *amount,
	})
	if err != nil {
		return nil, err
	}
	return amount, nil
}

//As minting amount has been computed for year, just derives it from context to save re-computation
func ComputeforConsecutiveYearinMiddle(ctx contract.Context) (*common.BigUInt, error) {
	var mintingAmount = &types.BigUInt{
		Value: *loom.NewBigUIntFromInt(0),
	}
	err := ctx.Get(mintingAmountKey, mintingAmount)
	if err != nil {
		return nil, err
	}
	return loom.NewBigUIntFromInt(mintingAmount.Value.Int64()), nil
}

//Computes inflation for that year which is divided by BlocksGeneratedPeryear to compute minting amount per block. Please note division is performed as last part of computation to avoid rounding off Loss
func ComputeInflationForYear(ctx contract.Context, baseAmount *common.BigUInt, changeRatioDenominator *common.BigUInt, basePercentage *common.BigUInt) (*common.BigUInt, error) {
	var econ Economy
	err := ctx.Get(economyKey, &econ)
	if err != nil {
		return nil, err
	}
	//Fetches Total Supply from coin contract and applies inflation ratio on total Supply
	baseAmount = &econ.TotalSupply.Value
	inflationforYear := baseAmount.Mul(baseAmount, basePercentage)
	if changeRatioDenominator.Cmp(loom.NewBigUIntFromInt(0)) == 0 {
		return nil, errors.New("ChangeRatioDenominator should be greater than zero")
	}
	inflationforYear = inflationforYear.Div(inflationforYear, changeRatioDenominator)
	inflationforYear = inflationforYear.Div(inflationforYear, loom.NewBigUIntFromInt(100))
	return inflationforYear, nil
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
