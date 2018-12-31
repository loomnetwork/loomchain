// +build evm

package gateway

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/pkg/errors"
)

type (
	InitRequest                     = tgtypes.TransferGatewayInitRequest
	AddOracleRequest                = tgtypes.TransferGatewayAddOracleRequest
	RemoveOracleRequest             = tgtypes.TransferGatewayRemoveOracleRequest
	GetOraclesRequest               = tgtypes.TransferGatewayGetOraclesRequest
	GetOraclesResponse              = tgtypes.TransferGatewayGetOraclesResponse
	GatewayState                    = tgtypes.TransferGatewayState
	OracleState                     = tgtypes.TransferGatewayOracleState
	ProcessEventBatchRequest        = tgtypes.TransferGatewayProcessEventBatchRequest
	GatewayStateRequest             = tgtypes.TransferGatewayStateRequest
	GatewayStateResponse            = tgtypes.TransferGatewayStateResponse
	WithdrawETHRequest              = tgtypes.TransferGatewayWithdrawETHRequest
	WithdrawTokenRequest            = tgtypes.TransferGatewayWithdrawTokenRequest
	WithdrawalReceiptRequest        = tgtypes.TransferGatewayWithdrawalReceiptRequest
	WithdrawalReceiptResponse       = tgtypes.TransferGatewayWithdrawalReceiptResponse
	ConfirmWithdrawalReceiptRequest = tgtypes.TransferGatewayConfirmWithdrawalReceiptRequest
	PendingWithdrawalsRequest       = tgtypes.TransferGatewayPendingWithdrawalsRequest
	PendingWithdrawalsResponse      = tgtypes.TransferGatewayPendingWithdrawalsResponse
	WithdrawalReceipt               = tgtypes.TransferGatewayWithdrawalReceipt
	UnclaimedToken                  = tgtypes.TransferGatewayUnclaimedToken
	ReclaimDepositorTokensRequest   = tgtypes.TransferGatewayReclaimDepositorTokensRequest
	ReclaimContractTokensRequest    = tgtypes.TransferGatewayReclaimContractTokensRequest
	LocalAccount                    = tgtypes.TransferGatewayLocalAccount
	ForeignAccount                  = tgtypes.TransferGatewayForeignAccount
	MainnetTokenDeposited           = tgtypes.TransferGatewayTokenDeposited
	MainnetTokenWithdrawn           = tgtypes.TransferGatewayTokenWithdrawn
	MainnetEvent                    = tgtypes.TransferGatewayMainnetEvent
	MainnetDepositEvent             = tgtypes.TransferGatewayMainnetEvent_Deposit
	MainnetWithdrawalEvent          = tgtypes.TransferGatewayMainnetEvent_Withdrawal
	TokenKind                       = tgtypes.TransferGatewayTokenKind
	PendingWithdrawalSummary        = tgtypes.TransferGatewayPendingWithdrawalSummary
	TokenWithdrawalSigned           = tgtypes.TransferGatewayTokenWithdrawalSigned
	TokenAmount                     = tgtypes.TransferGatewayTokenAmount

	WithdrawLoomCoinRequest = tgtypes.TransferGatewayWithdrawLoomCoinRequest
)

var (
	// Store keys
	stateKey                                = []byte("state")
	oracleStateKeyPrefix                    = []byte("oracle")
	localAccountKeyPrefix                   = []byte("account")
	foreignAccountKeyPrefix                 = []byte("facct")
	pendingContractMappingKeyPrefix         = []byte("pcm")
	contractAddrMappingKeyPrefix            = []byte("cam")
	unclaimedTokenDepositorByContractPrefix = []byte("utdc")
	unclaimedTokenByOwnerPrefix             = []byte("uto")

	// Permissions
	changeOraclesPerm   = []byte("change-oracles")
	submitEventsPerm    = []byte("submit-events")
	signWithdrawalsPerm = []byte("sign-withdrawals")
	verifyCreatorsPerm  = []byte("verify-creators")
)

const (
	// Roles
	ownerRole  = "owner"
	oracleRole = "oracle"

	// Events
	tokenWithdrawalSignedEventTopic    = "event:TokenWithdrawalSigned"
	contractMappingConfirmedEventTopic = "event:ContractMappingConfirmed"

	TokenKind_ERC721X = tgtypes.TransferGatewayTokenKind_ERC721X
	TokenKind_ERC721  = tgtypes.TransferGatewayTokenKind_ERC721
	TokenKind_ERC20   = tgtypes.TransferGatewayTokenKind_ERC20
	TokenKind_ETH     = tgtypes.TransferGatewayTokenKind_ETH

	TokenKind_LoomCoin = tgtypes.TransferGatewayTokenKind_LOOMCOIN
)

func localAccountKey(owner loom.Address) []byte {
	return util.PrefixKey(localAccountKeyPrefix, owner.Bytes())
}

func foreignAccountKey(owner loom.Address) []byte {
	return util.PrefixKey(foreignAccountKeyPrefix, owner.Bytes())
}

func oracleStateKey(oracle loom.Address) []byte {
	return util.PrefixKey(oracleStateKeyPrefix, oracle.Bytes())
}

func pendingContractMappingKey(mappingID uint64) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, mappingID)
	return util.PrefixKey(pendingContractMappingKeyPrefix, buf.Bytes())
}

func contractAddrMappingKey(contractAddr loom.Address) []byte {
	return util.PrefixKey(contractAddrMappingKeyPrefix, contractAddr.Bytes())
}

func unclaimedTokenDepositorKey(contractAddr, ownerAddr loom.Address) []byte {
	return util.PrefixKey(unclaimedTokenDepositorByContractPrefix, contractAddr.Bytes(), ownerAddr.Bytes())
}

// For iterating across all depositors with unclaimed tokens from the specified token contract
func unclaimedTokenDepositorsRangePrefix(contractAddr loom.Address) []byte {
	return util.PrefixKey(unclaimedTokenDepositorByContractPrefix, contractAddr.Bytes())
}

func unclaimedTokenKey(ownerAddr, contractAddr loom.Address) []byte {
	return util.PrefixKey(unclaimedTokenByOwnerPrefix, ownerAddr.Bytes(), contractAddr.Bytes())
}

// For iterating across all unclaimed tokens belonging to the specified depositor
func unclaimedTokensRangePrefix(ownerAddr loom.Address) []byte {
	return util.PrefixKey(unclaimedTokenByOwnerPrefix, ownerAddr.Bytes())
}

var (
	// ErrrNotAuthorized indicates that a contract method failed because the caller didn't have
	// the permission to execute that method.
	ErrNotAuthorized = errors.New("TG001: not authorized")
	// ErrInvalidRequest is a generic error that's returned when something is wrong with the
	// request message, e.g. missing or invalid fields.
	ErrInvalidRequest = errors.New("TG002: invalid request")
	// ErrPendingWithdrawalExists indicates that an account already has a withdrawal pending,
	// it must be completed or cancelled before another withdrawal can be started.
	ErrPendingWithdrawalExists   = errors.New("TG003: pending withdrawal already exists")
	ErrNoPendingWithdrawalExists = errors.New("TG004: no pending withdrawal exists")
	ErrMissingWithdrawalReceipt  = errors.New("TG005: missing withdrawal receipt")
	ErrWithdrawalReceiptSigned   = errors.New("TG006: withdrawal receipt already signed")
	ErrInvalidEventBatch         = errors.New("TG007: invalid event batch")
	ErrOwnerNotSpecified         = errors.New("TG008: owner not specified")
	ErrOracleAlreadyRegistered   = errors.New("TG009: oracle already registered")
	ErrOracleNotRegistered       = errors.New("TG010: oracle not registered")
	ErrOracleStateSaveFailed     = errors.New("TG011: failed to save oracle state")
	ErrContractMappingExists     = errors.New("TG012: contract mapping already exists")
	ErrFailedToReclaimToken      = errors.New("TG013: failed to reclaim token")
)

func reclaimDepositorTokens(ctx contract.Context, ownerAddr loom.Address) error {
	for _, entry := range ctx.Range(unclaimedTokensRangePrefix(ownerAddr)) {
		var unclaimedToken UnclaimedToken
		if err := proto.Unmarshal(entry.Value, &unclaimedToken); err != nil {
			// shouldn't actually ever happen
			return errors.Wrap(err, ErrFailedToReclaimToken.Error())
		}

		tokenAddr := loom.RootAddress("eth")
		if unclaimedToken.TokenContract != nil {
			tokenAddr = loom.UnmarshalAddressPB(unclaimedToken.TokenContract)
		}

		if err := reclaimDepositorTokensForContract(ctx, ownerAddr, tokenAddr, &unclaimedToken); err != nil {
			return err
		}
	}
	return nil
}

func reclaimDepositorTokensForContract(
	ctx contract.Context, ownerAddr, tokenAddr loom.Address, unclaimedToken *UnclaimedToken,
) error {
	failed := []*TokenAmount{}
	for _, a := range unclaimedToken.Amounts {
		err := transferTokenDeposit(ctx, ownerAddr, tokenAddr, unclaimedToken.TokenKind, a.TokenID, a.TokenAmount)
		if err != nil {
			failed = append(failed, a)
			tokenID := big.NewInt(0)
			if a.TokenID != nil {
				tokenID = a.TokenID.Value.Int
			}
			amount := big.NewInt(0)
			if a.TokenAmount != nil {
				amount = a.TokenAmount.Value.Int
			}
			ctx.Logger().Error(ErrFailedToReclaimToken.Error(),
				"owner", ownerAddr,
				"token", tokenAddr,
				"kind", unclaimedToken.TokenKind,
				"tokenID", tokenID.String(),
				"amount", amount.String(),
				"err", err)
			continue
		}
	}

	tokenKey := unclaimedTokenKey(ownerAddr, tokenAddr)
	depositorKey := unclaimedTokenDepositorKey(tokenAddr, ownerAddr)
	if len(failed) == 0 {
		ctx.Delete(tokenKey)
		ctx.Delete(depositorKey)
	} else {
		unclaimedToken.Amounts = failed
		if err := ctx.Set(tokenKey, unclaimedToken); err != nil {
			return errors.Wrap(err, ErrFailedToReclaimToken.Error())
		}
	}

	return nil
}

// Performs basic validation to ensure all required deposit fields are set.
func validateTokenDeposit(deposit *MainnetTokenDeposited) error {
	if deposit.TokenOwner == nil {
		return ErrInvalidRequest
	}

	if (deposit.TokenKind != TokenKind_ETH) && (deposit.TokenContract == nil) {
		return ErrInvalidRequest
	}

	switch deposit.TokenKind {
	case TokenKind_ERC721:
		// assume TokenID == nil means TokenID == 0
	case TokenKind_ERC721X, TokenKind_ERC20, TokenKind_ETH, TokenKind_LoomCoin:
		if deposit.TokenAmount == nil {
			return ErrInvalidRequest
		}
	default:
		return fmt.Errorf("%v deposits not supported", deposit.TokenKind)
	}
	return nil
}

// When a token is deposited to the Mainnet Gateway mint it on the DAppChain if it doesn't exist
// yet, and transfer it to the owner's DAppChain address.
func transferTokenDeposit(
	ctx contract.Context, ownerEthAddr, tokenEthAddr loom.Address,
	kind TokenKind, tokenID *types.BigUInt, tokenAmount *types.BigUInt,
) error {
	mapperAddr, err := ctx.Resolve("addressmapper")
	if err != nil {
		return err
	}

	ownerAddr, err := resolveToDAppAddr(ctx, mapperAddr, ownerEthAddr)
	if err != nil {
		return errors.Wrapf(err, "no mapping exists for account %v", ownerEthAddr)
	}

	var tokenAddr loom.Address
	if kind != TokenKind_ETH && kind != TokenKind_LoomCoin {
		tokenAddr, err = resolveToLocalContractAddr(ctx, tokenEthAddr)
		if err != nil {
			return errors.Wrapf(err, "no mapping exists for token %v", tokenEthAddr)
		}
	}

	safeTokenID := big.NewInt(0)
	if tokenID != nil {
		safeTokenID = tokenID.Value.Int
	}

	safeAmount := big.NewInt(0)
	if tokenAmount != nil {
		safeAmount = tokenAmount.Value.Int
	}

	switch kind {
	case TokenKind_ERC721:
		erc721 := newERC721Context(ctx, tokenAddr)

		exists, err := erc721.exists(safeTokenID)
		if err != nil {
			return err
		}

		if !exists {
			if err := erc721.mintToGateway(safeTokenID); err != nil {
				return errors.Wrapf(err, "failed to mint token %v - %s", tokenAddr, safeTokenID.String())
			}
		}

		// At this point the token is owned by the associated token contract, so transfer it back to the
		// original owner...
		if err := erc721.safeTransferFrom(ctx.ContractAddress(), ownerAddr, safeTokenID); err != nil {
			return errors.Wrapf(err, "failed to transfer ERC721 token")
		}

	case TokenKind_ERC721X:
		erc721x := newERC721XContext(ctx, tokenAddr)

		availableFunds, err := erc721x.balanceOf(ctx.ContractAddress(), safeTokenID)
		if err != nil {
			return err
		}

		if availableFunds.Cmp(safeAmount) < 0 {
			shortage := big.NewInt(0).Sub(safeAmount, availableFunds)
			if err := erc721x.mintToGateway(safeTokenID, shortage); err != nil {
				return errors.Wrapf(err, "failed to mint tokens %v - %s", tokenAddr, safeTokenID.String())
			}
		}

		if err := erc721x.safeTransferFrom(ctx.ContractAddress(), ownerAddr, safeTokenID, safeAmount); err != nil {
			return errors.Wrapf(err, "failed to transfer ERC721X token")
		}

	case TokenKind_ERC20:
		erc20 := newERC20Context(ctx, tokenAddr)
		availableFunds, err := erc20.balanceOf(ctx.ContractAddress())
		if err != nil {
			return err
		}

		// If the DAppChain Gateway doesn't have sufficient funds to complete the transfer then
		// try to mint the required amount...
		if availableFunds.Cmp(safeAmount) < 0 {
			if err := erc20.mintToGateway(safeAmount); err != nil {
				return errors.Wrapf(err, "failed to mint tokens %v - %s", tokenAddr, safeAmount.String())
			}
		}

		if err := erc20.transfer(ownerAddr, safeAmount); err != nil {
			return errors.Wrap(err, "failed to transfer ERC20 tokens")
		}

	case TokenKind_ETH:
		eth := newETHContext(ctx)
		availableFunds, err := eth.balanceOf(ctx.ContractAddress())
		if err != nil {
			return err
		}

		// If the DAppChain Gateway doesn't have sufficient funds to complete the transfer then
		// try to mint the required amount...
		if availableFunds.Cmp(safeAmount) < 0 {
			if err := eth.mintToGateway(safeAmount); err != nil {
				return errors.Wrapf(err, "failed to mint ETH - %s", safeAmount.String())
			}
		}

		if err := eth.transfer(ownerAddr, safeAmount); err != nil {
			return errors.Wrap(err, "failed to transfer ETH")
		}
	case TokenKind_LoomCoin:
		coin := newCoinContext(ctx)
		availableFunds, err := coin.balanceOf(ctx.ContractAddress())
		if err != nil {
			return err
		}

		if availableFunds.Cmp(safeAmount) < 0 {
			if err := coin.mintToGateway(safeAmount); err != nil {
				return errors.Wrapf(err, "failed to mint loom coin - %s", safeAmount.String())
			}
		}

		if err := coin.transfer(ownerAddr, safeAmount); err != nil {
			return errors.Wrap(err, "failed to transfer loom coin")
		}
	}

	return nil
}

func storeUnclaimedToken(ctx contract.Context, deposit *MainnetTokenDeposited) error {
	ownerAddr := loom.UnmarshalAddressPB(deposit.TokenOwner)
	tokenAddr := loom.RootAddress("eth")

	if deposit.TokenContract != nil {
		tokenAddr = loom.UnmarshalAddressPB(deposit.TokenContract)
	}

	unclaimedToken := UnclaimedToken{
		TokenContract: deposit.TokenContract,
		TokenKind:     deposit.TokenKind,
	}
	tokenKey := unclaimedTokenKey(ownerAddr, tokenAddr)
	err := ctx.Get(tokenKey, &unclaimedToken)
	if err != nil && err != contract.ErrNotFound {
		return errors.Wrapf(err, "failed to load unclaimed token for %v", ownerAddr)
	}

	switch deposit.TokenKind {
	case TokenKind_ERC721:
		unclaimedToken.Amounts = append(unclaimedToken.Amounts, &TokenAmount{
			TokenID: deposit.TokenID,
		})

	case TokenKind_ERC721X:
		// store the total amount per token ID
		var oldAmount *TokenAmount
		for _, a := range unclaimedToken.Amounts {
			if a.TokenID.Value.Cmp(&deposit.TokenID.Value) == 0 {
				oldAmount = a
				break
			}
		}
		if oldAmount != nil {
			val := &oldAmount.TokenAmount.Value
			val.Add(val, &deposit.TokenAmount.Value)
		} else {
			unclaimedToken.Amounts = append(unclaimedToken.Amounts, &TokenAmount{
				TokenID:     deposit.TokenID,
				TokenAmount: deposit.TokenAmount,
			})
		}

	case TokenKind_ERC20, TokenKind_ETH, TokenKind_LoomCoin:
		// store a single total amount
		oldAmount := big.NewInt(0)
		if len(unclaimedToken.Amounts) == 1 {
			oldAmount = unclaimedToken.Amounts[0].TokenAmount.Value.Int
		}
		newAmount := oldAmount.Add(oldAmount, deposit.TokenAmount.Value.Int)
		unclaimedToken.Amounts = []*TokenAmount{
			&TokenAmount{
				TokenAmount: &types.BigUInt{Value: *loom.NewBigUInt(newAmount)},
			},
		}
	}

	// make it possible to iterate all depositors with unclaimed tokens by token contract
	depositorKey := unclaimedTokenDepositorKey(tokenAddr, ownerAddr)
	if err := ctx.Set(depositorKey, ownerAddr.MarshalPB()); err != nil {
		return err
	}
	return ctx.Set(tokenKey, &unclaimedToken)
}

// When a token is withdrawn from the Mainnet Gateway find the corresponding withdrawal receipt
// and remove it from the owner's account, once the receipt is removed the owner will be able to
// initiate another withdrawal to Mainnet.
func completeTokenWithdraw(ctx contract.Context, state *GatewayState, withdrawal *MainnetTokenWithdrawn) error {
	if withdrawal.TokenOwner == nil {
		return ErrInvalidRequest
	}

	if (withdrawal.TokenKind != TokenKind_ETH) && (withdrawal.TokenContract == nil) {
		return ErrInvalidRequest
	}

	switch withdrawal.TokenKind {
	case TokenKind_ERC721:
		// assume TokenID == nil means TokenID == 0
	case TokenKind_ERC721X, TokenKind_ERC20, TokenKind_ETH, TokenKind_LoomCoin:
		if withdrawal.TokenAmount == nil {
			return ErrInvalidRequest
		}
	default:
		return fmt.Errorf("%v withdrawals not supported", withdrawal.TokenKind)
	}

	ownerEthAddr := loom.UnmarshalAddressPB(withdrawal.TokenOwner)
	foreignAccount, err := loadForeignAccount(ctx, ownerEthAddr)
	if err != nil {
		return err
	}

	if foreignAccount.CurrentWithdrawer == nil {
		return fmt.Errorf("no pending withdrawal to %v found", ownerEthAddr)
	}

	ownerAddr := loom.UnmarshalAddressPB(foreignAccount.CurrentWithdrawer)
	account, err := loadLocalAccount(ctx, ownerAddr)
	if err != nil {
		return err
	}

	if account.WithdrawalReceipt == nil {
		return fmt.Errorf("no pending withdrawal from %v to %v found", ownerAddr, ownerEthAddr)
	}
	// TODO: check contract address & token ID match the receipt
	account.WithdrawalReceipt = nil
	foreignAccount.WithdrawalNonce++
	foreignAccount.CurrentWithdrawer = nil

	if err := saveLocalAccount(ctx, account); err != nil {
		return err
	}

	if err := saveForeignAccount(ctx, foreignAccount); err != nil {
		return err
	}

	return removeTokenWithdrawer(ctx, state, ownerAddr)
}

func loadState(ctx contract.StaticContext) (*GatewayState, error) {
	var state GatewayState
	err := ctx.Get(stateKey, &state)
	if err != nil && err != contract.ErrNotFound {
		return nil, err
	}
	return &state, nil
}

func saveState(ctx contract.Context, state *GatewayState) error {
	return ctx.Set(stateKey, state)
}

// Returns the address of the DAppChain account or contract that corresponds to the given Ethereum address
func resolveToDAppAddr(ctx contract.StaticContext, mapperAddr, ethAddr loom.Address) (loom.Address, error) {
	var resp address_mapper.GetMappingResponse
	req := &address_mapper.GetMappingRequest{From: ethAddr.MarshalPB()}
	if err := contract.StaticCallMethod(ctx, mapperAddr, "GetMapping", req, &resp); err != nil {
		return loom.Address{}, err
	}
	return loom.UnmarshalAddressPB(resp.To), nil
}

// Returns the address of the Ethereum account or contract that corresponds to the given DAppChain address
func resolveToEthAddr(ctx contract.StaticContext, mapperAddr, dappAddr loom.Address) (loom.Address, error) {
	var resp address_mapper.GetMappingResponse
	req := &address_mapper.GetMappingRequest{From: dappAddr.MarshalPB()}
	if err := contract.StaticCallMethod(ctx, mapperAddr, "GetMapping", req, &resp); err != nil {
		return loom.Address{}, err
	}
	return loom.UnmarshalAddressPB(resp.To), nil
}

func loadLocalAccount(ctx contract.StaticContext, owner loom.Address) (*LocalAccount, error) {
	account := LocalAccount{Owner: owner.MarshalPB()}
	err := ctx.Get(localAccountKey(owner), &account)
	if err != nil && err != contract.ErrNotFound {
		return nil, errors.Wrapf(err, "failed to load account for %v", owner)
	}
	return &account, nil
}

func saveLocalAccount(ctx contract.Context, acct *LocalAccount) error {
	ownerAddr := loom.UnmarshalAddressPB(acct.Owner)
	if err := ctx.Set(localAccountKey(ownerAddr), acct); err != nil {
		return errors.Wrapf(err, "failed to save account for %v", ownerAddr)
	}
	return nil
}

func loadForeignAccount(ctx contract.StaticContext, owner loom.Address) (*ForeignAccount, error) {
	account := ForeignAccount{Owner: owner.MarshalPB()}
	err := ctx.Get(foreignAccountKey(owner), &account)
	if err != nil && err != contract.ErrNotFound {
		return nil, errors.Wrapf(err, "failed to load account for %v", owner)
	}
	return &account, nil
}

func saveForeignAccount(ctx contract.Context, acct *ForeignAccount) error {
	ownerAddr := loom.UnmarshalAddressPB(acct.Owner)
	if err := ctx.Set(foreignAccountKey(ownerAddr), acct); err != nil {
		return errors.Wrapf(err, "failed to save account for %v", ownerAddr)
	}
	return nil
}

func addTokenWithdrawer(ctx contract.StaticContext, state *GatewayState, owner loom.Address) error {
	// TODO: replace this with ctx.Range()
	ownerAddrPB := owner.MarshalPB()
	for _, addr := range state.TokenWithdrawers {
		if ownerAddrPB.ChainId == addr.ChainId && ownerAddrPB.Local.Compare(addr.Local) == 0 {
			return ErrPendingWithdrawalExists
		}
	}
	state.TokenWithdrawers = append(state.TokenWithdrawers, ownerAddrPB)
	return nil
}

func removeTokenWithdrawer(ctx contract.StaticContext, state *GatewayState, owner loom.Address) error {
	ownerAddrPB := owner.MarshalPB()
	for i, addr := range state.TokenWithdrawers {
		if ownerAddrPB.ChainId == addr.ChainId && ownerAddrPB.Local.Compare(addr.Local) == 0 {
			// TODO: keep the list sorted
			state.TokenWithdrawers[i] = state.TokenWithdrawers[len(state.TokenWithdrawers)-1]
			state.TokenWithdrawers = state.TokenWithdrawers[:len(state.TokenWithdrawers)-1]
			return nil
		}
	}

	return ErrNoPendingWithdrawalExists
}

func addOracle(ctx contract.Context, oracleAddr loom.Address) error {
	ctx.GrantPermissionTo(oracleAddr, submitEventsPerm, oracleRole)
	ctx.GrantPermissionTo(oracleAddr, signWithdrawalsPerm, oracleRole)
	ctx.GrantPermissionTo(oracleAddr, verifyCreatorsPerm, oracleRole)

	err := ctx.Set(oracleStateKey(oracleAddr), &OracleState{Address: oracleAddr.MarshalPB()})
	if err != nil {
		return errors.Wrap(err, ErrOracleStateSaveFailed.Error())
	}
	return nil
}
