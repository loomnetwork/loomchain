// +build evm

package gateway

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway/v1"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	ssha "github.com/miguelmota/go-solidity-sha3"
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
	WithdrawERC20Request            = tgtypes.TransferGatewayWithdrawERC20Request
	WithdrawERC721Request           = tgtypes.TransferGatewayWithdrawERC721Request
	WithdrawalReceiptRequest        = tgtypes.TransferGatewayWithdrawalReceiptRequest
	WithdrawalReceiptResponse       = tgtypes.TransferGatewayWithdrawalReceiptResponse
	ConfirmWithdrawalReceiptRequest = tgtypes.TransferGatewayConfirmWithdrawalReceiptRequest
	PendingWithdrawalsRequest       = tgtypes.TransferGatewayPendingWithdrawalsRequest
	PendingWithdrawalsResponse      = tgtypes.TransferGatewayPendingWithdrawalsResponse
	WithdrawalReceipt               = tgtypes.TransferGatewayWithdrawalReceipt
	UnclaimedToken                  = tgtypes.TransferGatewayUnclaimedToken
	ReclaimDepositorTokensRequest   = tgtypes.TransferGatewayReclaimDepositorTokensRequest
	ReclaimContractTokensRequest    = tgtypes.TransferGatewayReclaimContractTokensRequest
	Account                         = tgtypes.TransferGatewayAccount
	MainnetTokenDeposited           = tgtypes.TransferGatewayTokenDeposited
	MainnetTokenWithdrawn           = tgtypes.TransferGatewayTokenWithdrawn
	MainnetEvent                    = tgtypes.TransferGatewayMainnetEvent
	MainnetDepositEvent             = tgtypes.TransferGatewayMainnetEvent_Deposit
	MainnetWithdrawalEvent          = tgtypes.TransferGatewayMainnetEvent_Withdrawal
	TokenKind                       = tgtypes.TransferGatewayTokenKind
	PendingWithdrawalSummary        = tgtypes.TransferGatewayPendingWithdrawalSummary
	TokenWithdrawalSigned           = tgtypes.TransferGatewayTokenWithdrawalSigned
)

var (
	// Store keys
	stateKey                                = []byte("state")
	oracleStateKeyPrefix                    = []byte("oracle")
	accountKeyPrefix                        = []byte("account")
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

	TokenKind_ERC721 = tgtypes.TransferGatewayTokenKind_ERC721
	TokenKind_ERC20  = tgtypes.TransferGatewayTokenKind_ERC20
	TokenKind_ETH    = tgtypes.TransferGatewayTokenKind_ETH
)

func accountKey(owner loom.Address) []byte {
	return util.PrefixKey(accountKeyPrefix, owner.Bytes())
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

type Gateway struct {
}

func (gw *Gateway) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "gateway",
		Version: "0.1.0",
	}, nil
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
		Owner: req.Owner,
		NextContractMappingID: 1,
		LastMainnetBlockNum:   req.FirstMainnetBlockNum,
	})
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

	ctx.RevokePermissionFrom(oracleAddr, submitEventsPerm, oracleRole)
	ctx.RevokePermissionFrom(oracleAddr, signWithdrawalsPerm, oracleRole)
	ctx.RevokePermissionFrom(oracleAddr, verifyCreatorsPerm, oracleRole)
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
			if err := validateTokenDeposit(payload.Deposit); err != nil {
				ctx.Logger().Error("[Transfer Gateway] failed to process Mainnet deposit", "err", err)
				continue
			}

			ownerAddr := loom.UnmarshalAddressPB(payload.Deposit.TokenOwner)
			tokenAddr := loom.RootAddress("eth")
			if payload.Deposit.TokenContract != nil {
				tokenAddr = loom.UnmarshalAddressPB(payload.Deposit.TokenContract)
			}
			value := payload.Deposit.Value.Value.Int

			if err := transferTokenDeposit(ctx, ownerAddr, tokenAddr, payload.Deposit.TokenKind, value); err != nil {
				ctx.Logger().Error("[Transfer Gateway] failed to transfer Mainnet deposit", "err", err)
				if err := storeUnclaimedToken(ctx, payload.Deposit); err != nil {
					// this is a fatal error, discard the entire batch so that this deposit event
					// is resubmitted again in the next batch (hopefully after whatever caused this
					// error is resolved)
					return err
				}
			}

		case *tgtypes.TransferGatewayMainnetEvent_Withdrawal:
			if err := completeTokenWithdraw(ctx, state, payload.Withdrawal); err != nil {
				ctx.Logger().Error("[Transfer Gateway] failed to process Mainnet withdrawal", "err", err)
				continue
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

func (gw *Gateway) GetState(ctx contract.StaticContext, req *GatewayStateRequest) (*GatewayStateResponse, error) {
	state, err := loadState(ctx)
	if err != nil {
		return nil, err
	}
	return &GatewayStateResponse{State: state}, nil
}

// WithdrawERC721 will attempt to transfer an ERC721 token to the Gateway contract,
// if it's successful it will store a receipt than can be used by the depositor to reclaim ownership
// of the token through the Mainnet Gateway contract.
// NOTE: Currently an entity must complete each withdrawal by reclaiming ownership on Mainnet
//       before it can make another one withdrawal (even if the tokens originate from different
//       ERC721 or ERC20 contracts).
func (gw *Gateway) WithdrawERC721(ctx contract.Context, req *WithdrawERC721Request) error {
	if req.TokenId == nil || req.TokenContract == nil {
		return ErrInvalidRequest
	}

	ownerAddr := ctx.Message().Sender
	account, err := loadAccount(ctx, ownerAddr)
	if err != nil {
		return err
	}

	if account.WithdrawalReceipt != nil {
		return ErrPendingWithdrawalExists
	}

	mapperAddr, err := ctx.Resolve("addressmapper")
	if err != nil {
		return err
	}

	ownerEthAddr, err := resolveToEthAddr(ctx, mapperAddr, ownerAddr)
	if err != nil {
		return err
	}

	tokenAddr := loom.UnmarshalAddressPB(req.TokenContract)
	tokenEthAddr, err := resolveToForeignContractAddr(ctx, tokenAddr)
	if err != nil {
		return err
	}

	erc721 := newERC721Context(ctx, tokenAddr)
	// The entity wishing to make the withdrawal must first grant approval to the Gateway contract
	// to transfer the token, otherwise this will fail...
	if err = erc721.safeTransferFrom(ownerAddr, ctx.ContractAddress(), req.TokenId.Value.Int); err != nil {
		return err
	}

	// This check is mostly redundant, but might catch badly implemented ERC721 contracts
	curOwner, err := erc721.ownerOf(req.TokenId.Value.Int)
	if err != nil {
		return errors.Wrap(err, "failed to resolve token owner")
	}
	if curOwner.Compare(ctx.ContractAddress()) != 0 {
		return fmt.Errorf("token %v - %s hasn't been deposited to gateway",
			tokenAddr, req.TokenId.Value.Int.String())
	}

	ctx.Logger().Info("WithdrawERC721", "owner", ownerEthAddr, "token", tokenEthAddr)

	account.WithdrawalReceipt = &WithdrawalReceipt{
		TokenOwner:      ownerEthAddr.MarshalPB(),
		TokenContract:   tokenEthAddr.MarshalPB(),
		TokenKind:       TokenKind_ERC721,
		Value:           req.TokenId,
		WithdrawalNonce: account.WithdrawalNonce,
	}
	account.WithdrawalNonce++

	if err := saveAccount(ctx, account); err != nil {
		return err
	}

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	if err := addTokenWithdrawer(ctx, state, ownerAddr); err != nil {
		return err
	}

	return saveState(ctx, state)
}

// WithdrawERC20 will attempt to transfer ERC20 tokens to the Gateway contract,
// if it's successful it will store a receipt than can be used by the depositor to reclaim ownership
// of the tokens through the Mainnet Gateway contract.
// NOTE: Currently an entity must complete each withdrawal by reclaiming ownership on Mainnet
//       before it can make another one withdrawal (even if the tokens originate from different
//       ERC20 or ERC721 contracts).
func (gw *Gateway) WithdrawERC20(ctx contract.Context, req *WithdrawERC20Request) error {
	if req.Amount == nil || req.TokenContract == nil {
		return ErrInvalidRequest
	}

	ownerAddr := ctx.Message().Sender
	account, err := loadAccount(ctx, ownerAddr)
	if err != nil {
		return err
	}

	if account.WithdrawalReceipt != nil {
		return ErrPendingWithdrawalExists
	}

	mapperAddr, err := ctx.Resolve("addressmapper")
	if err != nil {
		return err
	}

	ownerEthAddr, err := resolveToEthAddr(ctx, mapperAddr, ownerAddr)
	if err != nil {
		return err
	}

	tokenAddr := loom.UnmarshalAddressPB(req.TokenContract)
	tokenEthAddr, err := resolveToForeignContractAddr(ctx, tokenAddr)
	if err != nil {
		return err
	}

	// The entity wishing to make the withdrawal must first grant approval to the Gateway contract
	// to transfer the tokens, otherwise this will fail...
	erc20 := newERC20Context(ctx, tokenAddr)
	if err := erc20.transferFrom(ownerAddr, ctx.ContractAddress(), req.Amount.Value.Int); err != nil {
		return err
	}

	ctx.Logger().Info("WithdrawERC20", "owner", ownerEthAddr, "token", tokenEthAddr)

	account.WithdrawalReceipt = &WithdrawalReceipt{
		TokenOwner:      ownerEthAddr.MarshalPB(),
		TokenContract:   tokenEthAddr.MarshalPB(),
		TokenKind:       TokenKind_ERC20,
		Value:           req.Amount,
		WithdrawalNonce: account.WithdrawalNonce,
	}
	account.WithdrawalNonce++

	if err := saveAccount(ctx, account); err != nil {
		return err
	}

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	if err := addTokenWithdrawer(ctx, state, ownerAddr); err != nil {
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

	ownerAddr := ctx.Message().Sender
	account, err := loadAccount(ctx, ownerAddr)
	if err != nil {
		return err
	}

	if account.WithdrawalReceipt != nil {
		return ErrPendingWithdrawalExists
	}

	mapperAddr, err := ctx.Resolve("addressmapper")
	if err != nil {
		return err
	}

	ownerEthAddr, err := resolveToEthAddr(ctx, mapperAddr, ownerAddr)
	if err != nil {
		return err
	}

	// The entity wishing to make the withdrawal must first grant approval to the Gateway contract
	// to transfer the tokens, otherwise this will fail...
	eth := newETHContext(ctx)
	if err := eth.transferFrom(ownerAddr, ctx.ContractAddress(), req.Amount.Value.Int); err != nil {
		return err
	}

	ctx.Logger().Info("WithdrawETH", "owner", ownerEthAddr)

	account.WithdrawalReceipt = &WithdrawalReceipt{
		TokenOwner:      ownerEthAddr.MarshalPB(),
		TokenContract:   req.MainnetGateway,
		TokenKind:       TokenKind_ETH,
		Value:           req.Amount,
		WithdrawalNonce: account.WithdrawalNonce,
	}
	account.WithdrawalNonce++

	if err := saveAccount(ctx, account); err != nil {
		return err
	}

	state, err := loadState(ctx)
	if err != nil {
		return err
	}

	if err := addTokenWithdrawer(ctx, state, ownerAddr); err != nil {
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
	account, err := loadAccount(ctx, owner)
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

	if req.TokenOwner == nil || req.OracleSignature == nil {
		return ErrInvalidRequest
	}

	ownerAddr := loom.UnmarshalAddressPB(req.TokenOwner)
	account, err := loadAccount(ctx, ownerAddr)
	if err != nil {
		return err
	}

	if account.WithdrawalReceipt == nil {
		return ErrMissingWithdrawalReceipt
	} else if account.WithdrawalReceipt.OracleSignature != nil {
		return ErrWithdrawalReceiptSigned
	}

	account.WithdrawalReceipt.OracleSignature = req.OracleSignature

	if err := saveAccount(ctx, account); err != nil {
		return err
	}

	wr := account.WithdrawalReceipt
	payload, err := proto.Marshal(&TokenWithdrawalSigned{
		TokenOwner:    wr.TokenOwner,
		TokenContract: wr.TokenContract,
		TokenKind:     wr.TokenKind,
		Value:         wr.Value,
		Sig:           wr.OracleSignature,
	})
	if err != nil {
		return err
	}
	// TODO: Re-enable the second topic when we fix an issue with subscribers receving the same
	//       event twice (or more depending on the number of topics).
	ctx.EmitTopics(payload, tokenWithdrawalSignedEventTopic /*, fmt.Sprintf("contract:%v", wr.TokenContract)*/)
	return nil
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

	mainnetGatewayAddr := common.BytesToAddress(req.MainnetGateway.Local)
	summaries := make([]*PendingWithdrawalSummary, 0, len(state.TokenWithdrawers))
	for _, ownerAddrPB := range state.TokenWithdrawers {
		ownerAddr := loom.UnmarshalAddressPB(ownerAddrPB)
		account, err := loadAccount(ctx, ownerAddr)
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

		var hash []byte
		switch receipt.TokenKind {
		case TokenKind_ERC721, TokenKind_ERC20:
			hash = ssha.SoliditySHA3(
				ssha.Uint256(receipt.GetValue().Value.Int),
				ssha.Address(common.BytesToAddress(receipt.TokenContract.Local)),
			)
		case TokenKind_ETH:
			hash = ssha.SoliditySHA3(ssha.Uint256(receipt.GetValue().Value.Int))
		default:
			ctx.Logger().Error("[Transfer Gateway] pending withdrawal has an invalid token kind",
				"tokenKind", receipt.TokenKind,
			)
			continue
		}

		hash = ssha.SoliditySHA3(
			ssha.Address(common.BytesToAddress(receipt.TokenOwner.Local)),
			ssha.Uint256(new(big.Int).SetUint64(receipt.WithdrawalNonce)),
			ssha.Address(mainnetGatewayAddr),
			hash,
		)

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
	mapperAddr, err := ctx.Resolve("addressmapper")
	if err != nil {
		return errors.Wrap(err, ErrFailedToReclaimToken.Error())
	}

	// Assume the caller is trying to reclaim their own tokens if depositors are not specified
	if len(req.Depositors) == 0 {
		ownerAddr, err := resolveToEthAddr(ctx, mapperAddr, ctx.Message().Sender)
		if err != nil {
			return errors.Wrap(err, ErrFailedToReclaimToken.Error())
		}

		return reclaimDepositorTokens(ctx, ownerAddr)
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
			ctx.Logger().Error("[Transfer Gateway] failed to reclaim depositor tokens",
				"owner", ownerAddr,
				"err", err,
			)
		}
	}
	return nil
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
		return errors.Wrap(err, ErrFailedToReclaimToken.Error())
	}

	cr, err := ctx.ContractRecord(localContractAddr)
	if err != nil {
		return errors.Wrap(err, ErrFailedToReclaimToken.Error())
	}

	callerAddr := ctx.Message().Sender
	if cr.CreatorAddress.Compare(callerAddr) != 0 {
		state, err := loadState(ctx)
		if err != nil {
			return errors.Wrap(err, ErrFailedToReclaimToken.Error())
		}
		if loom.UnmarshalAddressPB(state.Owner).Compare(callerAddr) != 0 {
			return ErrNotAuthorized
		}
	}

	for _, entry := range ctx.Range(unclaimedTokenDepositorsRangePrefix(foreignContractAddr)) {
		var addr types.Address
		if err := proto.Unmarshal(entry.Value, &addr); err != nil {
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
			ctx.Logger().Error("[Transfer Gateway] failed to load unclaimed tokens",
				"owner", ownerAddr,
				"token", foreignContractAddr,
				"err", err,
			)
		}
		if err := reclaimDepositorTokensForContract(ctx, ownerAddr, foreignContractAddr, &unclaimedToken); err != nil {
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
	failed := []*types.BigUInt{}
	for _, v := range unclaimedToken.Values {
		if err := transferTokenDeposit(ctx, ownerAddr, tokenAddr, unclaimedToken.TokenKind, v.Value.Int); err != nil {
			failed = append(failed, v)
			ctx.Logger().Error(ErrFailedToReclaimToken.Error(),
				"owner", ownerAddr,
				"token", tokenAddr,
				"kind", unclaimedToken.TokenKind,
				"value", v.Value.String(),
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
		unclaimedToken.Values = failed
		if err := ctx.Set(tokenKey, unclaimedToken); err != nil {
			return errors.Wrap(err, ErrFailedToReclaimToken.Error())
		}
	}

	return nil
}

// Performs basic validation to ensure all required deposit fields are set.
func validateTokenDeposit(deposit *MainnetTokenDeposited) error {
	switch deposit.TokenKind {
	case TokenKind_ERC721, TokenKind_ERC20:
		if deposit.TokenContract == nil {
			return ErrInvalidRequest
		}
		fallthrough

	case TokenKind_ETH:
		if deposit.TokenOwner == nil || deposit.Value == nil {
			return ErrInvalidRequest
		}

	default:
		return fmt.Errorf("%v deposits not supported", deposit.TokenKind)
	}
	return nil
}

// When a token is deposited to the Mainnet Gateway mint it on the DAppChain if it doesn't exist
// yet, and transfer it to the owner's DAppChain address.
func transferTokenDeposit(ctx contract.Context, ownerEthAddr, tokenEthAddr loom.Address, kind TokenKind, value *big.Int) error {
	mapperAddr, err := ctx.Resolve("addressmapper")
	if err != nil {
		return err
	}

	ownerAddr, err := resolveToDAppAddr(ctx, mapperAddr, ownerEthAddr)
	if err != nil {
		return errors.Wrapf(err, "no mapping exists for account %v", ownerEthAddr)
	}

	switch kind {
	case TokenKind_ERC721:
		tokenAddr, err := resolveToLocalContractAddr(ctx, tokenEthAddr)
		if err != nil {
			return errors.Wrapf(err, "no mapping exists for token %v", tokenEthAddr)
		}

		tokenID := value
		erc721 := newERC721Context(ctx, tokenAddr)

		exists, err := erc721.exists(tokenID)
		if err != nil {
			return err
		}

		if !exists {
			if err := erc721.mintToGateway(tokenID); err != nil {
				return errors.Wrapf(err, "failed to mint token %v - %s", tokenAddr, tokenID.String())
			}
		}

		// At this point the token is owned by the associated token contract, so transfer it back to the
		// original owner...
		if err := erc721.safeTransferFrom(ctx.ContractAddress(), ownerAddr, tokenID); err != nil {
			return errors.Wrapf(err, "failed to transfer ERC721 token")
		}

	case TokenKind_ERC20:
		tokenAddr, err := resolveToLocalContractAddr(ctx, tokenEthAddr)
		if err != nil {
			return errors.Wrapf(err, "no mapping exists for token %v", tokenEthAddr)
		}

		amount := value
		erc20 := newERC20Context(ctx, tokenAddr)
		availableFunds, err := erc20.balanceOf(ctx.ContractAddress())
		if err != nil {
			return err
		}

		// If the DAppChain Gateway doesn't have sufficient funds to complete the transfer then
		// try to mint the required amount...
		if availableFunds.Cmp(amount) < 0 {
			if err := erc20.mintToGateway(amount); err != nil {
				return errors.Wrapf(err, "failed to mint tokens %v - %s", tokenAddr, amount.String())
			}
		}

		if err := erc20.transfer(ownerAddr, amount); err != nil {
			return errors.Wrap(err, "failed to transfer ERC20 tokens")
		}

	case TokenKind_ETH:
		amount := value
		eth := newETHContext(ctx)
		availableFunds, err := eth.balanceOf(ctx.ContractAddress())
		if err != nil {
			return err
		}

		// If the DAppChain Gateway doesn't have sufficient funds to complete the transfer then
		// try to mint the required amount...
		if availableFunds.Cmp(amount) < 0 {
			if err := eth.mintToGateway(amount); err != nil {
				return errors.Wrapf(err, "failed to mint ETH - %s", amount.String())
			}
		}

		if err := eth.transfer(ownerAddr, amount); err != nil {
			return errors.Wrap(err, "failed to transfer ETH")
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
		unclaimedToken.Values = append(unclaimedToken.Values, deposit.Value)

	case TokenKind_ERC20, TokenKind_ETH:
		oldAmount := new(big.Int)
		if len(unclaimedToken.Values) == 1 && unclaimedToken.Values[0] != nil {
			oldAmount = unclaimedToken.Values[0].Value.Int
		}
		newAmount := oldAmount.Add(oldAmount, deposit.Value.Value.Int)
		unclaimedToken.Values = []*types.BigUInt{
			&types.BigUInt{Value: *loom.NewBigUInt(newAmount)},
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
	switch withdrawal.TokenKind {
	case TokenKind_ERC721, TokenKind_ERC20:
		if withdrawal.TokenContract == nil {
			return ErrInvalidRequest
		}
		fallthrough

	case TokenKind_ETH:
		if withdrawal.TokenOwner == nil || withdrawal.Value == nil {
			return ErrInvalidRequest
		}

	default:
		return fmt.Errorf("%v withdrawals not supported", withdrawal.TokenKind)
	}

	mapperAddr, err := ctx.Resolve("addressmapper")
	if err != nil {
		return err
	}

	ownerEthAddr := loom.UnmarshalAddressPB(withdrawal.TokenOwner)
	ownerAddr, err := resolveToDAppAddr(ctx, mapperAddr, ownerEthAddr)
	if err != nil {
		return errors.Wrapf(err, "no mapping exists for account %v", ownerEthAddr)
	}

	account, err := loadAccount(ctx, ownerAddr)
	if err != nil {
		return err
	}

	// TODO: check contract address & token ID match the receipt

	if account.WithdrawalReceipt == nil {
		return errors.New("no pending withdrawal found")
	}
	account.WithdrawalReceipt = nil

	if err := saveAccount(ctx, account); err != nil {
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

func loadAccount(ctx contract.StaticContext, owner loom.Address) (*Account, error) {
	account := Account{Owner: owner.MarshalPB()}
	err := ctx.Get(accountKey(owner), &account)
	if err != nil && err != contract.ErrNotFound {
		return nil, errors.Wrapf(err, "failed to load account for %v", owner)
	}
	return &account, nil
}

func saveAccount(ctx contract.Context, acct *Account) error {
	ownerAddr := loom.UnmarshalAddressPB(acct.Owner)
	if err := ctx.Set(accountKey(ownerAddr), acct); err != nil {
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

var Contract plugin.Contract = contract.MakePluginContract(&Gateway{})
