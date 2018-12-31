// +build evm

package gateway

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	ssha "github.com/miguelmota/go-solidity-sha3"
	"github.com/pkg/errors"
)

type LoomcoinGateway struct {
}

func (gw *LoomcoinGateway) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "loomcoin-gateway",
		Version: "0.1.0",
	}, nil
}

func (gw *LoomcoinGateway) Init(ctx contract.Context, req *InitRequest) error {
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

func (gw *LoomcoinGateway) AddOracle(ctx contract.Context, req *AddOracleRequest) error {
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

func (gw *LoomcoinGateway) RemoveOracle(ctx contract.Context, req *RemoveOracleRequest) error {
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

func (gw *LoomcoinGateway) GetOracles(ctx contract.StaticContext, req *GetOraclesRequest) (*GetOraclesResponse, error) {
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

func (gw *LoomcoinGateway) ProcessEventBatch(ctx contract.Context, req *ProcessEventBatchRequest) error {
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
			if payload.Deposit.TokenKind != TokenKind_LoomCoin {
				// TODO Replace with more contexual error
				return ErrInvalidRequest
			}

			if err := validateTokenDeposit(payload.Deposit); err != nil {
				ctx.Logger().Error("[Transfer Gateway] failed to process Mainnet deposit", "err", err)
				continue
			}

			ownerAddr := loom.UnmarshalAddressPB(payload.Deposit.TokenOwner)
			tokenAddr := loom.RootAddress("eth")
			if payload.Deposit.TokenContract != nil {
				tokenAddr = loom.UnmarshalAddressPB(payload.Deposit.TokenContract)
			}

			err = transferTokenDeposit(
				ctx, ownerAddr, tokenAddr,
				payload.Deposit.TokenKind, payload.Deposit.TokenID, payload.Deposit.TokenAmount)
			if err != nil {
				ctx.Logger().Error("[Transfer Gateway] failed to transfer Mainnet deposit", "err", err)
				if err := storeUnclaimedToken(ctx, payload.Deposit); err != nil {
					// this is a fatal error, discard the entire batch so that this deposit event
					// is resubmitted again in the next batch (hopefully after whatever caused this
					// error is resolved)
					return err
				}
			}

		case *tgtypes.TransferGatewayMainnetEvent_Withdrawal:
			if payload.Withdrawal.TokenKind != TokenKind_LoomCoin {
				// TODO Replace with more contexual error
				return ErrInvalidRequest
			}

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

func (gw *LoomcoinGateway) GetState(ctx contract.StaticContext, req *GatewayStateRequest) (*GatewayStateResponse, error) {
	state, err := loadState(ctx)
	if err != nil {
		return nil, err
	}
	return &GatewayStateResponse{State: state}, nil
}

// WithdrawLoomCoin will attempt to transfer Loomcoin to the Gateway contract,
// if it's successful it will store a receipt than can be used by the depositor to reclaim ownership
// of the Loomcoin through the Mainnet Gateway contract.
// NOTE: Currently an entity must complete each withdrawal by reclaiming ownership on Mainnet
//       before it can make another withdrawal (even if the tokens/ETH/Loom originate from different
//       ERC20 or ERC721 contracts).
func (gw *LoomcoinGateway) WithdrawLoomCoin(ctx contract.Context, req *WithdrawLoomCoinRequest) error {
	if req.Amount == nil || req.TokenContract == nil {
		return ErrInvalidRequest
	}

	ownerAddr := ctx.Message().Sender
	account, err := loadLocalAccount(ctx, ownerAddr)
	if err != nil {
		return err
	}

	if account.WithdrawalReceipt != nil {
		return ErrPendingWithdrawalExists
	}

	ownerEthAddr := loom.Address{}
	if req.Recipient != nil {
		ownerEthAddr = loom.UnmarshalAddressPB(req.Recipient)
	} else {
		mapperAddr, err := ctx.Resolve("addressmapper")
		if err != nil {
			return err
		}

		ownerEthAddr, err = resolveToEthAddr(ctx, mapperAddr, ownerAddr)
		if err != nil {
			return err
		}
	}

	foreignAccount, err := loadForeignAccount(ctx, ownerEthAddr)
	if err != nil {
		return err
	}

	if foreignAccount.CurrentWithdrawer != nil {
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

	if err := saveForeignAccount(ctx, foreignAccount); err != nil {
		return err
	}

	if err := saveLocalAccount(ctx, account); err != nil {
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
func (gw *LoomcoinGateway) WithdrawalReceipt(ctx contract.StaticContext, req *WithdrawalReceiptRequest) (*WithdrawalReceiptResponse, error) {
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
func (gw *LoomcoinGateway) ConfirmWithdrawalReceipt(ctx contract.Context, req *ConfirmWithdrawalReceiptRequest) error {
	if ok, _ := ctx.HasPermission(signWithdrawalsPerm, []string{oracleRole}); !ok {
		return ErrNotAuthorized
	}

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
	// TODO: Re-enable the second topic when we fix an issue with subscribers receving the same
	//       event twice (or more depending on the number of topics).
	ctx.EmitTopics(payload, tokenWithdrawalSignedEventTopic /*, fmt.Sprintf("contract:%v", wr.TokenContract)*/)
	return nil
}

// PendingWithdrawals will return the token owner & withdrawal hash for all pending withdrawals.
// The Oracle will call this method periodically and sign all the retrieved hashes.
func (gw *LoomcoinGateway) PendingWithdrawals(ctx contract.StaticContext, req *PendingWithdrawalsRequest) (*PendingWithdrawalsResponse, error) {
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

		safeAmount := big.NewInt(0)
		if receipt.TokenAmount != nil {
			safeAmount = receipt.TokenAmount.Value.Int
		}

		if receipt.TokenKind != TokenKind_LoomCoin {
			ctx.Logger().Error("[Transfer Gateway] pending withdrawal has an invalid token kind",
				"tokenKind", receipt.TokenKind,
			)
			continue
		}

		transferInfoHash := ssha.SoliditySHA3(
			ssha.Uint256(safeAmount),
			ssha.Address(common.BytesToAddress(receipt.TokenContract.Local)),
		)

		hash := ssha.SoliditySHA3(
			ssha.Address(common.BytesToAddress(receipt.TokenOwner.Local)),
			ssha.Uint256(new(big.Int).SetUint64(receipt.WithdrawalNonce)),
			ssha.Address(mainnetGatewayAddr),
			transferInfoHash,
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
func (gw *LoomcoinGateway) ReclaimDepositorTokens(ctx contract.Context, req *ReclaimDepositorTokensRequest) error {
	// Assume the caller is trying to reclaim their own tokens if depositors are not specified
	if len(req.Depositors) == 0 {
		mapperAddr, err := ctx.Resolve("addressmapper")
		if err != nil {
			return errors.Wrap(err, ErrFailedToReclaimToken.Error())
		}

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
func (gw *LoomcoinGateway) ReclaimContractTokens(ctx contract.Context, req *ReclaimContractTokensRequest) error {
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

var LoomCoinContract plugin.Contract = contract.MakePluginContract(&LoomcoinGateway{})
