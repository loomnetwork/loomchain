package user_deployer_whitelist

import (
	"bytes"
	"encoding/binary"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	dwtypes "github.com/loomnetwork/go-loom/builtin/types/deployer_whitelist"
	udwtypes "github.com/loomnetwork/go-loom/builtin/types/user_deployer_whitelist"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/loomnetwork/loomchain/features"
	"github.com/pkg/errors"
)

type (
	GetUserDeployersRequest        = udwtypes.GetUserDeployersRequest
	GetUserDeployersResponse       = udwtypes.GetUserDeployersResponse
	GetDeployedContractsRequest    = udwtypes.GetDeployedContractsRequest
	GetDeployedContractsResponse   = udwtypes.GetDeployedContractsResponse
	DestroyDeployedContractRequest = udwtypes.DestroyDeployedContractsRequest
	GetDeployerResponse            = dwtypes.GetDeployerResponse
	GetDeployerRequest             = dwtypes.GetDeployerRequest
	GetTierInfoRequest             = udwtypes.GetTierInfoRequest
	GetTierInfoResponse            = udwtypes.GetTierInfoResponse
	SetTierInfoRequest             = udwtypes.SetTierInfoRequest
	Deployer                       = dwtypes.Deployer
	UserDeployerState              = udwtypes.UserDeployerState
	AddUserDeployerRequest         = dwtypes.AddUserDeployerRequest
	WhitelistUserDeployerRequest   = udwtypes.WhitelistUserDeployerRequest
	UserState                      = udwtypes.UserState
	InitRequest                    = udwtypes.InitRequest
	Tier                           = udwtypes.Tier
	TierID                         = udwtypes.TierID
	RemoveUserDeployerRequest      = udwtypes.RemoveUserDeployerRequest
)

var (
	// ErrNotAuthorized indicates that a contract method failed because the caller didn't have
	// the permission to execute that method.
	ErrNotAuthorized = errors.New("[UserDeployerWhitelist] not authorized")
	// ErrInvalidRequest is a generic error that's returned when something is wrong with the
	// request message, e.g. missing or invalid fields.
	ErrInvalidRequest = errors.New("[UserDeployerWhitelist] invalid request")
	// ErrOwnerNotSpecified returned if init request does not have owner address
	ErrOwnerNotSpecified = errors.New("[UserDeployerWhitelist] owner not specified")
	// ErrDeployerAlreadyExists returned if an owner try to set an existing feature
	ErrDeployerAlreadyExists = errors.New("[UserDeployerWhitelist] deployer already exists")
	// ErrDeployerDoesNotExist returned if an owner try to to remove a deployer that does not exist
	ErrDeployerDoesNotExist = errors.New("[UserDeployerWhitelist] deployer does not exist")
	// ErrInsufficientBalance returned if loom balance is unsufficient for whitelisting
	ErrInsufficientBalance = errors.New("[UserDeployerWhitelist] Insufficient Loom Balance for whitelisting")
	// ErrInvalidTier returned if Tier provided is invalid
	ErrInvalidTier = errors.New("[UserDeployerWhitelist] Invalid Tier")
	// ErrMissingTierInfo is returned if init doesnt get atleast one tier
	ErrMissingTierInfo = errors.New("[UserDeployerWhitelist] no tiers provided")
	// ErrInvalidWhitelistingFee indicates the specified fee is invalid
	ErrInvalidWhitelistingFee = errors.New("[UserDeployerWhitelist] fee must be greater than zero")
	// ErrInvalidBlockRange indicates the specified block range is invalid
	ErrInvalidBlockRange = errors.New("[UserDeployerWhitelist] block range must be greater than zero")
	// ErrInvalidMaxTxs indicates the specified max txs count is invalid
	ErrInvalidMaxTxs = errors.New("[UserDeployerWhitelist] max txs must be greater than zero")
)

const (
	ownerRole = "owner"
	//This state stores deployer contract mapping
	deployerStatePrefix = "ds"
	//This state stores deployers corresponding to the user
	userStatePrefix = "us"
	tierKeyPrefix   = "ti"
)

var (
	modifyPerm = []byte("modp")
)

type UserDeployerWhitelist struct {
}

func deployerStateKey(deployer loom.Address) []byte {
	return util.PrefixKey([]byte(deployerStatePrefix), deployer.Bytes())
}

func userStateKey(user loom.Address) []byte {
	return util.PrefixKey([]byte(userStatePrefix), user.Bytes())
}

func tierKey(tierID TierID) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, tierID)
	return util.PrefixKey([]byte(tierKeyPrefix), buf.Bytes())
}

func (uw *UserDeployerWhitelist) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "user-deployer-whitelist",
		Version: "1.0.0",
	}, nil
}

func (uw *UserDeployerWhitelist) Init(ctx contract.Context, req *InitRequest) error {
	if req.Owner == nil {
		return ErrOwnerNotSpecified
	}
	if req.TierInfo == nil {
		return ErrMissingTierInfo
	}
	div := loom.NewBigUIntFromInt(10)
	div.Exp(div, loom.NewBigUIntFromInt(18), nil)
	ownerAddr := loom.UnmarshalAddressPB(req.Owner)

	// TODO: Add relevant methods to manage owner and permissions later on.
	ctx.GrantPermissionTo(ownerAddr, modifyPerm, ownerRole)

	for _, ti := range req.TierInfo {
		fees := loom.NewBigUIntFromInt(int64(ti.Fee))
		fees.Mul(fees, div)
		tier := &Tier{
			TierID: ti.TierID,
			Fee: &types.BigUInt{
				Value: *fees,
			},
			Name: ti.Name,
		}

		if ctx.FeatureEnabled(features.UserDeployerWhitelistVersion1_1Feature, false) {
			if ti.BlockRange == 0 {
				return ErrInvalidBlockRange
			}
			if ti.MaxTxs == 0 {
				return ErrInvalidMaxTxs
			}

			tier.BlockRange = ti.BlockRange
			tier.MaxTxs = ti.MaxTxs
		}

		if err := ctx.Set(tierKey(tier.TierID), tier); err != nil {
			return err
		}
	}
	return nil
}

// AddUserDeployer should be called by a third-party dev to authorize an account to deploy EVM contracts
// on their behalf. In order to authorize an account the caller must approve the contract to withdraw
// the fee for whitelisting (charged in LOOM coin) before calling this method, the fee will be
// deducted from the caller if the requested account is successfuly authorized.
func (uw *UserDeployerWhitelist) AddUserDeployer(ctx contract.Context, req *WhitelistUserDeployerRequest) error {
	if req.DeployerAddr == nil {
		return ErrInvalidRequest
	}
	dwAddr, err := ctx.Resolve("deployerwhitelist")
	if err != nil {
		return errors.Wrap(err, "unable to get address of deployer_whitelist")
	}
	coinAddr, err := ctx.Resolve("coin")
	if err != nil {
		return errors.Wrap(err, "unable to get address of coin contract")
	}
	// Check the deployer account is not already whitelisted
	if ctx.Has(deployerStateKey(loom.UnmarshalAddressPB(req.DeployerAddr))) {
		return ErrDeployerAlreadyExists
	}
	var tierInfo Tier
	if err := ctx.Get(tierKey(req.TierID), &tierInfo); err != nil {
		return err
	}
	var whitelistingFees *types.BigUInt
	whitelistingFees = &types.BigUInt{Value: tierInfo.Fee.Value}
	userAddr := ctx.Message().Sender
	// Debit the whitelisting fee the caller's LOOM balance
	coinReq := &coin.TransferFromRequest{
		To:     ctx.ContractAddress().MarshalPB(),
		From:   userAddr.MarshalPB(),
		Amount: whitelistingFees,
	}
	if err := contract.CallMethod(ctx, coinAddr, "TransferFrom", coinReq, nil); err != nil {
		if err == coin.ErrSenderBalanceTooLow {
			return ErrInsufficientBalance
		}
		return errors.Wrap(err, "Unable to transfer coin")
	}
	adddeprequest := &dwtypes.AddUserDeployerRequest{
		DeployerAddr: req.DeployerAddr,
	}
	if err := contract.CallMethod(ctx, dwAddr, "AddUserDeployer", adddeprequest, nil); err != nil {
		return errors.Wrap(err, "failed to whitelist deployer")
	}

	var userState UserState
	if err := ctx.Get(userStateKey(userAddr), &userState); err != nil {
		// This is taking care of boundary case also that user is whitelisting deployers for first time
		if err != contract.ErrNotFound {
			return errors.Wrap(err, "[UserDeployerWhitelist] Failed to load User State")
		}
	}
	userState.Deployers = append(userState.Deployers, req.DeployerAddr)
	if err := ctx.Set(userStateKey(userAddr), &userState); err != nil {
		return errors.Wrap(err, "Failed to Save Deployers mapping in user state")
	}
	deployer := &UserDeployerState{
		Address: req.DeployerAddr,
		TierID:  req.TierID,
	}
	if err := ctx.Set(deployerStateKey(loom.UnmarshalAddressPB(req.DeployerAddr)), deployer); err != nil {
		return errors.Wrap(err, "Failed to Save WhitelistedDeployer in whitelisted deployers state")
	}
	return nil

}

// RemoveUserDeployer to remove a deployerAddr from whitelisted deployers
func (uw *UserDeployerWhitelist) RemoveUserDeployer(ctx contract.Context, req *udwtypes.RemoveUserDeployerRequest) error {
	if req.DeployerAddr == nil {
		return ErrInvalidRequest
	}
	dwAddr, err := ctx.Resolve("deployerwhitelist")
	if err != nil {
		return errors.Wrap(err, "unable to get address of deployer_whitelist")
	}
	deployerAddr := loom.UnmarshalAddressPB(req.DeployerAddr)
	// check if sender is the user who whitelisted the DeployerAddr, if so remove from userState
	userAddr := ctx.Message().Sender
	var userState UserState
	if err := ctx.Get(userStateKey(userAddr), &userState); err != nil {
		if err != contract.ErrNotFound {
			return errors.Wrap(err, "[UserDeployerWhitelist] Failed to load User State")
		}
	}

	isInputDeployerAddrValid := false
	survivedDeployers := make([]*types.Address, 0, len(userState.Deployers))
	deployers := userState.Deployers
	for _, addr := range deployers {
		currentDeployerAddr := loom.UnmarshalAddressPB(addr)
		if currentDeployerAddr.Compare(deployerAddr) == 0 {
			isInputDeployerAddrValid = true
		} else {
			survivedDeployers = append(survivedDeployers, addr)
		}
	}

	if !isInputDeployerAddrValid {
		return ErrDeployerDoesNotExist
	}

	userState.Deployers = survivedDeployers
	if err := ctx.Set(userStateKey(userAddr), &userState); err != nil {
		return errors.Wrap(err, "failed to Save Deployers mapping in user state")
	}
	// remove from deployerwhitelist contract
	removeUserDeployerRequest := &RemoveUserDeployerRequest{
		DeployerAddr: req.DeployerAddr,
	}
	if err := contract.CallMethod(ctx, dwAddr, "RemoveUserDeployer", removeUserDeployerRequest, nil); err != nil {
		return errors.Wrap(err, "failed to remove deployer")
	}
	if ctx.FeatureEnabled(features.UserDeployerWhitelistVersion1_2Feature, false) {
		var userDeployer UserDeployerState
		if err := ctx.Get(deployerStateKey(deployerAddr), &userDeployer); err != nil {
			return errors.Wrap(err, "Failed to Get Deployer State")
		}
		userDeployer.Inactive = true
		if err := ctx.Set(deployerStateKey(deployerAddr), &userDeployer); err != nil {
			return errors.Wrap(err, "Saving WhitelistedDeployer in whitelisted deployers state")
		}
		return nil
	}
	if !ctx.Has(deployerStateKey(deployerAddr)) {
		return ErrDeployerDoesNotExist
	}
	ctx.Delete(deployerStateKey(deployerAddr))
	return nil
}

// GetUserDeployers returns the list of accounts authorized to deploy EVM contracts on behalf of the
// specified user.
func (uw *UserDeployerWhitelist) GetUserDeployers(
	ctx contract.StaticContext, req *GetUserDeployersRequest,
) (*GetUserDeployersResponse, error) {
	var userState UserState
	if req.UserAddr == nil {
		return nil, ErrInvalidRequest
	}
	deployers := []*UserDeployerState{}
	err := ctx.Get(userStateKey(loom.UnmarshalAddressPB(req.UserAddr)), &userState)
	if err != nil {
		if err == contract.ErrNotFound {
			return &GetUserDeployersResponse{}, nil
		}
		return nil, err
	}
	for _, deployerAddr := range userState.Deployers {
		var userDeployerState UserDeployerState
		err = ctx.Get(deployerStateKey(loom.UnmarshalAddressPB(deployerAddr)), &userDeployerState)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to load whitelisted deployers state")
		}
		if req.IncludeInactive || !userDeployerState.Inactive {
			deployers = append(deployers, &userDeployerState)
		}

	}
	return &GetUserDeployersResponse{
		Deployers: deployers,
	}, nil
}

// GetDeployedContracts returns the list of EVM contracts deployed by a particular deployer account.
func (uw *UserDeployerWhitelist) GetDeployedContracts(
	ctx contract.StaticContext, req *GetDeployedContractsRequest,
) (*GetDeployedContractsResponse, error) {
	if req.DeployerAddr == nil {
		return nil, ErrInvalidRequest
	}
	deployerAddr := loom.UnmarshalAddressPB(req.DeployerAddr)
	var userDeployer UserDeployerState
	err := ctx.Get(deployerStateKey(deployerAddr), &userDeployer)
	if err != nil {
		if err == contract.ErrNotFound {
			return &GetDeployedContractsResponse{}, nil
		}
		return nil, errors.Wrap(err, "Failed to load whitelisted deployers state")
	}
	return &GetDeployedContractsResponse{
		ContractAddresses: userDeployer.Contracts,
	}, nil
}

// DestroyDeployedContract destroys a deployed contract
func (uw *UserDeployerWhitelist) DestroyDeployedContract(
	ctx contract.Context, req *DestroyDeployedContractRequest,
) error {
	if req.ContractAddress == nil {
		return ErrInvalidRequest
	}
	contractAddress := loom.UnmarshalAddressPB(req.ContractAddress)
	isContractOwner, _ := ctx.HasPermission(modifyPerm, []string{ownerRole})
	// user deployer whitelist contract owner can destroy every contract
	if isContractOwner {
		return ctx.DestroyEVMContract(contractAddress)
	}

	senderAddr := ctx.Message().Sender
	var userDeployer UserDeployerState
	// load user deployer state
	err := ctx.Get(deployerStateKey(senderAddr), &userDeployer)
	if err != nil {
		return errors.Wrap(err, "Failed to load whitelisted deployers state")
	}
	authorized := false
	contracts := []*udwtypes.DeployerContract{}
	for _, contract := range userDeployer.Contracts {
		contractAddr := loom.UnmarshalAddressPB(contract.ContractAddress)
		if contractAddr.Compare(contractAddress) == 0 {
			authorized = true
		} else {
			contracts = append(contracts, contract)
		}
	}
	if !authorized {
		return ErrNotAuthorized
	}

	if err := ctx.DestroyEVMContract(contractAddress); err != nil {
		return err
	}

	// update deployed contracts list
	userDeployer.Contracts = contracts
	if err := ctx.Set(deployerStateKey(senderAddr), &userDeployer); err != nil {
		return errors.Wrap(err, "Failed to save deployer state")
	}

	return nil
}

// SwapUserDeployer allows a user to swap one of their deployer accounts for another (essentially
// rotating deployment keys), this operation removes one deployer account and adds another one,
// but doesn't require the user to pay the whitelisting fee again. The deployer account that's
// swapped out will become inactive which means that it will no longer be possible to send call txs
// to any contracts deployed by that account when the per-contract tx limiter is enabled.
func (uw *UserDeployerWhitelist) SwapUserDeployer(
	ctx contract.Context, req *udwtypes.SwapUserDeployerRequest,
) error {
	if req.OldDeployerAddr == nil || req.NewDeployerAddr == nil {
		return ErrInvalidRequest
	}
	if !ctx.FeatureEnabled(features.UserDeployerWhitelistVersion1_2Feature, false) {
		return errors.New("feature not enabled")
	}
	dwAddr, err := ctx.Resolve("deployerwhitelist")
	if err != nil {
		return errors.Wrap(err, "failed to resolve deployer whitelist contract")
	}
	if ctx.Has(deployerStateKey(loom.UnmarshalAddressPB(req.NewDeployerAddr))) {
		return ErrDeployerAlreadyExists
	}

	oldDeployerAddr := loom.UnmarshalAddressPB(req.OldDeployerAddr)
	var oldDeployer UserDeployerState
	err = ctx.Get(deployerStateKey(oldDeployerAddr), &oldDeployer)
	if err != nil {
		return errors.Wrap(err, "failed to load old deployer")
	}
	userAddr := ctx.Message().Sender
	var userState UserState
	if err := ctx.Get(userStateKey(userAddr), &userState); err != nil {
		return errors.Wrap(err, "failed to load user state")
	}
	isInputDeployerAddrValid := false
	survivedDeployers := make([]*types.Address, 0, len(userState.Deployers))
	for _, addr := range userState.Deployers {
		currentDeployerAddr := loom.UnmarshalAddressPB(addr)
		if currentDeployerAddr.Compare(oldDeployerAddr) == 0 {
			isInputDeployerAddrValid = true
		} else {
			survivedDeployers = append(survivedDeployers, addr)
		}
	}

	if !isInputDeployerAddrValid {
		return ErrDeployerDoesNotExist
	}

	survivedDeployers = append(survivedDeployers, req.NewDeployerAddr)
	userState.Deployers = survivedDeployers
	if err := ctx.Set(userStateKey(userAddr), &userState); err != nil {
		return errors.Wrap(err, "failed to save user state")
	}
	// Update DeployerWhitelist contract to reflect the deployer account swap
	removeUserDeployerRequest := &RemoveUserDeployerRequest{
		DeployerAddr: req.OldDeployerAddr,
	}
	if err := contract.CallMethod(ctx, dwAddr, "RemoveUserDeployer", removeUserDeployerRequest, nil); err != nil {
		return errors.Wrap(err, "failed to remove deployer")
	}
	addUserDeployerRequest := &dwtypes.AddUserDeployerRequest{
		DeployerAddr: req.NewDeployerAddr,
	}
	if err := contract.CallMethod(ctx, dwAddr, "AddUserDeployer", addUserDeployerRequest, nil); err != nil {
		return errors.Wrap(err, "failed to whitelist deployer")
	}

	newDeployer := &UserDeployerState{
		Address: req.NewDeployerAddr,
		TierID:  oldDeployer.TierID,
	}
	if err := ctx.Set(deployerStateKey(loom.UnmarshalAddressPB(req.NewDeployerAddr)), newDeployer); err != nil {
		return errors.Wrap(err, "failed to save new deployer")
	}
	oldDeployer.Inactive = true
	if err := ctx.Set(deployerStateKey(oldDeployerAddr), &oldDeployer); err != nil {
		return errors.Wrap(err, "failed to save old deployer")
	}
	return nil
}

// GetTierInfo returns the details of a specific tier.
func (uw *UserDeployerWhitelist) GetTierInfo(
	ctx contract.StaticContext, req *GetTierInfoRequest,
) (*GetTierInfoResponse, error) {
	var tier Tier
	if err := ctx.Get(tierKey(req.TierID), &tier); err != nil {
		return nil, errors.Wrap(err, "Failed to get Tier Information")
	}
	return &GetTierInfoResponse{
		Tier: &tier,
	}, nil
}

// SetTierInfo sets the details of a tier.
func (uw *UserDeployerWhitelist) SetTierInfo(ctx contract.Context, req *SetTierInfoRequest) error {
	if req.Fee == nil || req.Fee.Value.Cmp(loom.NewBigUIntFromInt(0)) == 0 {
		return ErrInvalidWhitelistingFee
	}
	if ok, _ := ctx.HasPermission(modifyPerm, []string{ownerRole}); !ok {
		return ErrNotAuthorized
	}

	tier := &Tier{
		TierID: req.TierID,
		Fee:    req.Fee,
		Name:   req.Name,
	}
	if ctx.FeatureEnabled(features.UserDeployerWhitelistVersion1_1Feature, false) {
		if req.BlockRange == 0 {
			return ErrInvalidBlockRange
		}
		if req.MaxTxs == 0 {
			return ErrInvalidMaxTxs
		}

		tier.BlockRange = req.BlockRange
		tier.MaxTxs = req.MaxTxs
	}
	if err := ctx.Set(tierKey(req.TierID), tier); err != nil {
		return errors.Wrap(err, "Failed to modify TierInfo")
	}
	return nil
}

// RecordEVMContractDeployment is called by the EVMDeployRecorderPostCommitMiddleware after an EVM
// contract is successfully deployed to record the deployment in the UserDeployerWhitelist contract.
func RecordEVMContractDeployment(ctx contract.Context, deployerAddress, contractAddr loom.Address) error {
	var userDeployer UserDeployerState
	err := ctx.Get(deployerStateKey(deployerAddress), &userDeployer)
	// If key is not part of whitelisted keys then error will be logged
	if err != nil {
		if err == contract.ErrNotFound {
			return nil
		}
		return errors.Wrap(err, "Failed to Get Deployer State")
	}
	contract := udwtypes.DeployerContract{
		ContractAddress: contractAddr.MarshalPB(),
	}

	userDeployer.Contracts = append(userDeployer.Contracts, &contract)
	if err := ctx.Set(deployerStateKey(deployerAddress), &userDeployer); err != nil {
		return errors.Wrap(err, "Saving WhitelistedDeployer in whitelisted deployers state")
	}
	return nil
}

func GetTierMap(ctx contract.StaticContext) (map[TierID]Tier, error) {
	tierMap := make(map[TierID]Tier)
	for _, rangeEntry := range ctx.Range([]byte(tierKeyPrefix)) {
		var tier Tier
		if err := proto.Unmarshal(rangeEntry.Value, &tier); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal tier")
		}
		tierMap[tier.TierID] = tier

	}
	return tierMap, nil
}

//GetTierInfo standalone function to get tier information
func GetTierInfo(ctx contract.StaticContext, tierID udwtypes.TierID) (udwtypes.Tier, error) {
	var tier Tier
	err := ctx.Get(tierKey(tierID), &tier)
	if err != nil {
		return Tier{}, errors.Wrap(err, "Failed to get Tier Information")
	}
	return tier, nil
}

type ContractInfo struct {
	ContractToTierMap         map[string]TierID
	InactiveDeployerContracts map[string]bool
}

// GetContractInfo to get ContractTierMapping and InactiveDeployerContracts for TxLimiter Middleware
func GetContractInfo(ctx contract.StaticContext) (*ContractInfo, error) {
	contractToTierMap := make(map[string]TierID)
	inactiveDeployerContracts := make(map[string]bool)
	for _, rangeEntry := range ctx.Range([]byte(deployerStatePrefix)) {
		var deployer UserDeployerState
		if err := proto.Unmarshal(rangeEntry.Value, &deployer); err != nil {
			return nil, errors.Wrap(err, "unmarshal UserDeployerState")
		}
		contracts := deployer.Contracts
		for _, contract := range contracts {
			key := loom.UnmarshalAddressPB(contract.ContractAddress).String()
			if deployer.Inactive {
				inactiveDeployerContracts[key] = true
			} else {
				contractToTierMap[key] = deployer.TierID
			}
		}
	}
	return &ContractInfo{
		ContractToTierMap:         contractToTierMap,
		InactiveDeployerContracts: inactiveDeployerContracts,
	}, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&UserDeployerWhitelist{})
