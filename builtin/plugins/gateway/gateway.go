// +build evm

package gateway

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/pkg/errors"

	"github.com/loomnetwork/go-loom/client"
)

type (
	InitRequest                        = tgtypes.TransferGatewayInitRequest
	AddOracleRequest                   = tgtypes.TransferGatewayAddOracleRequest
	RemoveOracleRequest                = tgtypes.TransferGatewayRemoveOracleRequest
	GetOraclesRequest                  = tgtypes.TransferGatewayGetOraclesRequest
	GetOraclesResponse                 = tgtypes.TransferGatewayGetOraclesResponse
	GatewayState                       = tgtypes.TransferGatewayState
	OracleState                        = tgtypes.TransferGatewayOracleState
	ProcessEventBatchRequest           = tgtypes.TransferGatewayProcessEventBatchRequest
	GatewayStateRequest                = tgtypes.TransferGatewayStateRequest
	GatewayStateResponse               = tgtypes.TransferGatewayStateResponse
	WithdrawETHRequest                 = tgtypes.TransferGatewayWithdrawETHRequest
	WithdrawTokenRequest               = tgtypes.TransferGatewayWithdrawTokenRequest
	WithdrawalReceiptRequest           = tgtypes.TransferGatewayWithdrawalReceiptRequest
	WithdrawalReceiptResponse          = tgtypes.TransferGatewayWithdrawalReceiptResponse
	ConfirmWithdrawalReceiptRequest    = tgtypes.TransferGatewayConfirmWithdrawalReceiptRequest
	PendingWithdrawalsRequest          = tgtypes.TransferGatewayPendingWithdrawalsRequest
	PendingWithdrawalsResponse         = tgtypes.TransferGatewayPendingWithdrawalsResponse
	WithdrawalReceipt                  = tgtypes.TransferGatewayWithdrawalReceipt
	GetUnclaimedTokensRequest          = tgtypes.TransferGatewayGetUnclaimedTokensRequest
	GetUnclaimedTokensResponse         = tgtypes.TransferGatewayGetUnclaimedTokensResponse
	GetUnclaimedContractTokensRequest  = tgtypes.TransferGatewayGetUnclaimedContractTokensRequest
	GetUnclaimedContractTokensResponse = tgtypes.TransferGatewayGetUnclaimedContractTokensResponse
	UnclaimedToken                     = tgtypes.TransferGatewayUnclaimedToken
	ReclaimDepositorTokensRequest      = tgtypes.TransferGatewayReclaimDepositorTokensRequest
	ReclaimContractTokensRequest       = tgtypes.TransferGatewayReclaimContractTokensRequest
	LocalAccount                       = tgtypes.TransferGatewayLocalAccount
	ForeignAccount                     = tgtypes.TransferGatewayForeignAccount
	MainnetTokenDeposited              = tgtypes.TransferGatewayTokenDeposited
	MainnetTokenWithdrawn              = tgtypes.TransferGatewayTokenWithdrawn
	MainnetEvent                       = tgtypes.TransferGatewayMainnetEvent
	MainnetDepositEvent                = tgtypes.TransferGatewayMainnetEvent_Deposit
	MainnetWithdrawalEvent             = tgtypes.TransferGatewayMainnetEvent_Withdrawal
	TokenKind                          = tgtypes.TransferGatewayTokenKind
	PendingWithdrawalSummary           = tgtypes.TransferGatewayPendingWithdrawalSummary
	TokenWithdrawalSigned              = tgtypes.TransferGatewayTokenWithdrawalSigned
	TokenAmount                        = tgtypes.TransferGatewayTokenAmount
	MainnetProcessEventError           = tgtypes.TransferGatewayProcessMainnetEventError
	ReclaimError                       = tgtypes.TransferGatewayReclaimError
	WithdrawETHError                   = tgtypes.TransferGatewayWithdrawETHError
	WithdrawTokenError                 = tgtypes.TransferGatewayWithdrawTokenError
	WithdrawLoomCoinError              = tgtypes.TransferGatewayWithdrawLoomCoinError
	MainnetEventTxHashInfo             = tgtypes.TransferGatewayMainnetEventTxHashInfo

	WithdrawLoomCoinRequest = tgtypes.TransferGatewayWithdrawLoomCoinRequest

	TrustedValidatorsRequest  = tgtypes.TransferGatewayTrustedValidatorsRequest
	TrustedValidatorsResponse = tgtypes.TransferGatewayTrustedValidatorsResponse

	UpdateTrustedValidatorsRequest = tgtypes.TransferGatewayUpdateTrustedValidatorsRequest

	TrustedValidators = tgtypes.TransferGatewayTrustedValidators

	ValidatorAuthConfig = tgtypes.TransferGatewayValidatorAuthConfig

	GetValidatorAuthStrategyRequest  = tgtypes.TransferGatewayGetValidatorAuthStrategyRequest
	GetValidatorAuthStrategyResponse = tgtypes.TransferGatewayGetValidatorAuthStrategyResponse

	UpdateValidatorAuthStrategyRequest = tgtypes.TransferGatewayUpdateValidatorAuthStrategyRequest

	ConfirmWithdrawalReceiptRequestV2 = tgtypes.TransferGatewayConfirmWithdrawalReceiptRequestV2

	SubmitDepositTxHashRequest         = tgtypes.TransferGatewaySubmitDepositTxHashRequest
	ClearInvalidDepositTxHashRequest   = tgtypes.TransferGatewayClearInvalidDepositTxHashRequest
	UnprocessedDepositTxHashesResponse = tgtypes.TransferGatewayUnprocessedDepositTxHashesResponse
	UnprocessedDepositTxHashesRequest  = tgtypes.TransferGatewayUnprocessedDepositTxHashesRequest

	ExtendedState = tgtypes.TransferGatewayExtendedState

	TransferGatewayTxHash = tgtypes.TransferGatewayTxHash
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
	seenTxHashKeyPrefix                     = []byte("stx")

	DepositTxHashKeyPrefix = []byte("dth")
	extendedStateKey       = []byte("ext-state")

	// Permissions
	changeOraclesPerm        = []byte("change-oracles")
	submitEventsPerm         = []byte("submit-events")
	signWithdrawalsPerm      = []byte("sign-withdrawals")
	verifyCreatorsPerm       = []byte("verify-creators")
	clearInvalidTxHashesPerm = []byte("clear-invalid-txhashes")

	validatorAuthConfigKey = []byte("validator-authcfg")

	// Tron's TRX fake fixed address to map to dApp's contract
	TRXTokenAddr = loom.MustParseAddress("tron:0x0000000000000000000000000000000000000001")
)

const (
	// Roles
	ownerRole  = "owner"
	oracleRole = "oracle"

	// Events
	tokenWithdrawalSignedEventTopic    = "event:TokenWithdrawalSigned"
	contractMappingConfirmedEventTopic = "event:ContractMappingConfirmed"
	withdrawETHTopic                   = "event:WithdrawETH"
	withdrawLoomCoinTopic              = "event:WithdrawLoomCoin"
	withdrawTokenTopic                 = "event:WithdrawToken"
	mainnetDepositEventTopic           = "event:MainnetDepositEvent"
	mainnetWithdrawalEventTopic        = "event:MainnetWithdrawalEvent"
	mainnetProcessEventErrorTopic      = "event:MainnetProcessEventError"
	reclaimErrorTopic                  = "event:ReclaimError"
	withdrawETHErrorTopic              = "event:WithdrawETHError"
	withdrawLoomCoinErrorTopic         = "event:WithdrawLoomCoinError"
	withdrawTokenErrorTopic            = "event:WithdrawTokenError"
	storeUnclaimedTokenTopic           = "event:StoreUnclaimedToken"

	TokenKind_ERC721X = tgtypes.TransferGatewayTokenKind_ERC721X
	TokenKind_ERC721  = tgtypes.TransferGatewayTokenKind_ERC721
	TokenKind_ERC20   = tgtypes.TransferGatewayTokenKind_ERC20
	TokenKind_ETH     = tgtypes.TransferGatewayTokenKind_ETH
	TokenKind_TRX     = tgtypes.TransferGatewayTokenKind_TRX
	TokenKind_TRC20   = tgtypes.TransferGatewayTokenKind_TRC20

	TokenKind_LoomCoin = tgtypes.TransferGatewayTokenKind_LOOMCOIN
)

func localAccountKey(owner loom.Address) []byte {
	return util.PrefixKey(localAccountKeyPrefix, owner.Bytes())
}

func foreignAccountKey(owner loom.Address) []byte {
	return util.PrefixKey(foreignAccountKeyPrefix, owner.Bytes())
}

func DepositTxHashKey(owner loom.Address) []byte {
	return util.PrefixKey(DepositTxHashKeyPrefix, owner.Bytes())
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

func seenTxHashKey(txHash []byte) []byte {
	return util.PrefixKey(seenTxHashKeyPrefix, txHash)
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
	ErrNotEnoughSignatures       = errors.New("TG014: failed to recover enough signatures from trusted validators")

	ErrUnprocessedTxHashAlreadyExists = errors.New("TG015: unprocessed tx hash already exists")
	ErrNoUnprocessedTxHashExists      = errors.New("TG016: no unprocessed tx hash exists")
	ErrCheckTxHashIsDisabled          = errors.New("TG017: check txhash feature is disabled")
)

type GatewayType int

const (
	EthereumGateway GatewayType = 0 // default type
	LoomCoinGateway GatewayType = 1
	TronGateway     GatewayType = 2
)

type Gateway struct {
	Type GatewayType
}

func (gw *Gateway) Meta() (plugin.Meta, error) {
	switch gw.Type {
	case EthereumGateway:
		return plugin.Meta{
			Name:    "gateway",
			Version: "0.1.0",
		}, nil
	case LoomCoinGateway:
		return plugin.Meta{
			Name:    "loomcoin-gateway",
			Version: "0.1.0",
		}, nil
	case TronGateway:
		return plugin.Meta{
			Name:    "tron-gateway",
			Version: "0.1.0",
		}, nil
	}
	return plugin.Meta{}, errors.Errorf("invalid Gateway Type: %v", gw.Type)
}

func (gw *Gateway) Init(ctx contract.Context, req *InitRequest) error {
	if req.Owner == nil {
		return ErrOwnerNotSpecified
	}
	ownerAddr := loom.UnmarshalAddressPB(req.Owner)
	ctx.GrantPermissionTo(ownerAddr, changeOraclesPerm, ownerRole)

	for _, oracleAddrPB := range req.Oracles {
		oracleAddr := loom.UnmarshalAddressPB(oracleAddrPB)
		if err := addOracle(ctx, oracleAddr); err != nil {
			return err
		}
	}

	return saveState(ctx, &GatewayState{
		Owner:                 req.Owner,
		NextContractMappingID: 1,
		LastMainnetBlockNum:   req.FirstMainnetBlockNum,
	})
}

func (gw *Gateway) GetValidatorAuthStrategy(ctx contract.StaticContext, req *GetValidatorAuthStrategyRequest) (*GetValidatorAuthStrategyResponse, error) {
	validatorAuthConfig := ValidatorAuthConfig{}
	if err := ctx.Get(validatorAuthConfigKey, &validatorAuthConfig); err != nil {
		if err == contract.ErrNotFound {
			return &GetValidatorAuthStrategyResponse{}, nil
		}
		return nil, err
	}

	return &GetValidatorAuthStrategyResponse{AuthStrategy: validatorAuthConfig.AuthStrategy}, nil
}

func (gw *Gateway) GetTrustedValidators(ctx contract.StaticContext, req *TrustedValidatorsRequest) (*TrustedValidatorsResponse, error) {
	validatorAuthConfig := ValidatorAuthConfig{}
	if err := ctx.Get(validatorAuthConfigKey, &validatorAuthConfig); err != nil {
		if err == contract.ErrNotFound {
			return &TrustedValidatorsResponse{}, nil
		}
		return nil, err
	}
	return &TrustedValidatorsResponse{TrustedValidators: validatorAuthConfig.TrustedValidators}, nil
}

func (gw *Gateway) UpdateTrustedValidators(ctx contract.Context, req *UpdateTrustedValidatorsRequest) error {
	if req.TrustedValidators == nil {
		return ErrInvalidRequest
	}

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	if loom.UnmarshalAddressPB(state.Owner).Compare(ctx.Message().Sender) != 0 {
		return ErrNotAuthorized
	}

	validatorAuthConfig := ValidatorAuthConfig{}
	if err := ctx.Get(validatorAuthConfigKey, &validatorAuthConfig); err != nil {
		if err != contract.ErrNotFound {
			return err
		}
	}

	validatorAuthConfig.TrustedValidators = req.TrustedValidators
	return ctx.Set(validatorAuthConfigKey, &validatorAuthConfig)

}

func (gw *Gateway) UpdateValidatorAuthStrategy(ctx contract.Context, req *UpdateValidatorAuthStrategyRequest) error {
	if req.AuthStrategy != tgtypes.ValidatorAuthStrategy_USE_DPOS_VALIDATORS && req.AuthStrategy != tgtypes.ValidatorAuthStrategy_USE_TRUSTED_VALIDATORS {
		return ErrInvalidRequest
	}

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	if loom.UnmarshalAddressPB(state.Owner).Compare(ctx.Message().Sender) != 0 {
		return ErrNotAuthorized
	}

	validatorAuthConfig := ValidatorAuthConfig{}
	if err := ctx.Get(validatorAuthConfigKey, &validatorAuthConfig); err != nil {
		if err != contract.ErrNotFound {
			return err
		}
	}

	if req.AuthStrategy == tgtypes.ValidatorAuthStrategy_USE_TRUSTED_VALIDATORS && validatorAuthConfig.TrustedValidators == nil {
		return ErrInvalidRequest
	}

	validatorAuthConfig.AuthStrategy = req.AuthStrategy

	return ctx.Set(validatorAuthConfigKey, &validatorAuthConfig)
}

func (gw *Gateway) AddOracle(ctx contract.Context, req *AddOracleRequest) error {
	if req.Oracle == nil {
		return ErrInvalidRequest
	}

	if ok, _ := ctx.HasPermission(changeOraclesPerm, []string{ownerRole}); !ok {
		return ErrNotAuthorized
	}

	oracleAddr := loom.UnmarshalAddressPB(req.Oracle)
	if ctx.Has(oracleStateKey(oracleAddr)) {
		return ErrOracleAlreadyRegistered
	}

	return addOracle(ctx, oracleAddr)
}

func (gw *Gateway) RemoveOracle(ctx contract.Context, req *RemoveOracleRequest) error {
	if req.Oracle == nil {
		return ErrInvalidRequest
	}

	if ok, _ := ctx.HasPermission(changeOraclesPerm, []string{ownerRole}); !ok {
		return ErrNotAuthorized
	}

	oracleAddr := loom.UnmarshalAddressPB(req.Oracle)
	if !ctx.Has(oracleStateKey(oracleAddr)) {
		return ErrOracleNotRegistered
	}

	return removeOracle(ctx, oracleAddr)
}

func (gw *Gateway) ReplaceOwner(ctx contract.Context, req *AddOracleRequest) error {
	if req.Oracle == nil {
		return ErrInvalidRequest
	}

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	if loom.UnmarshalAddressPB(state.Owner).Compare(ctx.Message().Sender) != 0 {
		return ErrNotAuthorized
	}

	// Revoke permissions from old owner
	oldOwnerAddr := loom.UnmarshalAddressPB(state.Owner)
	ctx.RevokePermissionFrom(oldOwnerAddr, changeOraclesPerm, ownerRole)

	// Update owner and grant permissions
	state.Owner = req.Oracle
	ownerAddr := loom.UnmarshalAddressPB(req.Oracle)
	ctx.GrantPermissionTo(ownerAddr, changeOraclesPerm, ownerRole)

	return saveState(ctx, state)
}

func removeOracle(ctx contract.Context, oracleAddr loom.Address) error {
	ctx.RevokePermissionFrom(oracleAddr, submitEventsPerm, oracleRole)
	ctx.RevokePermissionFrom(oracleAddr, signWithdrawalsPerm, oracleRole)
	ctx.RevokePermissionFrom(oracleAddr, verifyCreatorsPerm, oracleRole)
	ctx.RevokePermissionFrom(oracleAddr, clearInvalidTxHashesPerm, oracleRole)

	ctx.Delete(oracleStateKey(oracleAddr))
	return nil
}

func (gw *Gateway) GetOracles(ctx contract.StaticContext, req *GetOraclesRequest) (*GetOraclesResponse, error) {
	var oracles []*OracleState
	for _, entry := range ctx.Range(oracleStateKeyPrefix) {
		var oracleState OracleState
		if err := proto.Unmarshal(entry.Value, &oracleState); err != nil {
			return nil, err
		}
		oracles = append(oracles, &oracleState)
	}
	return &GetOraclesResponse{
		Oracles: oracles,
	}, nil
}

// ProcessDepositEventByTxHash tries to submit deposit events by tx hash
// This method expects that TGCheckTxHashFeature is enabled on chain
func (gw *Gateway) ProcessDepositEventByTxHash(ctx contract.Context, req *ProcessEventBatchRequest) error {
	if ok, _ := ctx.HasPermission(submitEventsPerm, []string{oracleRole}); !ok {
		return ErrNotAuthorized
	}

	checkTxHash := ctx.FeatureEnabled(loomchain.TGCheckTxHashFeature, false)
	if !checkTxHash {
		return ErrCheckTxHashIsDisabled
	}

	for _, ev := range req.Events {
		switch payload := ev.Payload.(type) {
		case *tgtypes.TransferGatewayMainnetEvent_Deposit:
			// We need to pass ev here, as emitProcessEvent expects it.
			if err := gw.handleDeposit(ctx, ev, checkTxHash); err != nil {
				return err
			}
		case nil:
			ctx.Logger().Error("[Transfer Gateway] missing event payload")
			continue
		default:
			ctx.Logger().Error("[Transfer Gateway] only deposit event is supported to be submitted by txhash, got %T", payload)
			continue
		}
	}

	return nil
}

func (gw *Gateway) ProcessEventBatch(ctx contract.Context, req *ProcessEventBatchRequest) error {
	if ok, _ := ctx.HasPermission(submitEventsPerm, []string{oracleRole}); !ok {
		return ErrNotAuthorized
	}

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	blockCount := 0           // number of blocks that were actually processed in this batch
	lastEthBlock := uint64(0) // the last block processed in this batch
	checkTxHash := ctx.FeatureEnabled(loomchain.TGCheckTxHashFeature, false)

	for _, ev := range req.Events {
		// Events in the batch are expected to be ordered by block, so a batch should contain
		// events from block N, followed by events from block N+1, any other order is invalid.
		if ev.EthBlock < lastEthBlock {
			ctx.Logger().Error("[Transfer Gateway] invalid event batch, block has already been processed",
				"block", ev.EthBlock)
			return ErrInvalidEventBatch
		}

		// Multiple validators might submit batches with overlapping block ranges because the
		// Gateway oracles will fetch events from Ethereum at different times, with different
		// latencies, etc. Simply skip blocks that have already been processed.
		if ev.EthBlock <= state.LastMainnetBlockNum {
			continue
		}

		switch payload := ev.Payload.(type) {
		case *tgtypes.TransferGatewayMainnetEvent_Deposit:
			// We need to pass ev here, as emitProcessEvent expects it.
			if err := gw.handleDeposit(ctx, ev, checkTxHash); err != nil {
				return err
			}
		case *tgtypes.TransferGatewayMainnetEvent_Withdrawal:
			if !isTokenKindAllowed(gw.Type, payload.Withdrawal.TokenKind) {
				return ErrInvalidRequest
			}

			if checkTxHash {
				if len(payload.Withdrawal.TxHash) == 0 {
					ctx.Logger().Error("[Transfer Gateway] missing Mainnet withdrawal tx hash")
					return ErrInvalidRequest
				}
				if hasSeenTxHash(ctx, payload.Withdrawal.TxHash) {
					msg := fmt.Sprintf("[TransferGateway] skipping Mainnet withdrawal with dupe tx hash: %x",
						payload.Withdrawal.TxHash,
					)
					ctx.Logger().Info(msg)
					emitProcessEventError(ctx, msg, ev)
					continue
				}
			}

			if err := completeTokenWithdraw(ctx, state, payload.Withdrawal); err != nil {
				ctx.Logger().Error("[Transfer Gateway] failed to process Mainnet withdrawal", "err", err)
				emitProcessEventError(ctx, "[TransferGateway completeTokenWithdraw]"+err.Error(), ev)
				continue
			}

			withdrawal, err := proto.Marshal(payload.Withdrawal)
			if err != nil {
				return err
			}
			ctx.EmitTopics(withdrawal, mainnetWithdrawalEventTopic)

			if checkTxHash {
				if err := saveSeenTxHash(ctx, payload.Withdrawal.TxHash, payload.Withdrawal.TokenKind); err != nil {
					return err
				}
			}

		case nil:
			ctx.Logger().Error("[Transfer Gateway] missing event payload")
			continue

		default:
			ctx.Logger().Error("[Transfer Gateway] unknown event payload type %T", payload)
			continue
		}

		if ev.EthBlock > lastEthBlock {
			blockCount++
			lastEthBlock = ev.EthBlock
		}
	}

	// If there are no new events in this batch return an error so that the batch tx isn't
	// propagated to the other nodes.
	if blockCount == 0 {
		return fmt.Errorf("no new events found in the batch")
	}

	state.LastMainnetBlockNum = lastEthBlock

	return saveState(ctx, state)
}

func (gw *Gateway) handleDeposit(ctx contract.Context, ev *MainnetEvent, checkTxHash bool) error {
	var err error

	// This should be already checked in Process* function, so returning false here means function
	// was invoked from somewhere else.
	payload, ok := ev.Payload.(*tgtypes.TransferGatewayMainnetEvent_Deposit)
	if !ok {
		return fmt.Errorf("[Transfer Gateway] unknown event payload type %T", payload)
	}

	if !isTokenKindAllowed(gw.Type, payload.Deposit.TokenKind) {
		return ErrInvalidRequest
	}

	if err := validateTokenDeposit(payload.Deposit); err != nil {
		ctx.Logger().Error("[Transfer Gateway] failed to process Mainnet deposit", "err", err)
		emitProcessEventError(ctx, "[TransferGateway validateTokenDeposit]"+err.Error(), ev)
		return nil
	}

	if checkTxHash {
		if len(payload.Deposit.TxHash) == 0 {
			ctx.Logger().Error("[Transfer Gateway] missing Mainnet deposit tx hash")
			return ErrInvalidRequest
		}
		if hasSeenTxHash(ctx, payload.Deposit.TxHash) {
			msg := fmt.Sprintf("[TransferGateway] skipping Mainnet deposit with dupe tx hash: %x",
				payload.Deposit.TxHash,
			)
			ctx.Logger().Info(msg)
			emitProcessEventError(ctx, msg, ev)
			return nil
		}
	}

	ownerAddr := loom.UnmarshalAddressPB(payload.Deposit.TokenOwner)
	tokenAddr := loom.RootAddress("eth")
	if payload.Deposit.TokenContract != nil {
		tokenAddr = loom.UnmarshalAddressPB(payload.Deposit.TokenContract)
	}

	// TODO: This should be behind feature flag
	if err := clearDepositTxHashIfExists(ctx, loom.UnmarshalAddressPB(payload.Deposit.TokenOwner)); err != nil {
		return err
	}

	err = transferTokenDeposit(
		ctx, ownerAddr, tokenAddr,
		payload.Deposit.TokenKind, payload.Deposit.TokenID, payload.Deposit.TokenAmount)
	if err != nil {
		ctx.Logger().Error("[Transfer Gateway] failed to transfer Mainnet deposit", "err", err)
		emitProcessEventError(ctx, "[TransferGateway transferTokenDeposit]"+err.Error(), ev)
		if err := storeUnclaimedToken(ctx, payload.Deposit); err != nil {
			// this is a fatal error, discard the entire batch so that this deposit event
			// is resubmitted again in the next batch (hopefully after whatever caused this
			// error is resolved)
			emitProcessEventError(ctx, err.Error(), ev)
			return err
		}
	} else {
		deposit, err := proto.Marshal(payload.Deposit)
		if err != nil {
			return err
		}
		ctx.EmitTopics(deposit, mainnetDepositEventTopic)
	}

	if checkTxHash {
		if err := saveSeenTxHash(ctx, payload.Deposit.TxHash, payload.Deposit.TokenKind); err != nil {
			return err
		}
	}

	return nil
}

func (gw *Gateway) GetState(ctx contract.StaticContext, req *GatewayStateRequest) (*GatewayStateResponse, error) {
	state, err := loadState(ctx)
	if err != nil {
		return nil, err
	}
	return &GatewayStateResponse{State: state}, nil
}

// WithdrawToken will attempt to transfer an ERC20/ERC721/X token to the Gateway contract,
// if the transfer is successful the contract will create a receipt than can be used by the
// depositor to reclaim ownership of the token through the Mainnet Gateway contract.
// NOTE: Currently an entity must complete each withdrawal by reclaiming ownership on Mainnet
//       before it can make another one withdrawal (even if the tokens originate from different
//       contracts).
func (gw *Gateway) WithdrawToken(ctx contract.Context, req *WithdrawTokenRequest) error {
	if req.TokenContract == nil {
		return ErrInvalidRequest
	}
	switch req.TokenKind {
	case TokenKind_ERC721:
		// assume TokenID == nil means TokenID == 0
	case TokenKind_ERC721X, TokenKind_ERC20, TokenKind_TRX, TokenKind_TRC20:
		if req.TokenAmount == nil {
			return ErrInvalidRequest
		}
	default:
		return ErrInvalidRequest
	}

	ownerAddr := ctx.Message().Sender
	account, err := loadLocalAccount(ctx, ownerAddr)
	if err != nil {
		emitWithdrawTokenError(ctx, err.Error(), req)
		return err
	}

	if account.WithdrawalReceipt != nil {
		emitWithdrawTokenError(ctx, ErrPendingWithdrawalExists.Error(), req)
		return ErrPendingWithdrawalExists
	}

	ownerEthAddr := loom.Address{}
	if req.Recipient != nil {
		ownerEthAddr = loom.UnmarshalAddressPB(req.Recipient)
	} else {
		mapperAddr, err := ctx.Resolve("addressmapper")
		if err != nil {
			emitWithdrawTokenError(ctx, err.Error(), req)
			return err
		}

		ownerEthAddr, err = resolveToEthAddr(ctx, mapperAddr, ownerAddr)
		if err != nil {
			emitWithdrawTokenError(ctx, err.Error(), req)
			return err
		}
	}

	foreignAccount, err := loadForeignAccount(ctx, ownerEthAddr)
	if err != nil {
		emitWithdrawTokenError(ctx, err.Error(), req)
		return err
	}

	if foreignAccount.CurrentWithdrawer != nil {
		ctx.Logger().Error(ErrPendingWithdrawalExists.Error(), "from", ownerAddr, "to", ownerEthAddr)
		emitWithdrawTokenError(ctx, ErrPendingWithdrawalExists.Error(), req)
		return ErrPendingWithdrawalExists
	}

	tokenAddr := loom.UnmarshalAddressPB(req.TokenContract)
	tokenEthAddr, err := resolveToForeignContractAddr(ctx, tokenAddr)
	if err != nil {
		emitWithdrawTokenError(ctx, err.Error(), req)
		return err
	}

	tokenID := big.NewInt(0)
	if req.TokenID != nil {
		tokenID = req.TokenID.Value.Int
	}

	tokenAmount := big.NewInt(0)
	if req.TokenAmount != nil {
		tokenAmount = req.TokenAmount.Value.Int
	}

	// The entity wishing to make the withdrawal must first grant approval to the Gateway contract
	// to transfer the token, otherwise this will fail...
	switch req.TokenKind {
	case TokenKind_ERC721:
		erc721 := newERC721Context(ctx, tokenAddr)
		if err = erc721.safeTransferFrom(ownerAddr, ctx.ContractAddress(), tokenID); err != nil {
			emitWithdrawTokenError(ctx, err.Error(), req)
			return err
		}
		ctx.Logger().Info("WithdrawERC721", "owner", ownerEthAddr, "token", tokenEthAddr)

	case TokenKind_ERC721X:
		erc721x := newERC721XContext(ctx, tokenAddr)
		if err = erc721x.safeTransferFrom(ownerAddr, ctx.ContractAddress(), tokenID, tokenAmount); err != nil {
			emitWithdrawTokenError(ctx, err.Error(), req)
			return err
		}
		ctx.Logger().Info("WithdrawERC721X", "owner", ownerEthAddr, "token", tokenEthAddr)

	case TokenKind_ERC20, TokenKind_TRC20, TokenKind_TRX:
		erc20 := newERC20Context(ctx, tokenAddr)
		if err := erc20.transferFrom(ownerAddr, ctx.ContractAddress(), tokenAmount); err != nil {
			emitWithdrawTokenError(ctx, err.Error(), req)
			return err
		}
		ctx.Logger().Info("WithdrawToken", "kind", req.TokenKind, "owner", ownerEthAddr, "token", tokenEthAddr)
	}

	account.WithdrawalReceipt = &WithdrawalReceipt{
		TokenOwner:      ownerEthAddr.MarshalPB(),
		TokenContract:   tokenEthAddr.MarshalPB(),
		TokenKind:       req.TokenKind,
		TokenID:         req.TokenID,
		TokenAmount:     req.TokenAmount,
		WithdrawalNonce: foreignAccount.WithdrawalNonce,
	}
	foreignAccount.CurrentWithdrawer = ownerAddr.MarshalPB()

	event, err := proto.Marshal(account.WithdrawalReceipt)
	if err != nil {
		return err
	}
	ctx.EmitTopics(event, withdrawTokenTopic)

	if err := saveForeignAccount(ctx, foreignAccount); err != nil {
		emitWithdrawTokenError(ctx, err.Error(), req)
		return err
	}

	if err := saveLocalAccount(ctx, account); err != nil {
		emitWithdrawTokenError(ctx, err.Error(), req)
		return err
	}

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	if err := addTokenWithdrawer(ctx, state, ownerAddr); err != nil {
		emitWithdrawTokenError(ctx, err.Error(), req)
		return err
	}

	return saveState(ctx, state)
}

// WithdrawETH will attempt to transfer ETH to the Gateway contract,
// if it's successful it will store a receipt than can be used by the depositor to reclaim ownership
// of the ETH through the Mainnet Gateway contract.
// NOTE: Currently an entity must complete each withdrawal by reclaiming ownership on Mainnet
//       before it can make another withdrawal (even if the tokens/ETH originate from different
//       ERC20 or ERC721 contracts).
func (gw *Gateway) WithdrawETH(ctx contract.Context, req *WithdrawETHRequest) error {
	if req.Amount == nil || req.MainnetGateway == nil {
		return ErrInvalidRequest
	}

	if gw.Type != EthereumGateway {
		return ErrInvalidRequest
	}

	ownerAddr := ctx.Message().Sender
	account, err := loadLocalAccount(ctx, ownerAddr)
	if err != nil {
		emitWithdrawETHError(ctx, err.Error(), req)
		return err
	}

	if account.WithdrawalReceipt != nil {
		emitWithdrawETHError(ctx, ErrPendingWithdrawalExists.Error(), req)
		return ErrPendingWithdrawalExists
	}

	ownerEthAddr := loom.Address{}
	if req.Recipient != nil {
		ownerEthAddr = loom.UnmarshalAddressPB(req.Recipient)
	} else {
		mapperAddr, err := ctx.Resolve("addressmapper")
		if err != nil {
			emitWithdrawETHError(ctx, err.Error(), req)
			return err
		}

		ownerEthAddr, err = resolveToEthAddr(ctx, mapperAddr, ownerAddr)
		if err != nil {
			emitWithdrawETHError(ctx, err.Error(), req)
			return err
		}
	}

	foreignAccount, err := loadForeignAccount(ctx, ownerEthAddr)
	if err != nil {
		emitWithdrawETHError(ctx, err.Error(), req)
		return err
	}

	if foreignAccount.CurrentWithdrawer != nil {
		ctx.Logger().Error(ErrPendingWithdrawalExists.Error(), "from", ownerAddr, "to", ownerEthAddr)
		emitWithdrawETHError(ctx, ErrPendingWithdrawalExists.Error(), req)
		return ErrPendingWithdrawalExists
	}

	// The entity wishing to make the withdrawal must first grant approval to the Gateway contract
	// to transfer the tokens, otherwise this will fail...
	eth := newETHContext(ctx)
	if err := eth.transferFrom(ownerAddr, ctx.ContractAddress(), req.Amount.Value.Int); err != nil {
		return err
	}

	account.WithdrawalReceipt = &WithdrawalReceipt{
		TokenOwner:      ownerEthAddr.MarshalPB(),
		TokenContract:   req.MainnetGateway,
		TokenKind:       TokenKind_ETH,
		TokenAmount:     req.Amount,
		WithdrawalNonce: foreignAccount.WithdrawalNonce,
	}
	foreignAccount.CurrentWithdrawer = ownerAddr.MarshalPB()

	event, err := proto.Marshal(account.WithdrawalReceipt)
	if err != nil {
		return err
	}
	ctx.EmitTopics(event, withdrawETHTopic)

	if err := saveForeignAccount(ctx, foreignAccount); err != nil {
		emitWithdrawETHError(ctx, err.Error(), req)
		return err
	}

	if err := saveLocalAccount(ctx, account); err != nil {
		emitWithdrawETHError(ctx, err.Error(), req)
		return err
	}

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	if err := addTokenWithdrawer(ctx, state, ownerAddr); err != nil {
		emitWithdrawETHError(ctx, err.Error(), req)
		return err
	}

	return saveState(ctx, state)
}

// SubmitLoomCoinDepositTxHash is called by User to add txhash, which will be used to submit deposit event
// User's account need to be mapped to an eth address for this to work.
func (gw *Gateway) SubmitDepositTxHash(ctx contract.Context, req *SubmitDepositTxHashRequest) error {
	if req.TxHash == nil {
		return ErrInvalidRequest
	}

	addressMapperAddress, err := ctx.Resolve("addressmapper")
	if err != nil {
		return err
	}

	ownerAddr := ctx.Message().Sender
	ownerEthAddr, err := resolveToEthAddr(ctx, addressMapperAddress, ownerAddr)
	if err != nil {
		return err
	}

	extState, err := loadExtendedState(ctx)
	if err != nil {
		return err
	}

	if err := addDepositTxHashSubmitter(ctx, extState, ownerEthAddr); err != nil {
		return err
	}

	if err := saveExtendedState(ctx, extState); err != nil {
		return err
	}

	return saveDepositTxHash(ctx, ownerEthAddr, req.TxHash)
}

// ClearInvalidLoomCoinDepositTxHash is oracle only method called by oracle to clear
// invalid tx hashes submitted by users
func (gw *Gateway) ClearInvalidDepositTxHash(ctx contract.Context, req *ClearInvalidDepositTxHashRequest) error {
	if ok, _ := ctx.HasPermission(clearInvalidTxHashesPerm, []string{oracleRole}); !ok {
		return ErrNotAuthorized
	}

	if req.TxHashes == nil {
		return ErrInvalidRequest
	}

	extState, err := loadExtendedState(ctx)
	if err != nil {
		return err
	}

	if err := removeDepositTxHashes(ctx, extState, req.TxHashes); err != nil {
		return err
	}

	return saveExtendedState(ctx, extState)
}

// UnprocessedDepositTxHashes returns tx hashes that havent been processed yet by oracle
func (gw *Gateway) UnprocessedDepositTxHashes(ctx contract.StaticContext, req *UnprocessedDepositTxHashesRequest) (*UnprocessedDepositTxHashesResponse, error) {
	extState, err := loadExtendedState(ctx)
	if err != nil {
		return nil, err
	}

	unprocessedHashes, err := getDepositTxHashes(ctx, extState)
	if err != nil {
		return nil, err
	}

	return &UnprocessedDepositTxHashesResponse{TxHashes: unprocessedHashes}, nil
}

func clearDepositTxHashIfExists(ctx contract.Context, ownerAddress loom.Address) error {
	extState, err := loadExtendedState(ctx)
	if err != nil {
		return err
	}

	if err := removeDepositTxHashSubmitter(ctx, extState, ownerAddress); err != nil {
		if err != ErrNoUnprocessedTxHashExists {
			return err
		}
	}

	if err := saveExtendedState(ctx, extState); err != nil {
		return err
	}

	return nil
}

// WithdrawLoomCoin will attempt to transfer Loomcoin to the Gateway contract,
// if it's successful it will store a receipt than can be used by the depositor to reclaim ownership
// of the Loomcoin through the Mainnet Gateway contract.
// NOTE: Currently an entity must complete each withdrawal by reclaiming ownership on Mainnet
//       before it can make another withdrawal (even if the tokens/ETH/Loom originate from different
//       ERC20 or ERC721 contracts).
func (gw *Gateway) WithdrawLoomCoin(ctx contract.Context, req *WithdrawLoomCoinRequest) error {
	if req.Amount == nil || req.TokenContract == nil {
		return ErrInvalidRequest
	}

	if gw.Type != LoomCoinGateway {
		return ErrInvalidRequest
	}

	ownerAddr := ctx.Message().Sender
	account, err := loadLocalAccount(ctx, ownerAddr)
	if err != nil {
		emitWithdrawLoomCoinError(ctx, err.Error(), req)
		return err
	}

	if account.WithdrawalReceipt != nil {
		emitWithdrawLoomCoinError(ctx, ErrPendingWithdrawalExists.Error(), req)
		return ErrPendingWithdrawalExists
	}

	ownerEthAddr := loom.Address{}
	if req.Recipient != nil {
		ownerEthAddr = loom.UnmarshalAddressPB(req.Recipient)
	} else {
		mapperAddr, err := ctx.Resolve("addressmapper")
		if err != nil {
			emitWithdrawLoomCoinError(ctx, err.Error(), req)
			return err
		}

		ownerEthAddr, err = resolveToEthAddr(ctx, mapperAddr, ownerAddr)
		if err != nil {
			emitWithdrawLoomCoinError(ctx, err.Error(), req)
			return err
		}
	}

	foreignAccount, err := loadForeignAccount(ctx, ownerEthAddr)
	if err != nil {
		emitWithdrawLoomCoinError(ctx, err.Error(), req)
		return err
	}

	if foreignAccount.CurrentWithdrawer != nil {
		emitWithdrawLoomCoinError(ctx, ErrPendingWithdrawalExists.Error(), req)
		ctx.Logger().Error(ErrPendingWithdrawalExists.Error(), "from", ownerAddr, "to", ownerEthAddr)
		return ErrPendingWithdrawalExists
	}

	// Burning the coin from dappchain to keep amount of coin consistent between two chains
	coin := newCoinContext(ctx)
	if err := coin.burn(ownerAddr, req.Amount.Value.Int); err != nil {
		return err
	}

	ctx.Logger().Info("WithdrawLoomCoin", "owner", ownerEthAddr, "token", req.TokenContract)

	account.WithdrawalReceipt = &WithdrawalReceipt{
		TokenOwner:      ownerEthAddr.MarshalPB(),
		TokenContract:   req.TokenContract,
		TokenKind:       TokenKind_LoomCoin,
		TokenAmount:     req.Amount,
		WithdrawalNonce: foreignAccount.WithdrawalNonce,
	}
	foreignAccount.CurrentWithdrawer = ownerAddr.MarshalPB()

	event, err := proto.Marshal(account.WithdrawalReceipt)
	if err != nil {
		return err
	}
	ctx.EmitTopics(event, withdrawLoomCoinTopic)

	if err := saveForeignAccount(ctx, foreignAccount); err != nil {
		emitWithdrawLoomCoinError(ctx, err.Error(), req)
		return err
	}

	if err := saveLocalAccount(ctx, account); err != nil {
		emitWithdrawLoomCoinError(ctx, err.Error(), req)
		return err
	}

	state, err := loadState(ctx)
	if err != nil {
		emitWithdrawLoomCoinError(ctx, err.Error(), req)
		return err
	}

	if err := addTokenWithdrawer(ctx, state, ownerAddr); err != nil {
		emitWithdrawLoomCoinError(ctx, err.Error(), req)
		return err
	}

	return saveState(ctx, state)
}

// WithdrawalReceipt will return the receipt generated by the last successful call to WithdrawERC721.
// The receipt can be used to reclaim ownership of the token through the Mainnet Gateway.
func (gw *Gateway) WithdrawalReceipt(ctx contract.StaticContext, req *WithdrawalReceiptRequest) (*WithdrawalReceiptResponse, error) {
	// assume the caller is the owner if the request doesn't specify one
	owner := ctx.Message().Sender
	if req.Owner != nil {
		owner = loom.UnmarshalAddressPB(req.Owner)
	}
	if owner.IsEmpty() {
		return nil, errors.New("no owner specified")
	}
	account, err := loadLocalAccount(ctx, owner)
	if err != nil {
		return nil, err
	}
	return &WithdrawalReceiptResponse{Receipt: account.WithdrawalReceipt}, nil
}

// ConfirmWithdrawalReceipt will attempt to set the Oracle signature on an existing withdrawal
// receipt. This method is only allowed to be invoked by Oracles with withdrawal signing permission,
// and only one Oracle will ever be able to successfully set the signature for any particular
// receipt, all other attempts will error out.
func (gw *Gateway) ConfirmWithdrawalReceipt(ctx contract.Context, req *ConfirmWithdrawalReceiptRequest) error {
	if ok, _ := ctx.HasPermission(signWithdrawalsPerm, []string{oracleRole}); !ok {
		return ErrNotAuthorized
	}

	return gw.doConfirmWithdrawalReceipt(ctx, req)
}

// (added as a separate method to not break consensus - backwards compatibility)
// ConfirmWithdrawalReceiptV2 will attempt to set the Oracle signature on an existing withdrawal
// receipt. This method is allowed to be invoked by any Validator ,
// and only one Validator will ever be able to successfully set the signature for any particular
// receipt, all other attempts will error out.
func (gw *Gateway) ConfirmWithdrawalReceiptV2(ctx contract.Context, req *ConfirmWithdrawalReceiptRequestV2) error {
	valAddresses, powers, clusterStake, err := getCurrentValidators(ctx)
	if err != nil {
		return err
	}

	sender := ctx.Message().Sender
	var found bool = false
	for _, v := range valAddresses {
		if sender.Compare(loom.UnmarshalAddressPB(v)) == 0 {
			found = true
			break
		}
	}
	if !found {
		return ErrNotAuthorized
	}

	validatorsAuthConfig := &ValidatorAuthConfig{}
	if err := ctx.Get(validatorAuthConfigKey, validatorsAuthConfig); err != nil {
		return err
	}

	if req.TokenOwner == nil || req.OracleSignature == nil {
		return ErrInvalidRequest
	}

	ownerAddr := loom.UnmarshalAddressPB(req.TokenOwner)
	ownerAccount, err := loadLocalAccount(ctx, ownerAddr)
	if err != nil {
		return err
	}

	if ownerAccount.WithdrawalReceipt == nil {
		return ErrMissingWithdrawalReceipt
	} else if ownerAccount.WithdrawalReceipt.OracleSignature != nil {
		return ErrWithdrawalReceiptSigned
	}

	hash := client.ToEthereumSignedMessage(gw.calculateHashFromReceiptV2(req.MainnetGateway, ownerAccount.WithdrawalReceipt))

	switch validatorsAuthConfig.AuthStrategy {
	case tgtypes.ValidatorAuthStrategy_USE_TRUSTED_VALIDATORS:
		// Convert array of validator to array of address, try to resolve via address mapper
		// Feed the mapped addresses to ParseSigs
		ethAddresses, err := getMappedEthAddress(ctx, validatorsAuthConfig.TrustedValidators.Validators)
		if err != nil {
			return err
		}

		_, _, _, valIndexes, err := client.ParseSigs(req.OracleSignature, hash, ethAddresses)
		if err != nil {
			return err
		}

		// Reject if not all trusted validators signed
		if len(valIndexes) != len(validatorsAuthConfig.TrustedValidators.Validators) {
			return ErrNotEnoughSignatures
		}
		break
	case tgtypes.ValidatorAuthStrategy_USE_DPOS_VALIDATORS:
		requiredStakeForMaj23 := big.NewInt(0)
		requiredStakeForMaj23.Mul(clusterStake, big.NewInt(2))
		requiredStakeForMaj23.Div(requiredStakeForMaj23, big.NewInt(3))
		requiredStakeForMaj23.Add(requiredStakeForMaj23, big.NewInt(1))

		ethAddresses, err := getMappedEthAddress(ctx, valAddresses)
		if err != nil {
			return err
		}

		_, _, _, valIndexes, err := client.ParseSigs(req.OracleSignature, hash, ethAddresses)
		if err != nil {
			return err
		}

		signedValStakes := big.NewInt(0)

		// Map to store, whether we already seen this validator
		seenVal := make(map[int]bool)

		for i, valIndex := range valIndexes {
			valIndexInt := int(valIndex.Int64())

			// Prevents double counting distribution total
			if seenVal[valIndexInt] {
				continue
			}
			seenVal[valIndexInt] = true

			signedValStakes.Add(signedValStakes, powers[i])
		}

		if signedValStakes.Cmp(requiredStakeForMaj23) < 0 {
			return ErrNotAuthorized
		}

		break
	}

	return gw.doConfirmWithdrawalReceiptV2(ctx, ownerAccount, req.OracleSignature)
}

func (gw *Gateway) calculateHashFromReceipt(mainnetGatewayAddr *types.Address, receipt *tgtypes.TransferGatewayWithdrawalReceipt) []byte {
	safeTokenID := big.NewInt(0)
	if receipt.TokenID != nil {
		safeTokenID = receipt.TokenID.Value.Int
	}

	safeAmount := big.NewInt(0)
	if receipt.TokenAmount != nil {
		safeAmount = receipt.TokenAmount.Value.Int
	}

	mainnetGatewayEthAddress := common.BytesToAddress(mainnetGatewayAddr.Local)

	hash := client.WithdrawalHash(
		common.BytesToAddress(receipt.TokenOwner.Local),
		common.BytesToAddress(receipt.TokenContract.Local),
		mainnetGatewayEthAddress,
		receipt.TokenKind,
		safeTokenID,
		safeAmount,
		big.NewInt(int64(receipt.WithdrawalNonce)),
		false,
	)

	return hash
}

func (gw *Gateway) calculateHashFromReceiptV2(mainnetGatewayAddr *types.Address, receipt *tgtypes.TransferGatewayWithdrawalReceipt) []byte {
	safeTokenID := big.NewInt(0)
	if receipt.TokenID != nil {
		safeTokenID = receipt.TokenID.Value.Int
	}

	safeAmount := big.NewInt(0)
	if receipt.TokenAmount != nil {
		safeAmount = receipt.TokenAmount.Value.Int
	}

	mainnetGatewayEthAddress := common.BytesToAddress(mainnetGatewayAddr.Local)

	hash := client.WithdrawalHash(
		common.BytesToAddress(receipt.TokenOwner.Local),
		common.BytesToAddress(receipt.TokenContract.Local),
		mainnetGatewayEthAddress,
		receipt.TokenKind,
		safeTokenID,
		safeAmount,
		big.NewInt(int64(receipt.WithdrawalNonce)),
		true,
	)

	return hash
}

func (gw *Gateway) doConfirmWithdrawalReceiptV2(ctx contract.Context, account *LocalAccount, oracleSignature []byte) error {
	account.WithdrawalReceipt.OracleSignature = oracleSignature

	if err := saveLocalAccount(ctx, account); err != nil {
		return err
	}

	wr := account.WithdrawalReceipt
	payload, err := proto.Marshal(&TokenWithdrawalSigned{
		TokenOwner:    wr.TokenOwner,
		TokenContract: wr.TokenContract,
		TokenKind:     wr.TokenKind,
		TokenID:       wr.TokenID,
		TokenAmount:   wr.TokenAmount,
		Sig:           wr.OracleSignature,
	})
	if err != nil {
		return err
	}
	// TODO: Re-enable the second topic when we fix an issue with subscribers receiving the same
	//       event twice (or more depending on the number of topics).
	ctx.EmitTopics(payload, tokenWithdrawalSignedEventTopic /*, fmt.Sprintf("contract:%v", wr.TokenContract)*/)
	return nil
}

func (gw *Gateway) doConfirmWithdrawalReceipt(ctx contract.Context, req *ConfirmWithdrawalReceiptRequest) error {

	if req.TokenOwner == nil || req.OracleSignature == nil {
		return ErrInvalidRequest
	}

	ownerAddr := loom.UnmarshalAddressPB(req.TokenOwner)
	account, err := loadLocalAccount(ctx, ownerAddr)
	if err != nil {
		return err
	}

	if account.WithdrawalReceipt == nil {
		return ErrMissingWithdrawalReceipt
	} else if account.WithdrawalReceipt.OracleSignature != nil {
		return ErrWithdrawalReceiptSigned
	}

	account.WithdrawalReceipt.OracleSignature = req.OracleSignature

	if err := saveLocalAccount(ctx, account); err != nil {
		return err
	}

	wr := account.WithdrawalReceipt
	payload, err := proto.Marshal(&TokenWithdrawalSigned{
		TokenOwner:    wr.TokenOwner,
		TokenContract: wr.TokenContract,
		TokenKind:     wr.TokenKind,
		TokenID:       wr.TokenID,
		TokenAmount:   wr.TokenAmount,
		Sig:           wr.OracleSignature,
	})
	if err != nil {
		return err
	}
	// TODO: Re-enable the second topic when we fix an issue with subscribers receiving the same
	//       event twice (or more depending on the number of topics).
	ctx.EmitTopics(payload, tokenWithdrawalSignedEventTopic /*, fmt.Sprintf("contract:%v", wr.TokenContract)*/)
	return nil
}

// PendingWithdrawals will return the token owner & withdrawal hash for all pending withdrawals.
// The Oracle will call this method periodically and sign all the retrieved hashes.
func (gw *Gateway) PendingWithdrawalsV2(ctx contract.StaticContext, req *PendingWithdrawalsRequest) (*PendingWithdrawalsResponse, error) {
	if req.MainnetGateway == nil {
		return nil, ErrInvalidRequest
	}

	state, err := loadState(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]*PendingWithdrawalSummary, 0, len(state.TokenWithdrawers))
	for _, ownerAddrPB := range state.TokenWithdrawers {
		ownerAddr := loom.UnmarshalAddressPB(ownerAddrPB)
		account, err := loadLocalAccount(ctx, ownerAddr)
		if err != nil {
			return nil, err
		}
		receipt := account.WithdrawalReceipt

		if receipt == nil {
			return nil, ErrMissingWithdrawalReceipt
		}
		// If the receipt is already signed, skip it
		if receipt.OracleSignature != nil {
			continue
		}

		hash := gw.calculateHashFromReceiptV2(req.MainnetGateway, receipt)

		summaries = append(summaries, &PendingWithdrawalSummary{
			TokenOwner: ownerAddrPB,
			Hash:       hash,
		})
	}

	// TODO: should probably enforce an upper bound on the response size
	return &PendingWithdrawalsResponse{Withdrawals: summaries}, nil
}

// PendingWithdrawals will return the token owner & withdrawal hash for all pending withdrawals.
// The Oracle will call this method periodically and sign all the retrieved hashes.
func (gw *Gateway) PendingWithdrawals(ctx contract.StaticContext, req *PendingWithdrawalsRequest) (*PendingWithdrawalsResponse, error) {
	if req.MainnetGateway == nil {
		return nil, ErrInvalidRequest
	}

	state, err := loadState(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]*PendingWithdrawalSummary, 0, len(state.TokenWithdrawers))
	for _, ownerAddrPB := range state.TokenWithdrawers {
		ownerAddr := loom.UnmarshalAddressPB(ownerAddrPB)
		account, err := loadLocalAccount(ctx, ownerAddr)
		if err != nil {
			return nil, err
		}
		receipt := account.WithdrawalReceipt

		if receipt == nil {
			return nil, ErrMissingWithdrawalReceipt
		}
		// If the receipt is already signed, skip it
		if receipt.OracleSignature != nil {
			continue
		}

		hash := gw.calculateHashFromReceipt(req.MainnetGateway, receipt)

		summaries = append(summaries, &PendingWithdrawalSummary{
			TokenOwner: ownerAddrPB,
			Hash:       hash,
		})
	}

	// TODO: should probably enforce an upper bound on the response size
	return &PendingWithdrawalsResponse{Withdrawals: summaries}, nil
}

// ReclaimDepositorTokens will attempt to transfer any tokens that the caller may have deposited
// into the Mainnet Gateway but hasn't yet received from the DAppChain Gateway because of a missing
// identity or contract mapping.
func (gw *Gateway) ReclaimDepositorTokens(ctx contract.Context, req *ReclaimDepositorTokensRequest) error {
	// Assume the caller is trying to reclaim their own tokens if depositors are not specified
	if len(req.Depositors) == 0 {
		mapperAddr, err := ctx.Resolve("addressmapper")
		if err != nil {
			return errors.Wrap(err, ErrFailedToReclaimToken.Error())
		}

		ownerAddr, err := resolveToEthAddr(ctx, mapperAddr, ctx.Message().Sender)
		if err != nil {
			emitReclaimError(ctx, "[ReclaimDepositorTokens resolveToEthAddress] "+err.Error(), ownerAddr)
			return errors.Wrap(err, ErrFailedToReclaimToken.Error())
		}

		if err := reclaimDepositorTokens(ctx, ownerAddr); err != nil {
			emitReclaimError(ctx, "[ReclaimDepositorTokens reclaimDepositorTokens] "+err.Error(), ownerAddr)
			return err
		}
		return nil
	}

	// Otherwise only the Gateway owner is allowed to reclaim tokens for depositors
	state, err := loadState(ctx)
	if err != nil {
		return errors.Wrap(err, ErrFailedToReclaimToken.Error())
	}
	if loom.UnmarshalAddressPB(state.Owner).Compare(ctx.Message().Sender) != 0 {
		return ErrNotAuthorized
	}

	for _, depAddr := range req.Depositors {
		ownerAddr := loom.UnmarshalAddressPB(depAddr)
		if err := reclaimDepositorTokens(ctx, ownerAddr); err != nil {
			emitReclaimError(ctx, "[ReclaimDepositorTokens reclaimDepositorTokens] "+err.Error(), ownerAddr)
			ctx.Logger().Error("[Transfer Gateway] failed to reclaim depositor tokens",
				"owner", ownerAddr,
				"err", err,
			)
		}
	}
	return nil
}

func (gw *Gateway) GetUnclaimedTokens(ctx contract.StaticContext, req *GetUnclaimedTokensRequest) (*GetUnclaimedTokensResponse, error) {
	ownerAddr := loom.UnmarshalAddressPB(req.Owner)
	unclaimedTokens, err := unclaimedTokensByOwner(ctx, ownerAddr)
	if err != nil {
		return nil, err
	}

	return &GetUnclaimedTokensResponse{
		UnclaimedTokens: unclaimedTokens,
	}, nil
}

func (gw *Gateway) GetUnclaimedContractTokens(
	ctx contract.StaticContext, req *GetUnclaimedContractTokensRequest,
) (*GetUnclaimedContractTokensResponse, error) {
	if req.TokenAddress == nil {
		return nil, ErrInvalidRequest
	}
	ethTokenAddress := loom.UnmarshalAddressPB(req.TokenAddress)
	depositors, err := unclaimedTokenDepositorsByContract(ctx, ethTokenAddress)
	if err != nil {
		return nil, err
	}
	unclaimedAmount := loom.NewBigUIntFromInt(0)
	var unclaimedToken UnclaimedToken
	amount := loom.NewBigUIntFromInt(0)
	for _, address := range depositors {
		tokenKey := unclaimedTokenKey(address, ethTokenAddress)
		err := ctx.Get(tokenKey, &unclaimedToken)
		if err != nil && err != contract.ErrNotFound {
			return nil, errors.Wrapf(err, "failed to load unclaimed token for %v", address)
		}
		switch unclaimedToken.TokenKind {
		case TokenKind_ERC721:
			unclaimedAmount = unclaimedAmount.Add(unclaimedAmount, loom.NewBigUIntFromInt(int64(len(unclaimedToken.Amounts))))
		case TokenKind_ERC721X:
			for _, a := range unclaimedToken.Amounts {
				unclaimedAmount = unclaimedAmount.Add(unclaimedAmount, loom.NewBigUInt(a.TokenAmount.Value.Int))

			}
		case TokenKind_ERC20, TokenKind_ETH, TokenKind_LoomCoin:
			if len(unclaimedToken.Amounts) == 1 {
				amount = loom.NewBigUInt(unclaimedToken.Amounts[0].TokenAmount.Value.Int)
			}
			unclaimedAmount = unclaimedAmount.Add(unclaimedAmount, amount)

		}
	}
	return &GetUnclaimedContractTokensResponse{
		UnclaimedAmount: &types.BigUInt{Value: *unclaimedAmount},
	}, nil
}

// ReclaimContractTokens will attempt to transfer tokens that originated from the specified Mainnet
// contract, and that have been deposited to the Mainnet Gateway, but haven't yet been received by
// the depositors on the DAppChain because of a missing identity or contract mapping. This function
// can only be called by the creator of the specified token contract, or the Gateway contract owner.
func (gw *Gateway) ReclaimContractTokens(ctx contract.Context, req *ReclaimContractTokensRequest) error {
	if req.TokenContract == nil {
		return ErrInvalidRequest
	}

	foreignContractAddr := loom.UnmarshalAddressPB(req.TokenContract)
	localContractAddr, err := resolveToLocalContractAddr(ctx, foreignContractAddr)
	if err != nil {
		emitReclaimError(ctx, "[ReclaimContractTokens resolveToLocalContractAddr] "+err.Error(), localContractAddr)
		return errors.Wrap(err, ErrFailedToReclaimToken.Error())
	}

	cr, err := ctx.ContractRecord(localContractAddr)
	if err != nil {
		emitReclaimError(ctx, "[ReclaimContractTokens ContractRecord] "+err.Error(), localContractAddr)
		return errors.Wrap(err, ErrFailedToReclaimToken.Error())
	}

	callerAddr := ctx.Message().Sender
	if cr.CreatorAddress.Compare(callerAddr) != 0 {
		state, err := loadState(ctx)
		if err != nil {
			return errors.Wrap(err, ErrFailedToReclaimToken.Error())
		}
		if loom.UnmarshalAddressPB(state.Owner).Compare(callerAddr) != 0 {
			emitReclaimError(ctx, "[ReclaimContractTokens CompareAddress] "+ErrNotAuthorized.Error(), localContractAddr)
			return ErrNotAuthorized
		}
	}

	for _, entry := range ctx.Range(unclaimedTokenDepositorsRangePrefix(foreignContractAddr)) {
		var addr types.Address
		if err := proto.Unmarshal(entry.Value, &addr); err != nil {
			emitReclaimError(ctx, "[ReclaimContractTokens ContractRecord] "+err.Error(), foreignContractAddr)
			ctx.Logger().Error(
				"[Transfer Gateway] ReclaimContractTokens failed to unmarshal depositor address",
				"token", foreignContractAddr,
				"err", err,
			)
			continue
		}

		ownerAddr := loom.UnmarshalAddressPB(&addr)
		tokenKey := unclaimedTokenKey(ownerAddr, foreignContractAddr)
		var unclaimedToken UnclaimedToken
		if err := ctx.Get(tokenKey, &unclaimedToken); err != nil {
			emitReclaimError(ctx, "[ReclaimContractTokens failed to load unclaimed tokens] "+err.Error(), ownerAddr)
			ctx.Logger().Error("[Transfer Gateway] failed to load unclaimed tokens",
				"owner", ownerAddr,
				"token", foreignContractAddr,
				"err", err,
			)
		}
		if err := reclaimDepositorTokensForContract(ctx, ownerAddr, foreignContractAddr, &unclaimedToken); err != nil {
			emitReclaimError(ctx, "[ReclaimContractTokens failed to reclaim depositor tokens] "+err.Error(), ownerAddr)
			ctx.Logger().Error("[Transfer Gateway] failed to reclaim depositor tokens",
				"owner", ownerAddr,
				"token", foreignContractAddr,
				"err", err,
			)
		}
	}

	return nil
}

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
	case TokenKind_ERC721X, TokenKind_ERC20, TokenKind_ETH, TokenKind_LoomCoin, TokenKind_TRX, TokenKind_TRC20:
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

	case TokenKind_ERC20, TokenKind_TRC20, TokenKind_TRX:
		// TRC20 and TRX are also others ERC20s compatible
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
	var tokenAddr loom.Address
	switch deposit.TokenKind {
	case TokenKind_TRC20, TokenKind_TRX:
		tokenAddr = loom.RootAddress("tron")
	default:
		tokenAddr = loom.RootAddress("eth")
	}

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

	case TokenKind_ERC20, TokenKind_ETH, TokenKind_LoomCoin, TokenKind_TRX, TokenKind_TRC20:
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

	// emit storeUnclaimedToken
	unclaimTokenBytes, err := proto.Marshal(&unclaimedToken)
	if err != nil {
		return err
	}
	ctx.EmitTopics(unclaimTokenBytes, storeUnclaimedTokenTopic)

	return ctx.Set(tokenKey, &unclaimedToken)
}

// When a token is withdrawn from the Mainnet Gateway find the corresponding withdrawal receipt
// and remove it from the owner's account, once the receipt is removed the owner will be able to
// initiate another withdrawal to Mainnet.
func completeTokenWithdraw(ctx contract.Context, state *GatewayState, withdrawal *MainnetTokenWithdrawn) error {
	if withdrawal.TokenOwner == nil {
		return ErrInvalidRequest
	}

	// non-native coins must have token contract address
	if (withdrawal.TokenKind != TokenKind_ETH) && (withdrawal.TokenContract == nil) {
		return ErrInvalidRequest
	}

	switch withdrawal.TokenKind {
	case TokenKind_ERC721:
		// assume TokenID == nil means TokenID == 0
	case TokenKind_ERC721X, TokenKind_ERC20, TokenKind_ETH, TokenKind_LoomCoin, TokenKind_TRX, TokenKind_TRC20:
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

func loadDepositTxHash(ctx contract.StaticContext, owner loom.Address) ([]byte, error) {
	tgTxHash := &TransferGatewayTxHash{}
	err := ctx.Get(DepositTxHashKey(owner), tgTxHash)
	if err != nil {
		if err == contract.ErrNotFound {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to load deposit tx hash")
	}

	return tgTxHash.TxHash, nil
}

func deleteDepositTxHash(ctx contract.Context, owner loom.Address) {
	ctx.Delete(DepositTxHashKey(owner))
}

func saveDepositTxHash(ctx contract.Context, owner loom.Address, txHash []byte) error {
	tgTxHash := &TransferGatewayTxHash{
		TxHash: txHash,
	}
	return ctx.Set(DepositTxHashKey(owner), tgTxHash)
}

func loadExtendedState(ctx contract.StaticContext) (*ExtendedState, error) {
	state := &ExtendedState{}
	err := ctx.Get(extendedStateKey, state)
	if err != nil && err != contract.ErrNotFound {
		return nil, err
	}
	return state, nil
}

func saveExtendedState(ctx contract.Context, extState *ExtendedState) error {
	return ctx.Set(extendedStateKey, extState)
}

func addDepositTxHashSubmitter(ctx contract.StaticContext, extState *ExtendedState, owner loom.Address) error {
	ownerAddrPB := owner.MarshalPB()
	for _, addr := range extState.DepositTxHashSubmitters {
		if ownerAddrPB.ChainId == addr.ChainId && ownerAddrPB.Local.Compare(addr.Local) == 0 {
			return ErrUnprocessedTxHashAlreadyExists
		}
	}
	extState.DepositTxHashSubmitters = append(extState.DepositTxHashSubmitters, ownerAddrPB)
	return nil
}

func getDepositTxHashes(ctx contract.StaticContext, extState *ExtendedState) ([][]byte, error) {
	txHashes := make([][]byte, len(extState.DepositTxHashSubmitters))
	for i, txHashSubmitter := range extState.DepositTxHashSubmitters {
		hash, err := loadDepositTxHash(ctx, loom.UnmarshalAddressPB(txHashSubmitter))
		if err != nil {
			return nil, errors.Wrap(err, "error while loading loomcoin deposit tx hash")
		}
		if hash == nil {
			return nil, ErrNoUnprocessedTxHashExists
		}
		txHashes[i] = hash
	}

	return txHashes, nil
}

func removeDepositTxHashes(ctx contract.Context, extState *ExtendedState, toBeDeletedTxHashes [][]byte) error {
	// Temporarily store it in map for faster lookup
	toBeDeletedTxHashMap := make(map[string]bool)
	for _, toBeDeletedTxHash := range toBeDeletedTxHashes {
		toBeDeletedTxHashMap[hex.EncodeToString(toBeDeletedTxHash)] = true
	}

	survivedTxHashSubmitters := make([]*types.Address, 0, len(extState.DepositTxHashSubmitters))

	for _, txHashSubmitter := range extState.DepositTxHashSubmitters {
		hash, err := loadDepositTxHash(ctx, loom.UnmarshalAddressPB(txHashSubmitter))
		if err != nil {
			return errors.Wrap(err, "error while loading loomcoin deposit tx hash")
		}
		if hash == nil {
			return ErrNoUnprocessedTxHashExists
		}

		if _, ok := toBeDeletedTxHashMap[hex.EncodeToString(hash)]; ok {
			deleteDepositTxHash(ctx, loom.UnmarshalAddressPB(txHashSubmitter))
		} else {
			survivedTxHashSubmitters = append(survivedTxHashSubmitters, txHashSubmitter)
		}
	}

	extState.DepositTxHashSubmitters = survivedTxHashSubmitters
	return nil
}

func removeDepositTxHashSubmitter(ctx contract.StaticContext, extState *ExtendedState, owner loom.Address) error {
	ownerAddrPB := owner.MarshalPB()
	for i, addr := range extState.DepositTxHashSubmitters {
		if ownerAddrPB.ChainId == addr.ChainId && ownerAddrPB.Local.Compare(addr.Local) == 0 {
			// TODO: keep the list sorted
			extState.DepositTxHashSubmitters[i] = extState.DepositTxHashSubmitters[len(extState.DepositTxHashSubmitters)-1]
			extState.DepositTxHashSubmitters = extState.DepositTxHashSubmitters[:len(extState.DepositTxHashSubmitters)-1]
			return nil
		}
	}

	return ErrNoUnprocessedTxHashExists
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
	ctx.GrantPermissionTo(oracleAddr, clearInvalidTxHashesPerm, oracleRole)

	err := ctx.Set(oracleStateKey(oracleAddr), &OracleState{Address: oracleAddr.MarshalPB()})
	if err != nil {
		return errors.Wrap(err, ErrOracleStateSaveFailed.Error())
	}
	return nil
}

var Contract plugin.Contract = contract.MakePluginContract(&Gateway{
	Type: EthereumGateway,
})

var LoomCoinContract plugin.Contract = contract.MakePluginContract(&Gateway{
	Type: LoomCoinGateway,
})

var UnsafeContract plugin.Contract = contract.MakePluginContract(&UnsafeGateway{Gateway{
	Type: EthereumGateway,
}})

var UnsafeLoomCoinContract plugin.Contract = contract.MakePluginContract(&UnsafeGateway{Gateway{
	Type: LoomCoinGateway,
}})

var TronContract plugin.Contract = contract.MakePluginContract(&Gateway{
	Type: TronGateway,
})

var UnsafeTronContract plugin.Contract = contract.MakePluginContract(&UnsafeGateway{Gateway{
	Type: TronGateway,
}})

func emitProcessEventError(ctx contract.Context, errorMessage string, event *MainnetEvent) error {
	eventError, err := proto.Marshal(&MainnetProcessEventError{
		EthBlock:     event.EthBlock,
		Event:        event,
		ErrorMessage: errorMessage,
	})
	if err != nil {
		return err
	}
	ctx.EmitTopics(eventError, mainnetProcessEventErrorTopic)
	return nil
}

func hasSeenTxHash(ctx contract.StaticContext, txHash []byte) bool {
	return ctx.Has(seenTxHashKey(txHash))
}

func saveSeenTxHash(ctx contract.Context, txHash []byte, tokenKind TokenKind) error {
	seenTxHash := MainnetEventTxHashInfo{TokenKind: tokenKind}
	if err := ctx.Set(seenTxHashKey(txHash), &seenTxHash); err != nil {
		return errors.Wrapf(err, "failed to save seen tx hash for %x", txHash)
	}
	return nil
}

func emitReclaimError(ctx contract.Context, errorMessage string, ownerAddress loom.Address) error {
	eventError, err := proto.Marshal(&ReclaimError{
		Owner:        ownerAddress.MarshalPB(),
		ErrorMessage: errorMessage,
	})
	if err != nil {
		return err
	}
	ctx.EmitTopics(eventError, reclaimErrorTopic)
	return nil
}

func emitWithdrawETHError(ctx contract.Context, errorMessage string, request *tgtypes.TransferGatewayWithdrawETHRequest) error {
	withdrawETHError, err := proto.Marshal(&WithdrawETHError{
		WithdrawEthRequest: request,
		ErrorMessage:       errorMessage,
	})
	if err != nil {
		return err
	}
	ctx.EmitTopics(withdrawETHError, withdrawETHErrorTopic)
	return nil
}

func emitWithdrawTokenError(ctx contract.Context, errorMessage string, request *tgtypes.TransferGatewayWithdrawTokenRequest) error {
	withdrawTokenError, err := proto.Marshal(&WithdrawTokenError{
		WithdrawTokenRequest: request,
		ErrorMessage:         errorMessage,
	})
	if err != nil {
		return err
	}
	ctx.EmitTopics(withdrawTokenError, withdrawTokenErrorTopic)
	return nil
}

func emitWithdrawLoomCoinError(ctx contract.Context, errorMessage string, request *tgtypes.TransferGatewayWithdrawLoomCoinRequest) error {
	withdrawLoomCoinError, err := proto.Marshal(&WithdrawLoomCoinError{
		WithdrawLoomCoinRequest: request,
		ErrorMessage:            errorMessage,
	})
	if err != nil {
		return err
	}
	ctx.EmitTopics(withdrawLoomCoinError, withdrawLoomCoinErrorTopic)
	return nil
}

func getMappedEthAddress(ctx contract.StaticContext, trustedValidators []*types.Address) ([]common.Address, error) {
	validatorEthAddresses := make([]common.Address, len(trustedValidators))

	addressMapper, err := ctx.Resolve("addressmapper")
	if err != nil {
		return nil, err
	}

	for i, validator := range trustedValidators {
		validatorAddress := loom.UnmarshalAddressPB(validator)
		ethAddress, err := resolveToEthAddr(ctx, addressMapper, validatorAddress)
		if err != nil {
			return nil, err
		}

		validatorEthAddresses[i] = common.BytesToAddress(ethAddress.Local)
	}

	return validatorEthAddresses, nil
}

// Returns all unclaimed tokens for an account
func unclaimedTokensByOwner(ctx contract.StaticContext, ownerAddr loom.Address) ([]*UnclaimedToken, error) {
	result := []*UnclaimedToken{}
	ownerKey := unclaimedTokensRangePrefix(ownerAddr)
	for _, entry := range ctx.Range(ownerKey) {
		var unclaimedToken UnclaimedToken
		if err := proto.Unmarshal(entry.Value, &unclaimedToken); err != nil {
			return nil, errors.Wrap(err, ErrFailedToReclaimToken.Error())
		}
		result = append(result, &unclaimedToken)
	}
	return result, nil
}

// Returns all unclaimed tokens for a token contract
func unclaimedTokenDepositorsByContract(ctx contract.StaticContext, tokenAddr loom.Address) ([]loom.Address, error) {
	result := []loom.Address{}
	contractKey := unclaimedTokenDepositorsRangePrefix(tokenAddr)
	for _, entry := range ctx.Range(contractKey) {
		var addr types.Address
		if err := proto.Unmarshal(entry.Value, &addr); err != nil {
			return nil, errors.Wrap(err, ErrFailedToReclaimToken.Error())
		}
		result = append(result, loom.UnmarshalAddressPB(&addr))
	}
	return result, nil
}

// taken from https://github.com/loomnetwork/loomchain/blob/2bd54308109c5a53526ae45f7b26a5a7042ffe5f/builtin/plugins/chainconfig/chainconfig.go#L359
func getCurrentValidators(ctx contract.StaticContext) ([]*types.Address, []*big.Int, *big.Int, error) {
	validatorsList := ctx.Validators()
	chainID := ctx.Block().ChainID

	if len(validatorsList) == 0 {
		return nil, nil, nil, errors.New("Empty validator list")
	}

	clusterStake := big.NewInt(0)
	valAddresses := make([]*types.Address, len(validatorsList))
	powers := make([]*big.Int, len(validatorsList))
	for i, v := range validatorsList {
		if v != nil {
			valAddresses[i] = loom.Address{
				ChainID: chainID,
				Local:   loom.LocalAddressFromPublicKey(v.PubKey),
			}.MarshalPB()
			powers[i] = big.NewInt(v.Power)

			clusterStake.Add(powers[i], clusterStake)
		}
	}

	return valAddresses, powers, clusterStake, nil
}

func isTokenKindAllowed(gwType GatewayType, tokenKind TokenKind) bool {
	switch gwType {
	case EthereumGateway:
		switch tokenKind {
		case TokenKind_ETH, TokenKind_ERC20, TokenKind_ERC721, TokenKind_ERC721X:
			return true
		default:
			return false
		}
	case LoomCoinGateway:
		switch tokenKind {
		case TokenKind_LoomCoin:
			return true
		default:
			return false
		}
	case TronGateway:
		switch tokenKind {
		case TokenKind_TRX, TokenKind_TRC20:
			return true
		default:
			return false
		}
	}
	return false
}
