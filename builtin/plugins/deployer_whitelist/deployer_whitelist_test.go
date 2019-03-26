package deployer_whitelist

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/stretchr/testify/suite"
)

var (
	chainId = "default"
	addr1   = loom.MustParseAddress("default:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2   = loom.MustParseAddress("default:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	addr3   = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	addr4   = loom.MustParseAddress("default:0x46ecd1f7261e1f4c684e297be3edf03b825e01c4")
	addr5   = loom.MustParseAddress("default:0x76ecd1f7261fcf4c684e297be3edf03b825e01c4")
)

type DeployerWhitelistTestSuite struct {
	suite.Suite
}

func (dw *DeployerWhitelistTestSuite) SetupTest() {
}

func TestDeployerWhitelistTestSuite(t *testing.T) {
	suite.Run(t, new(DeployerWhitelistTestSuite))
}

func formatJSON(pb proto.Message) (string, error) {
	marshaler := jsonpb.Marshaler{
		Indent:       "  ",
		EmitDefaults: true,
	}
	return marshaler.MarshalToString(pb)
}

func (dw *DeployerWhitelistTestSuite) TestDeployerWhitelistContract() {
	require := dw.Require()

	pctx := plugin.CreateFakeContext(addr1, addr1).WithBlock(loom.BlockHeader{
		ChainID: chainId,
		Time:    time.Now().Unix(),
	})
	ctx := contractpb.WrapPluginContext(pctx)

	//setup deployer whitelist contract
	deployerContract := &DeployerWhitelist{}
	err := deployerContract.Init(ctx, &InitRequest{
		Owner: addr1.MarshalPB(),
		Deployers: []*Deployer{
			&Deployer{
				Address:    addr5.MarshalPB(),
				Permission: AllowEVMDeploy,
			},
		},
	})
	require.NoError(err)

	// test ListDeployers
	list, err := deployerContract.ListDeployers(ctx, &ListDeployersRequest{})
	require.NoError(err)
	require.Equal(2, len(list.Deployers))

	// test GetDeployer
	get, err := deployerContract.GetDeployer(ctx, &GetDeployerRequest{
		DeployerAddr: addr1.MarshalPB(),
	})
	require.NoError(err)
	gotAddr := loom.UnmarshalAddressPB(get.Deployer.Address)
	require.Equal(addr1.Local.String(), gotAddr.Local.String())
	require.Equal(AllowAnyDeploy, get.Deployer.Permission)

	// test AddDeployer
	err = deployerContract.AddDeployer(ctx, &AddDeployerRequest{
		DeployerAddr: addr2.MarshalPB(),
		Permission:   AllowGoDeploy,
	})
	require.NoError(err)

	list, err = deployerContract.ListDeployers(ctx, &ListDeployersRequest{})
	require.NoError(err)
	require.Equal(3, len(list.Deployers))

	err = deployerContract.AddDeployer(ctx, &AddDeployerRequest{
		DeployerAddr: addr3.MarshalPB(),
		Permission:   AllowEVMDeploy,
	})
	require.NoError(err)

	list, err = deployerContract.ListDeployers(ctx, &ListDeployersRequest{})
	require.NoError(err)
	require.Equal(4, len(list.Deployers))

	//Check permission
	get, err = deployerContract.GetDeployer(ctx, &GetDeployerRequest{
		DeployerAddr: addr2.MarshalPB(),
	})
	require.NoError(err)
	gotAddr = loom.UnmarshalAddressPB(get.Deployer.Address)
	require.Equal(addr2.Local.String(), gotAddr.Local.String())
	require.Equal(AllowGoDeploy, get.Deployer.Permission)

	get, err = deployerContract.GetDeployer(ctx, &GetDeployerRequest{
		DeployerAddr: addr3.MarshalPB(),
	})
	require.NoError(err)
	gotAddr = loom.UnmarshalAddressPB(get.Deployer.Address)
	require.Equal(addr3.Local.String(), gotAddr.Local.String())
	require.Equal(AllowEVMDeploy, get.Deployer.Permission)

	// get a deployer that does not exists
	get, err = deployerContract.GetDeployer(ctx, &GetDeployerRequest{
		DeployerAddr: addr4.MarshalPB(),
	})
	require.Equal(AllowNoneDeploy, get.Deployer.Permission)

	//test RemoveDeployer
	err = deployerContract.RemoveDeployer(ctx, &RemoveDeployerRequest{
		DeployerAddr: addr3.MarshalPB(),
	})
	require.NoError(err)

	err = deployerContract.RemoveDeployer(ctx, &RemoveDeployerRequest{
		DeployerAddr: addr2.MarshalPB(),
	})
	require.NoError(err)

	err = deployerContract.RemoveDeployer(ctx, &RemoveDeployerRequest{
		DeployerAddr: addr4.MarshalPB(),
	})
	require.Equal(ErrDeployerDoesNotExist, err)

	// list must have only owner left
	list, err = deployerContract.ListDeployers(ctx, &ListDeployersRequest{})
	require.NoError(err)
	require.Equal(2, len(list.Deployers))
}

func (dw *DeployerWhitelistTestSuite) TestPermission() {
	require := dw.Require()

	pctx := plugin.CreateFakeContext(addr1, addr1).WithBlock(loom.BlockHeader{
		ChainID: chainId,
		Time:    time.Now().Unix(),
	})
	ctx := contractpb.WrapPluginContext(pctx)

	//setup deployer whitelist contract
	deployerContract := &DeployerWhitelist{}
	err := deployerContract.Init(ctx, &InitRequest{
		Owner: addr1.MarshalPB(),
	})
	require.NoError(err)

	// test ListDeployers
	list, err := deployerContract.ListDeployers(ctx, &ListDeployersRequest{})
	require.NoError(err)
	require.Equal(1, len(list.Deployers))

	// test GetDeployer
	get, err := deployerContract.GetDeployer(ctx, &GetDeployerRequest{
		DeployerAddr: addr1.MarshalPB(),
	})
	require.NoError(err)
	gotAddr := loom.UnmarshalAddressPB(get.Deployer.Address)
	require.Equal(addr1.Local.String(), gotAddr.Local.String())
	require.Equal(AllowAnyDeploy, get.Deployer.Permission)

	// test AddDeployer
	err = deployerContract.AddDeployer(ctx, &AddDeployerRequest{
		DeployerAddr: addr2.MarshalPB(),
		Permission:   AllowGoDeploy,
	})
	require.NoError(err)

	err = deployerContract.AddDeployer(ctx, &AddDeployerRequest{
		DeployerAddr: addr3.MarshalPB(),
		Permission:   AllowEVMDeploy,
	})
	require.NoError(err)

	//permission test
	err = deployerContract.AddDeployer(contractpb.WrapPluginContext(pctx.WithSender(addr3)), &AddDeployerRequest{
		DeployerAddr: addr4.MarshalPB(),
		Permission:   AllowEVMDeploy,
	})
	require.Equal(ErrNotAuthorized, err)

	//permission test
	err = deployerContract.RemoveDeployer(contractpb.WrapPluginContext(pctx.WithSender(addr3)), &RemoveDeployerRequest{
		DeployerAddr: addr3.MarshalPB(),
	})
	require.Equal(ErrNotAuthorized, err)
}
