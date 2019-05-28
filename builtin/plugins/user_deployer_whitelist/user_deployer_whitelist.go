package user_deployer_whitelist

import (
	"bytes"
	"encoding/binary"

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
	Deployer                     = dwtypes.Deployer
	UserDeployerState            = udwtypes.UserDeployerState
	AddUserDeployerRequest       = dwtypes.AddUserDeployerRequest
	WhitelistUserDeployerRequest = udwtypes.WhitelistUserDeployerRequest
	UserState                    = udwtypes.UserState
	InitRequest                  = udwtypes.InitRequest
	Tier                         = udwtypes.Tier
	TierID                       = udwtypes.TierID
	Address                      = types.Address
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
			Name: tier.Name,
		}

		if err := ctx.Set(TierKey(tier.TierID), tier); err != nil {
			return err
		}
	}
	return nil
}

/* Add User Deployer - Adds Deployer in UserState
This method will be called by User who wants to whitelist deployer, takes deployer Address as input
Afer adding a deployer User state will change, it will contains deployer information - (
address + TierId) corresponding to UserKey
After adding a deployer, deployer state will also change it contains mapping of deployer key corresponding to
deployerinfo
If Loomcoin balance of user is >= whitelisting fees, deployed is whitelisted otherwise error is returned.
*/

func (uw *UserDeployerWhitelist) AddUserDeployer(
	ctx contract.Context, req *WhitelistUserDeployerRequest) error {
	var userState UserState
	var tierInfo Tier
	var whitelistingFees *types.BigUInt
	if req.DeployerAddr == nil {
		return ErrInvalidRequest
	}
	if req.TierID != udwtypes.TierID_DEFAULT {
		return ErrInvalidTier
	}
	dwAddr, err := ctx.Resolve("deployerwhitelist")
	if err != nil {
		return errors.Wrap(err, "unable to get address of deployer_whitelist")
	}
	coinAddr, err := ctx.Resolve("coin")
	if err != nil {
		return errors.Wrap(err, "unable to get address of coin contract")
	}
	//Check for deployer is not already whitelisted
	//Further code runs if deployer is not already whitelisted
	if ctx.Has(DeployerStateKey(loom.UnmarshalAddressPB(req.DeployerAddr))) {
		return ErrDeployerAlreadyExists
	}
	if err := ctx.Get(TierKey(req.TierID), &tierInfo); err != nil {
		return err
	}
	whitelistingFees = &types.BigUInt{Value: tierInfo.Fee.Value}
	userAddr := ctx.Message().Sender
	//Amount equal to whitelisting fees is debited from users loomcoin balance,
	// whitelisting fees is transferred to user deployer whitelist contract
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
	//Retrieving Deployer Address and Corresponding Flags after Deployer Addition
	if err := contract.CallMethod(ctx, dwAddr, "AddUserDeployer", adddeprequest, nil); err != nil {
		return errors.Wrap(err, "failed to whitelist deployer")
	}
	if err := ctx.Get(UserStateKey(userAddr), &userState); err != nil {
		//This is taking care of boundary case also that user is whitelisting deployers for first time
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
		TierId:  req.TierID,
	}
	//Storing Full Deployer object corresponding to Deployer Key - Deployer State
	err = ctx.Set(DeployerStateKey(loom.UnmarshalAddressPB(req.DeployerAddr)), deployer)
	if err != nil {
		return errors.Wrap(err, "Failed to Save WhitelistedDeployer in whitelisted deployers state")
	}
	return nil

}

// GetUserDeployers returns whitelisted deployers corresponding to specific user
//Gets Deployer data from user state
func (uw *UserDeployerWhitelist) GetUserDeployers(
	ctx contract.StaticContext, req *GetUserDeployersRequest,
) (*GetUserDeployersResponse, error) {
	var userState UserState
	if req.UserAddr == nil {
		return nil, ErrInvalidRequest
	}
	err := ctx.Get(UserStateKey(loom.UnmarshalAddressPB(req.UserAddr)), &userState)
	if err != nil {
		return nil, err
	}
	deployers := []*UserDeployerState{}
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

// GetDeployedContracts return contract addresses deployed by particular deployer
//Gets Deployed Contracts from deployer state corresponding to deployer key
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
		return nil, errors.Wrap(err, "Failed to load whitelisted deployers state")
	}
	return &GetDeployedContractsResponse{
		ContractAddresses: userDeployer.Contracts,
	}, nil
}

// RecordEVMContractDeployment will record contract deployer address,newly deployed contract.
// If key is not part of whitelisted key, Ignore.
// Stores deployed contract information in Deployer State corresponding to deployer key
func RecordEVMContractDeployment(ctx contract.Context, deployerAddress loom.Address, contractAddr loom.Address) error {
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

var Contract plugin.Contract = contract.MakePluginContract(&UserDeployerWhitelist{})
