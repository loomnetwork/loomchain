package deployer_whitelist

import (
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	dwtypes "github.com/loomnetwork/go-loom/builtin/types/deployer_whitelist"
	udwtypes "github.com/loomnetwork/go-loom/builtin/types/user_deployer_whitelist"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/util"
	"github.com/pkg/errors"
)

type (
	InitRequest               = dwtypes.InitRequest
	ListDeployersRequest      = dwtypes.ListDeployersRequest
	ListDeployersResponse     = dwtypes.ListDeployersResponse
	GetDeployerRequest        = dwtypes.GetDeployerRequest
	GetDeployerResponse       = dwtypes.GetDeployerResponse
	AddDeployerRequest        = dwtypes.AddDeployerRequest
	AddDeployerResponse       = dwtypes.AddDeployerResponse
	RemoveDeployerRequest     = dwtypes.RemoveDeployerRequest
	RemoveDeployerResponse    = dwtypes.RemoveDeployerResponse
	Deployer                  = dwtypes.Deployer
	AddUserDeployerRequest    = dwtypes.AddUserDeployerRequest
	RemoveUserDeployerRequest = udwtypes.RemoveUserDeployerRequest
)

const (
	// AllowEVMDeployFlag indicates that a deployer is permitted to deploy EVM contract.
	AllowEVMDeployFlag = dwtypes.Flags_EVM
	// AllowGoDeployFlag indicates that a deployer is permitted to deploy GO contract.
	AllowGoDeployFlag = dwtypes.Flags_GO
	// AllowMigrationFlag indicates that a deployer is permitted to migrate GO contract.
	AllowMigrationFlag = dwtypes.Flags_MIGRATION
)

var (
	// ErrrNotAuthorized indicates that a contract method failed because the caller didn't have
	// the permission to execute that method.
	ErrNotAuthorized = errors.New("[DeployerWhitelist] not authorized")
	// ErrInvalidRequest is a generic error that's returned when something is wrong with the
	// request message, e.g. missing or invalid fields.
	ErrInvalidRequest = errors.New("[DeployerWhitelist] invalid request")
	// ErrOwnerNotSpecified returned if init request does not have owner address
	ErrOwnerNotSpecified = errors.New("[DeployerWhitelist] owner not specified")
	// ErrFeatureFound returned if an owner try to set an existing feature
	ErrDeployerAlreadyExists = errors.New("[DeployerWhitelist] deployer already exists")
	// ErrDeployerDoesNotExist returned if an owner try to to remove a deployer that does not exist
	ErrDeployerDoesNotExist = errors.New("[DeployerWhitelist] deployer does not exist")
)

const (
	ownerRole      = "owner"
	deployerPrefix = "dep"
)

var (
	modifyPerm = []byte("modp")
)

func deployerKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte(deployerPrefix), addr.Bytes())
}

type DeployerWhitelist struct {
}

func (dw *DeployerWhitelist) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "deployerwhitelist",
		Version: "1.0.0",
	}, nil
}

func (dw *DeployerWhitelist) Init(ctx contract.Context, req *InitRequest) error {
	if req.Owner == nil {
		return ErrOwnerNotSpecified
	}
	ownerAddr := loom.UnmarshalAddressPB(req.Owner)
	ctx.GrantPermissionTo(ownerAddr, modifyPerm, ownerRole)

	//add owner to deployer list
	flags := PackFlags(uint32(AllowEVMDeployFlag), uint32(AllowGoDeployFlag), uint32(AllowMigrationFlag))
	deployer := &Deployer{
		Address: ownerAddr.MarshalPB(),
		Flags:   flags,
	}
	if err := ctx.Set(deployerKey(ownerAddr), deployer); err != nil {
		return err
	}

	//add init deployers to deployer list
	for _, d := range req.Deployers {
		addr := loom.UnmarshalAddressPB(d.Address)
		if err := ctx.Set(deployerKey(addr), d); err != nil {
			return err
		}
	}
	return nil
}

func (dw *DeployerWhitelist) AddUserDeployer(ctx contract.Context, req *AddUserDeployerRequest) error {
	userWhitelistContract, err := ctx.Resolve("user-deployer-whitelist")
	if err != nil {
		return errors.Wrap(err, "unable to resolve user_deployer_whitelist contract")
	}

	if ctx.Message().Sender.Compare(userWhitelistContract) != 0 {
		return ErrNotAuthorized
	}

	if req.DeployerAddr == nil {
		return ErrInvalidRequest
	}

	deployerAddr := loom.UnmarshalAddressPB(req.DeployerAddr)

	if ctx.Has(deployerKey(deployerAddr)) {
		return ErrDeployerAlreadyExists
	}

	deployer := &Deployer{
		Address: deployerAddr.MarshalPB(),
		Flags:   PackFlags(uint32(AllowEVMDeployFlag)),
	}

	return ctx.Set(deployerKey(deployerAddr), deployer)
}

func (dw *DeployerWhitelist) RemoveUserDeployer(ctx contract.Context, req *RemoveUserDeployerRequest) error {
	if req.DeployerAddr == nil {
		return ErrInvalidRequest
	}

	//check if authorized
	userWhitelistContract, err := ctx.Resolve("user-deployer-whitelist")
	if err != nil {
		return errors.Wrap(err, "unable to resolve user_deployer_whitelist contract")
	}

	if ctx.Message().Sender.Compare(userWhitelistContract) != 0 {
		return ErrNotAuthorized
	}

	// remove deployer
	deployerAddr := loom.UnmarshalAddressPB(req.DeployerAddr)

	if !ctx.Has(deployerKey(deployerAddr)) {
		return ErrDeployerDoesNotExist
	}
	ctx.Delete(deployerKey(deployerAddr))

	return nil
}

// AddDeployer
func (dw *DeployerWhitelist) AddDeployer(ctx contract.Context, req *AddDeployerRequest) error {
	if ok, _ := ctx.HasPermission(modifyPerm, []string{ownerRole}); !ok {
		return ErrNotAuthorized
	}

	if req.DeployerAddr == nil || req.Flags <= 0 {
		return ErrInvalidRequest
	}

	deployerAddr := loom.UnmarshalAddressPB(req.DeployerAddr)

	if ctx.Has(deployerKey(deployerAddr)) {
		return ErrDeployerAlreadyExists
	}

	deployer := &Deployer{
		Address: deployerAddr.MarshalPB(),
		Flags:   req.Flags,
	}

	return ctx.Set(deployerKey(deployerAddr), deployer)
}

// GetDeployer
func (dw *DeployerWhitelist) GetDeployer(ctx contract.StaticContext, req *GetDeployerRequest) (*GetDeployerResponse, error) {
	if req.DeployerAddr == nil {
		return nil, ErrInvalidRequest
	}

	deployerAddr := loom.UnmarshalAddressPB(req.DeployerAddr)

	deployer, err := GetDeployer(ctx, deployerAddr)
	if err != nil {
		return nil, err
	}

	return &GetDeployerResponse{
		Deployer: deployer,
	}, nil
}

// RemoveDeployer
func (dw *DeployerWhitelist) RemoveDeployer(ctx contract.Context, req *RemoveDeployerRequest) error {
	if req.DeployerAddr == nil {
		return ErrInvalidRequest
	}

	if ok, _ := ctx.HasPermission(modifyPerm, []string{ownerRole}); !ok {
		return ErrNotAuthorized
	}

	deployerAddr := loom.UnmarshalAddressPB(req.DeployerAddr)

	if !ctx.Has(deployerKey(deployerAddr)) {
		return ErrDeployerDoesNotExist
	}
	ctx.Delete(deployerKey(deployerAddr))

	return nil
}

// ListDeployers
func (dw *DeployerWhitelist) ListDeployers(ctx contract.StaticContext, req *ListDeployersRequest) (*ListDeployersResponse, error) {
	deployerRange := ctx.Range([]byte(deployerPrefix))
	deployers := []*Deployer{}
	for _, m := range deployerRange {
		var deployer Deployer
		if err := proto.Unmarshal(m.Value, &deployer); err != nil {
			return nil, errors.Wrapf(err, "unmarshal deployer %x", m.Key)
		}
		deployers = append(deployers, &deployer)
	}

	return &ListDeployersResponse{
		Deployers: deployers,
	}, nil
}

// GetDeployer is called by DeployerWhitelist middleware to retrieve deployer's permission
func GetDeployer(ctx contract.StaticContext, deployerAddr loom.Address) (*Deployer, error) {
	var deployer Deployer

	err := ctx.Get(deployerKey(deployerAddr), &deployer)
	if err == contract.ErrNotFound {
		return &Deployer{
			Address: deployerAddr.MarshalPB(),
		}, nil
	}
	return &deployer, err
}

func PackFlags(flags ...uint32) uint32 {
	packedFlags := uint32(0)
	for _, flag := range flags {
		packedFlags = packedFlags | flag
	}
	return packedFlags
}

func UnpackFlags(flags uint32) []uint32 {
	allFlags := []uint32{}
	for i := uint(0); i < 32; i++ {
		f := uint32(1) << i
		if (f & flags) != 0 {
			allFlags = append(allFlags, f)
		}
	}
	return allFlags
}

func IsFlagSet(flags uint32, flag uint32) bool {
	return (flags & flag) != 0
}

var Contract plugin.Contract = contract.MakePluginContract(&DeployerWhitelist{})
