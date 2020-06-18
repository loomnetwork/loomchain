package deployer_whitelist

import (
	"reflect"
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

	//Set up the deployer whitelist contract
	deployerContract := &DeployerWhitelist{}
	err := deployerContract.Init(ctx, &InitRequest{
		Owner: addr1.MarshalPB(),
		Deployers: []*Deployer{
			&Deployer{
				Address: addr5.MarshalPB(),
				Flags:   uint32(AllowEVMDeployFlag),
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
	require.Equal(true, IsFlagSet(get.Deployer.Flags, uint32(AllowGoDeployFlag)))
	require.Equal(true, IsFlagSet(get.Deployer.Flags, uint32(AllowEVMDeployFlag)))
	require.Equal(true, IsFlagSet(get.Deployer.Flags, uint32(AllowMigrationFlag)))

	// test AddDeployer
	err = deployerContract.AddDeployer(ctx, &AddDeployerRequest{
		DeployerAddr: addr2.MarshalPB(),
		Flags:        uint32(AllowGoDeployFlag),
	})
	require.NoError(err)

	list, err = deployerContract.ListDeployers(ctx, &ListDeployersRequest{})
	require.NoError(err)
	require.Equal(3, len(list.Deployers))

	err = deployerContract.AddDeployer(ctx, &AddDeployerRequest{
		DeployerAddr: addr3.MarshalPB(),
		Flags:        uint32(AllowEVMDeployFlag),
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
	require.Equal(true, IsFlagSet(get.Deployer.Flags, uint32(AllowGoDeployFlag)))

	get, err = deployerContract.GetDeployer(ctx, &GetDeployerRequest{
		DeployerAddr: addr3.MarshalPB(),
	})
	require.NoError(err)
	gotAddr = loom.UnmarshalAddressPB(get.Deployer.Address)
	require.Equal(addr3.Local.String(), gotAddr.Local.String())
	require.Equal(true, IsFlagSet(get.Deployer.Flags, uint32(AllowEVMDeployFlag)))

	// get a deployer that does not exist
	get, err = deployerContract.GetDeployer(ctx, &GetDeployerRequest{
		DeployerAddr: addr4.MarshalPB(),
	})
	require.Equal(uint32(0), get.Deployer.Flags)

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

	// the list must have only the owner left
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

	// Set up the deployer whitelist contract
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
	require.Equal(true, IsFlagSet(get.Deployer.Flags, uint32(AllowEVMDeployFlag)))
	require.Equal(true, IsFlagSet(get.Deployer.Flags, uint32(AllowGoDeployFlag)))

	// test AddDeployer
	err = deployerContract.AddDeployer(ctx, &AddDeployerRequest{
		DeployerAddr: addr2.MarshalPB(),
		Flags:        uint32(AllowGoDeployFlag),
	})
	require.NoError(err)

	err = deployerContract.AddDeployer(ctx, &AddDeployerRequest{
		DeployerAddr: addr3.MarshalPB(),
		Flags:        uint32(AllowEVMDeployFlag),
	})
	require.NoError(err)

	//permission test
	err = deployerContract.AddDeployer(contractpb.WrapPluginContext(pctx.WithSender(addr3)), &AddDeployerRequest{
		DeployerAddr: addr4.MarshalPB(),
		Flags:        uint32(AllowEVMDeployFlag),
	})
	require.Equal(ErrNotAuthorized, err)

	//permission test
	err = deployerContract.RemoveDeployer(contractpb.WrapPluginContext(pctx.WithSender(addr3)), &RemoveDeployerRequest{
		DeployerAddr: addr3.MarshalPB(),
	})
	require.Equal(ErrNotAuthorized, err)
}

func (dw *DeployerWhitelistTestSuite) TestDeployerWhitelistFlagUtil() {
	require := dw.Require()
	allFlags := []uint32{}
	numFlag := uint32(0)
	for i := uint(0); i < 32; i++ {
		f := uint32(1) << i
		numFlag += f
		allFlags = append(allFlags, f)
	}
	packedFlag := PackFlags(allFlags...)
	require.Equal(numFlag, packedFlag)

	unpackedFlag := UnpackFlags(packedFlag)
	require.Equal(true, reflect.DeepEqual(unpackedFlag, allFlags))

	for i := uint(0); i < 32; i++ {
		f := uint32(1) << i
		require.Equal(true, IsFlagSet(packedFlag, f))
	}
}
