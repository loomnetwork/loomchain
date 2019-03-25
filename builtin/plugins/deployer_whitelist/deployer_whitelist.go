package deployer_whitelist

import (
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	dwtypes "github.com/loomnetwork/go-loom/builtin/types/deployer_whitelist"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/util"
	"github.com/pkg/errors"
)

type (
	InitRequest            = dwtypes.InitRequest
	ListDeployersRequest   = dwtypes.ListDeployersRequest
	ListDeployersResponse  = dwtypes.ListDeployersResponse
	GetDeployerRequest     = dwtypes.GetDeployerRequest
	GetDeployerResponse    = dwtypes.GetDeployerResponse
	AddDeployerRequest     = dwtypes.AddDeployerRequest
	AddDeployerResponse    = dwtypes.AddDeployerResponse
	RemoveDeployerRequest  = dwtypes.RemoveDeployerRequest
	RemoveDeployerResponse = dwtypes.RemoveDeployerResponse
	Deployer               = dwtypes.Deployer
)

const (
	// EVMDeployer permission indicates that a deployer is permitted to deploy EVM contract.
	EVMDeployer = dwtypes.DeployPermission_EVM
	// GODeployer permission indicates that a deployer is permitted to deploy GO Contract.
	GODeployer = dwtypes.DeployPermission_GO
	// BOTHDeployer permission indicates that a deployer is permitted to deploy both GO and EVM contracts.
	BOTHDeployer = dwtypes.DeployPermission_BOTH
)

var (
	// ErrrNotAuthorized indicates that a contract method failed because the caller didn't have
	// the permission to execute that method.
	ErrNotAuthorized = errors.New("[DeployerWhitelist] not authorized")
	// ErrInvalidRequest is a generic error that's returned when something is wrong with the
	// request message, e.g. missing or invalid fields.
	ErrInvalidRequest = errors.New("[Deployer-Whitelist] invalid request")
	// ErrOwnerNotSpecified returned if init request does not have owner address
	ErrOwnerNotSpecified = errors.New("[Deployer-Whitelist] owner not specified")
	// ErrFeatureFound returned if an owner try to set an existing feature
	ErrDeployerAlreadyExists = errors.New("[DeployerWhitelist] deployer already exists")
	// ErrDeployerDoesNotExist returned if an owner try to to remove a deployer that does not exist
	ErrDeployerDoesNotExist = errors.New("[DeployerWhitelist] deployer does not exist")
)

const (
	ownerRole      = "owner"
	deployerPrefix = "deployer"
)

var (
	addDeployerPerm    = []byte("addp")
	removeDeployerPerm = []byte("removep")
)

func deployerKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte("deployer"), addr.Bytes())
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
	ctx.GrantPermissionTo(ownerAddr, addDeployerPerm, ownerRole)
	ctx.GrantPermissionTo(ownerAddr, removeDeployerPerm, ownerRole)

	//add owner to deployer list
	deployer := &Deployer{
		Address:    ownerAddr.MarshalPB(),
		Permission: BOTHDeployer,
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

// AddDeployer
func (dw *DeployerWhitelist) AddDeployer(ctx contract.Context, req *AddDeployerRequest) error {
	if ok, _ := ctx.HasPermission(removeDeployerPerm, []string{ownerRole}); !ok {
		return ErrNotAuthorized
	}

	deployerAddr := loom.UnmarshalAddressPB(req.DeployerAddr)

	if ctx.Has(deployerKey(deployerAddr)) {
		return ErrDeployerAlreadyExists
	}

	deployer := &Deployer{
		Address:    deployerAddr.MarshalPB(),
		Permission: req.Permission,
	}

	return ctx.Set(deployerKey(deployerAddr), deployer)
}

// GetDeployer
func (dw *DeployerWhitelist) GetDeployer(ctx contract.StaticContext, req *GetDeployerRequest) (*GetDeployerResponse, error) {
	deployerAddr := loom.UnmarshalAddressPB(req.DeployerAddr)

	if !ctx.Has(deployerKey(deployerAddr)) {
		return nil, ErrDeployerDoesNotExist
	}

	var deployer Deployer
	if err := ctx.Get(deployerKey(deployerAddr), &deployer); err != nil {
		return nil, err
	}
	res := &GetDeployerResponse{
		Deployer: &deployer,
	}
	return res, nil
}

// RemoveDeployer
func (dw *DeployerWhitelist) RemoveDeployer(ctx contract.Context, req *RemoveDeployerRequest) error {
	if ok, _ := ctx.HasPermission(removeDeployerPerm, []string{ownerRole}); !ok {
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
			return nil, errors.Wrapf(err, "unmarshal deployer %s", string(m.Key))
		}
		deployers = append(deployers, &deployer)
	}

	res := &ListDeployersResponse{
		Deployers: deployers,
	}

	return res, nil
}

func GetDeployer(ctx contract.Context, deployerAddr loom.Address) (*Deployer, error) {
	if !ctx.Has(deployerKey(deployerAddr)) {
		return nil, ErrDeployerDoesNotExist
	}
	var deployer Deployer
	if err := ctx.Get(deployerKey(deployerAddr), &deployer); err != nil {
		return nil, err
	}
	return &deployer, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&DeployerWhitelist{})
