package user_deployer_whitelist

import (
	"github.com/golang/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
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
	GetUserDeployersRequest      = dwtypes.ListDeployersRequest
	GetUserDeployersResponse     = dwtypes.ListDeployersResponse
	Deployer                     = dwtypes.Deployer
	AddUserDeployerRequest       = dwtypes.AddUserDeployerRequest
	WhitelistUserDeployerRequest = udwtypes.WhitelistUserDeployerRequest
	InitRequest                  = udwtypes.InitRequest
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
	ownerRole           = "owner"
	deployerPrefix      = "dep"
	deployerStatePrefix = "dep-state"
	userStatePrefix     = "user-state"
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
	coinAddr, err := ctx.Resolve("coin")
	if err != nil {
		return errors.Wrap(err, "address of coin contract")
	}
	request := &ctypes.BalanceOfRequest{
		Owner: req.DeployerAddr,
	}
	tier := req.Tier
	if tier != 0 {
		return ErrInvalidTier
	}
	var response coin.BalanceOfResponse
	err = contract.StaticCallMethod(ctx, coinAddr, "BalanceOf", request, &response)
	if err != nil {
		return err
	}
	if (response.Balance != nil) && (response.Balance.Value.Int.Cmp(whitelistingfees) >= 0) {
		dwAddr, err := ctx.Resolve("deployerwhitelist")
		if err != nil {
			return errors.Wrap(err, "address of deployer_whitelist")
		}
		adddeprequest := &dwtypes.AddUserDeployerRequest{
			DeployerAddr: req.DeployerAddr,
		}
		if err := contract.CallMethod(ctx, dwAddr, "AddUserDeployer", adddeprequest, nil); err != nil {
			return errors.Wrap(err, "Adding User Deployer")
		}
		coinReq := &coin.TransferFromRequest{
			To:     coinAddr.MarshalPB(),
			From:   req.DeployerAddr,
			Amount: &types.BigUInt{Value: *loom.NewBigUInt(whitelistingfees)},
		}
		if err := contract.CallMethod(ctx, coinAddr, "TransferFrom", coinReq, nil); err != nil {
			return errors.Wrap(err, "Transferring whitelisting fees to coin contract")
		}
		return nil

	}
	return ErrInsufficientBalance

}

// GetUserDeployers returns whitelisted deployer with addresses and flags
func (uw *UserDeployerWhitelist) GetUserDeployers(ctx contract.StaticContext, req *GetUserDeployersRequest) (*GetUserDeployersResponse, error) {
	deployerRange := ctx.Range([]byte(deployerPrefix))
	deployers := []*Deployer{}
	for _, m := range deployerRange {
		var deployer Deployer
		if err := proto.Unmarshal(m.Value, &deployer); err != nil {
			return nil, errors.Wrapf(err, "unmarshal deployer %x", m.Key)
		}
		deployers = append(deployers, &deployer)
	}

	return &GetUserDeployersResponse{
		Deployers: deployers,
	}, nil
}

// GetDeployedContracts return contract addresses deployed by particular deployer
// func (uw *UserDeployerWhitelist) GetDeployedContracts(ctx contract.StaticContext, req *GetDeployedContractsRequest) (*GetDeployedContractsResponse, error) {
// 	return nil, nil
// }

// RecordContractDeployment will record contract deployer address, newly deployed contract and on which vm it is deployed.
// If key is not part of whitelisted key, Ignore.
func RecordContractDeployment(ctx contract.Context, deployerAddress loom.Address, contractAddr loom.Address, vmType vm.VMType) error {
	panic("not implemented")
}

var Contract plugin.Contract = contract.MakePluginContract(&UserDeployerWhitelist{})
