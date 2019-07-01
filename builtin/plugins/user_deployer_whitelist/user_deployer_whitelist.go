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
	"github.com/pkg/errors"
)

type (
	GetUserDeployersRequest      = udwtypes.GetUserDeployersRequest
	GetUserDeployersResponse     = udwtypes.GetUserDeployersResponse
	GetDeployedContractsRequest  = udwtypes.GetDeployedContractsRequest
	GetDeployedContractsResponse = udwtypes.GetDeployedContractsResponse
	GetDeployerResponse          = dwtypes.GetDeployerResponse
	GetDeployerRequest           = dwtypes.GetDeployerRequest
	GetTierInfoRequest           = udwtypes.GetTierInfoRequest
	GetTierInfoResponse          = udwtypes.GetTierInfoResponse
	SetTierInfoRequest           = udwtypes.SetTierInfoRequest
	Deployer                     = dwtypes.Deployer
	UserDeployerState            = udwtypes.UserDeployerState
	AddUserDeployerRequest       = dwtypes.AddUserDeployerRequest
	WhitelistUserDeployerRequest = udwtypes.WhitelistUserDeployerRequest
	UserState                    = udwtypes.UserState
	InitRequest                  = udwtypes.InitRequest
	Tier                         = udwtypes.Tier
	TierID                       = udwtypes.TierID
	RemoveUserDeployerRequest    = udwtypes.RemoveUserDeployerRequest
)

var (
	// ErrrNotAuthorized indicates that a contract method failed because the caller didn't have
	// the permission to execute that method.
	ErrNotAuthorized = errors.New("[UserDeployerWhitelist] not authorized")
	// ErrInvalidRequest is a generic error that's returned when something is wrong with the
	// request message, e.g. missing or invalid fields.
	ErrInvalidRequest = errors.New("[UserDeployerWhitelist] invalid request")
	// ErrOwnerNotSpecified returned if init request does not have owner address
	ErrOwnerNotSpecified = errors.New("[UserDeployerWhitelist] owner not specified")
	// ErrFeatureFound returned if an owner try to set an existing feature
	ErrDeployerAlreadyExists = errors.New("[UserDeployerWhitelist] deployer already exists")
	// ErrDeployerDoesNotExist returned if an owner try to to remove a deployer that does not exist
	ErrDeployerDoesNotExist = errors.New("[UserDeployerWhitelist] deployer does not exist")
	// ErrInsufficientBalance returned if loom balance is unsufficient for whitelisting
	ErrInsufficientBalance = errors.New("[UserDeployerWhitelist] Insufficient Loom Balance for whitelisting")
	// ErrInvalidTier returned if Tier provided is invalid
	ErrInvalidTier = errors.New("[UserDeployerWhitelist] Invalid Tier")
	// ErrMissingTierInfo is returned if init doesnt get atleast one tier
	ErrMissingTierInfo = errors.New("[UserDeployerWhitelist] no tiers provided")
	// Invalid whitelisting fees check
	ErrInvalidWhitelistingFee = errors.New("[UserDeployerWhitelist] fee must be greater than zero")
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

func DeployerStateKey(deployer loom.Address) []byte {
	return util.PrefixKey([]byte(deployerStatePrefix), deployer.Bytes())
}

func UserStateKey(user loom.Address) []byte {
	return util.PrefixKey([]byte(userStatePrefix), user.Bytes())
}

func TierKey(tierID TierID) []byte {
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

	for _, tier := range req.TierInfo {
		fees := loom.NewBigUIntFromInt(int64(tier.Fee))
		fees.Mul(fees, div)
		tier := &Tier{
			TierID: tier.TierID,
			Fee: &types.BigUInt{
				Value: *fees,
			},
			Name:       tier.Name,
			BlockRange: tier.BlockRange,
			MaxTx:      tier.MaxTx,
		}

		if err := ctx.Set(TierKey(tier.TierID), tier); err != nil {
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
	if ctx.Has(DeployerStateKey(loom.UnmarshalAddressPB(req.DeployerAddr))) {
		return ErrDeployerAlreadyExists
	}
	var tierInfo Tier
	if err := ctx.Get(TierKey(req.TierID), &tierInfo); err != nil {
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
	if err := ctx.Get(UserStateKey(userAddr), &userState); err != nil {
		// This is taking care of boundary case also that user is whitelisting deployers for first time
		if err != contract.ErrNotFound {
			return errors.Wrap(err, "[UserDeployerWhitelist] Failed to load User State")
		}
	}
	userState.Deployers = append(userState.Deployers, req.DeployerAddr)
	if err := ctx.Set(UserStateKey(userAddr), &userState); err != nil {
		return errors.Wrap(err, "Failed to Save Deployers mapping in user state")
	}
	deployer := &UserDeployerState{
		Address: req.DeployerAddr,
		TierID:  req.TierID,
	}
	err = ctx.Set(DeployerStateKey(loom.UnmarshalAddressPB(req.DeployerAddr)), deployer)
	if err != nil {
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
	if err := ctx.Get(UserStateKey(userAddr), &userState); err != nil {
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
	if err := ctx.Set(UserStateKey(userAddr), &userState); err != nil {
		return errors.Wrap(err, "failed to Save Deployers mapping in user state")
	}
	// remove from deployerwhitelist contract
	removeUserDeployerRequest := &RemoveUserDeployerRequest{
		DeployerAddr: req.DeployerAddr,
	}
	if err := contract.CallMethod(ctx, dwAddr, "RemoveUserDeployer", removeUserDeployerRequest, nil); err != nil {
		return errors.Wrap(err, "failed to remove deployer")
	}
	// remove from current contract deployer address
	if !ctx.Has(DeployerStateKey(deployerAddr)) {
		return ErrDeployerDoesNotExist
	}
	ctx.Delete(DeployerStateKey(deployerAddr))
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
	err := ctx.Get(UserStateKey(loom.UnmarshalAddressPB(req.UserAddr)), &userState)
	if err != nil {
		if err == contract.ErrNotFound {
			return &GetUserDeployersResponse{}, nil
		}
		return nil, err
	}
	for _, deployerAddr := range userState.Deployers {
		var userDeployerState UserDeployerState
		err = ctx.Get(DeployerStateKey(loom.UnmarshalAddressPB(deployerAddr)), &userDeployerState)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to load whitelisted deployers state")
		}
		deployers = append(deployers, &userDeployerState)

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
	err := ctx.Get(DeployerStateKey(deployerAddr), &userDeployer)
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

// GetTierInfo returns the details of a specific tier.
func (uw *UserDeployerWhitelist) GetTierInfo(
	ctx contract.StaticContext, req *GetTierInfoRequest,
) (*GetTierInfoResponse, error) {
	var tier Tier
	err := ctx.Get(TierKey(req.TierID), &tier)
	if err != nil {
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
	if req.BlockRange <= 0 {
		return errors.New("[UserDeployerWhitelist] blockRange must be greater than zero")
	}
	if req.MaxTx <= 0 {
		return errors.New("[UserDeployerWhitelist] MaxTx must be greater than zero")
	}

	tier := &Tier{
		TierID:     req.TierID,
		Fee:        req.Fee,
		Name:       req.Name,
		BlockRange: req.BlockRange,
		MaxTx:      req.MaxTx,
	}
	err := ctx.Set(TierKey(req.TierID), tier)
	if err != nil {
		return errors.Wrap(err, "Failed to modify TierInfo")
	}
	return nil
}

// RecordEVMContractDeployment is called by the EVMDeployRecorderPostCommitMiddleware after an EVM
// contract is successfully deployed to record the deployment in the UserDeployerWhitelist contract.
func RecordEVMContractDeployment(ctx contract.Context, deployerAddress, contractAddr loom.Address) error {
	var userDeployer UserDeployerState
	err := ctx.Get(DeployerStateKey(deployerAddress), &userDeployer)
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
	err = ctx.Set(DeployerStateKey(deployerAddress), &userDeployer)
	if err != nil {
		return errors.Wrap(err, "Saving WhitelistedDeployer in whitelisted deployers state")
	}
	return nil
}

func GetTierMap(ctx contract.StaticContext) (map[TierID]Tier, error) {
	tierMap := make(map[TierID]Tier, 0)
	for _, rangeEntry := range ctx.Range([]byte(tierKeyPrefix)) {
		var tier Tier
		if err := proto.Unmarshal(rangeEntry.Value, &tier); err != nil {
			return nil, errors.Wrap(err, "Marshalling error")
		}
		tierMap[tier.TierID] = tier

	}
	return tierMap, nil
}

//GetTierInfo standalone function to get tier information
func GetTierInfo(ctx contract.StaticContext, tierID udwtypes.TierID) (udwtypes.Tier, error) {
	var tier Tier
	err := ctx.Get(TierKey(tierID), &tier)
	if err != nil {
		return Tier{}, errors.Wrap(err, "Failed to get Tier Information")
	}
	return tier, nil
}

// GetContractTierMapping create a map of contract to TxLimiter to be used in ContractTxLimiter
func GetContractTierMapping(ctx contract.StaticContext) (map[string]TierID, error) {
	contractToTierMap := make(map[string]TierID, 0)
	for _, rangeEntry := range ctx.Range([]byte(deployerStatePrefix)) {
		var deployer UserDeployerState
		if err := proto.Unmarshal(rangeEntry.Value, &deployer); err != nil {
			return nil, errors.Wrap(err, "unmarshal UserDeployerState")
		}
		var tier Tier
		err := ctx.Get(TierKey(deployer.TierID), &tier)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get Tier Information")
		}
		contracts := deployer.Contracts
		for _, contract := range contracts {
			key := loom.UnmarshalAddressPB(contract.ContractAddress).String()
			contractToTierMap[key] = tier.TierID
		}
	}
	return contractToTierMap, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&UserDeployerWhitelist{})
