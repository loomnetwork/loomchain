package user_deployer_whitelist

import (
	loom "github.com/loomnetwork/go-loom"
	dwtypes "github.com/loomnetwork/go-loom/builtin/types/deployer_whitelist"
	udwtypes "github.com/loomnetwork/go-loom/builtin/types/user_deployer_whitelist"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/loomnetwork/loomchain/vm"
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
	UserDeployer                 = udwtypes.Deployer
	AddUserDeployerRequest       = dwtypes.AddUserDeployerRequest
	WhitelistUserDeployerRequest = udwtypes.WhitelistUserDeployerRequest
	UserDeployers                = udwtypes.UserDeployers
	InitRequest                  = udwtypes.InitRequest
	TierInfo                     = udwtypes.TierInfo
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
	deployerStatePrefix = "dep-state"
	//This state stores deployers corresponding to the user
	userStatePrefix = "user-state"
)

var (
	modifyPerm  = []byte("modp")
	tierInfoKey = []byte("tier-info")
)

type UserDeployerWhitelist struct {
}

func DeployerStateKey(deployer loom.Address) []byte {
	return util.PrefixKey([]byte(deployerStatePrefix), deployer.Bytes())
}

func UserStateKey(user loom.Address) []byte {
	return util.PrefixKey([]byte(userStatePrefix), user.Bytes())
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

	ownerAddr := loom.UnmarshalAddressPB(req.Owner)

	// TODO: Add relevant methods to manage owner and permissions later on.
	ctx.GrantPermissionTo(ownerAddr, modifyPerm, ownerRole)

	if err := ctx.Set(tierInfoKey, req.TierInfo); err != nil {
		return err
	}

	return nil
}

// Add User Deployer
func (uw *UserDeployerWhitelist) AddUserDeployer(ctx contract.Context, req *WhitelistUserDeployerRequest) error {
	var userdeployers UserDeployers
	var tierInfo TierInfo
	var whitelistingFees *types.BigUInt
	dwAddr, err := ctx.Resolve("deployerwhitelist")
	if err != nil {
		return errors.Wrap(err, "unable to get address of deployer_whitelist")
	}
	//Check for deployer is not already whitelisted
	//Further code runs if deployer is not already whitelisted
	if ctx.Has(DeployerStateKey(loom.UnmarshalAddressPB(req.DeployerAddr))) {
		return ErrDeployerAlreadyExists
	}
	if req.TierId != udwtypes.TierID_DEFAULT {
		return ErrInvalidTier
	}
	if err := ctx.Get(tierInfoKey, &tierInfo); err != nil {
		return err
	}
	for k := range tierInfo.Tiers {
		if tierInfo.Tiers[k].Id == req.TierId {
			whitelistingFees = tierInfo.Tiers[k].Fee
		}

	}
	coinAddr, err := ctx.Resolve("coin")
	if err != nil {
		return errors.Wrap(err, "unable to get address of coin contract")
	}
	coinReq := &coin.TransferFromRequest{
		To:     coinAddr.MarshalPB(),
		From:   req.DeployerAddr,
		Amount: whitelistingFees,
	}
	if err := contract.CallMethod(ctx, coinAddr, "TransferFrom", coinReq, nil); err != nil {
		if err == coin.ErrSenderBalanceTooLow {
			return ErrInsufficientBalance
		}
		return errors.Wrap(err, "Unable to transfer coin")
	}
	userAddr := ctx.Message().Sender
	adddeprequest := &dwtypes.AddUserDeployerRequest{
		DeployerAddr: req.DeployerAddr,
	}
	if err := contract.CallMethod(ctx, dwAddr, "AddUserDeployer", adddeprequest, nil); err != nil {
		return errors.Wrap(err, "Adding User Deployer")
	}

	err = ctx.Get(UserStateKey(userAddr), &userdeployers)
	if err != nil {
		//This is taking care of boundary cases that user is whitelisting deployers for first time
		if err != contract.ErrNotFound {
			return errors.Wrap(err, "[UserDeployerWhitelist] Failed to load User State")
		}
	}
	userdeployers.Deployers = append(userdeployers.Deployers, req.DeployerAddr)
	userdeployer := &UserDeployers{
		Deployers: userdeployers.Deployers,
	}
	err = ctx.Set(UserStateKey(userAddr), userdeployer)
	if err != nil {
		return errors.Wrap(err, "Failed to Save Deployers mapping in user state")
	}
	deployerReq := &GetDeployerRequest{
		DeployerAddr: req.DeployerAddr,
	}
	var getDeployerResponse GetDeployerResponse
	if err := contract.StaticCallMethod(ctx, dwAddr,
		"GetDeployer", deployerReq, &getDeployerResponse); err != nil {
		return err
	}
	deployer := &UserDeployer{
		Address: getDeployerResponse.Deployer.GetAddress(),
		Flags:   getDeployerResponse.Deployer.GetFlags(),
	}
	//Storing Full Deployer object corresponding to Deployer Key
	err = ctx.Set(DeployerStateKey(loom.UnmarshalAddressPB(req.DeployerAddr)), deployer)
	if err != nil {
		return errors.Wrap(err, "Failed to Save WhitelistedDeployer in whitelisted deployers state")
	}
	return nil

}

// GetUserDeployers returns whitelisted deployers corresponding to specific user
func (uw *UserDeployerWhitelist) GetUserDeployers(ctx contract.StaticContext,
	req *GetUserDeployersRequest) (*GetUserDeployersResponse, error) {
	var userDeployers UserDeployers
	err := ctx.Get(UserStateKey(ctx.Message().Sender), &userDeployers)
	if err != nil {
		return nil, err
	}
	deployers := []*Deployer{}
	for _, deployerAddr := range userDeployers.Deployers {
		var userDeployer UserDeployer
		err = ctx.Get(DeployerStateKey(loom.UnmarshalAddressPB(deployerAddr)), &userDeployer)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to load whitelisted deployers state")
		}
		deployers = append(deployers, &Deployer{
			Address: deployerAddr,
			Flags:   userDeployer.Flags,
		})
	}
	return &GetUserDeployersResponse{
		Deployers: deployers,
	}, nil
}

// GetDeployedContracts return contract addresses deployed by particular deployer
func (uw *UserDeployerWhitelist) GetDeployedContracts(ctx contract.StaticContext,
	req *GetDeployedContractsRequest) (*GetDeployedContractsResponse, error) {
	if req.DeployerAddr == nil {
		return nil, ErrInvalidRequest
	}
	deployerAddr := loom.UnmarshalAddressPB(req.DeployerAddr)
	var userDeployer UserDeployer
	err := ctx.Get(DeployerStateKey(deployerAddr), &userDeployer)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to load whitelisted deployers state")
	}
	contractAddresses := []*types.Address{}
	for _, contract := range userDeployer.Contracts {
		contractAddresses = append(contractAddresses, contract.ContractAddress)
	}
	return &GetDeployedContractsResponse{
		ContractAddresses: contractAddresses,
	}, nil
}

// RecordContractDeployment will record contract deployer address, newly deployed contract and on which vm it is deployed.
// If key is not part of whitelisted key, Ignore.
func RecordContractDeployment(ctx contract.Context, deployerAddress loom.Address, contractAddr loom.Address, vmType vm.VMType) error {
	var userDeployer UserDeployer
	err := ctx.Get(DeployerStateKey(deployerAddress), &userDeployer)
	// If key is not part of whitelisted keys then error will be logged
	if err != nil {
	        ctx.Logger().Error("Deployer not whitelisted","error", err)
	}
	contract := udwtypes.Contract{
		ContractAddress: contractAddr.MarshalPB(),
		VmType:          vmType,
	}
	userDeployer.Contracts = append(userDeployer.Contracts, &contract)
	err = ctx.Set(DeployerStateKey(deployerAddress), &userDeployer)
	if err != nil {
		return errors.Wrap(err, "Saving WhitelistedDeployer in whitelisted deployers state")
	}
	return nil

}

var Contract plugin.Contract = contract.MakePluginContract(&UserDeployerWhitelist{})
