package user_deployer_whitelist

import (
	"github.com/golang/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	dwtypes "github.com/loomnetwork/go-loom/builtin/types/deployer_whitelist"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/pkg/errors"
)

type (
	GetUserDeployersRequest  = dwtypes.ListDeployersRequest
	GetUserDeployersResponse = dwtypes.ListDeployersResponse
	Deployer                 = dwtypes.Deployer

	AddUserDeployerRequest = dwtypes.AddUserDeployerRequest
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
)

const (
	ownerRole      = "owner"
	deployerPrefix = "dep"
)

type UserDeployerWhitelist struct {
}

func (uw *UserDeployerWhitelist) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "user-deployer-whitelist",
		Version: "1.0.0",
	}, nil
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

func RecordContractDeployment(ctx contract.Context, deployerAddress loom.Address, contractAddr loom.Address) error {
	return nil
}

var Contract plugin.Contract = contract.MakePluginContract(&UserDeployerWhitelist{})
