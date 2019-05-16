package user_deployer_whitelist

import (
	"math/big"
	"testing"
	"time"

	"github.com/loomnetwork/go-loom"
	udwtypes "github.com/loomnetwork/go-loom/builtin/types/user_deployer_whitelist"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	vm "github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/loomnetwork/loomchain/builtin/plugins/deployer_whitelist"
	"github.com/stretchr/testify/require"
)

var (
	addr1        = loom.MustParseAddress("default:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	addr3        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c5")
	addr4        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c7")
	addr5        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c9")
	contractAddr = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01ab")
	user         = addr1.MarshalPB()
	user_addr    = addr3
	chainID      = "default"
)

func TestUserDeployerWhitelistContract(t *testing.T) {

	tier := &udwtypes.Tier{
		Id:   0,
		Fee:  &types.BigUInt{Value: *loom.NewBigUInt(big.NewInt(100))},
		Name: "DEFAULT",
	}
	tierList := []*udwtypes.Tier{}
	tierList = append(tierList, tier)
	tierInfo := &udwtypes.TierInfo{
		Tiers: tierList,
	}

	pctx := createCtx()

	deployContract := &deployer_whitelist.DeployerWhitelist{}
	dAddr := pctx.CreateContract(deployer_whitelist.Contract)
	dctx := pctx.WithAddress(dAddr)
	err := deployContract.Init(contractpb.WrapPluginContext(dctx), &deployer_whitelist.InitRequest{
		Owner: addr4.MarshalPB(),
		Deployers: []*Deployer{
			&Deployer{
				Address: addr5.MarshalPB(),
				Flags:   uint32(1),
			},
		},
	})

	coinContract := &coin.Coin{}
	coinAddr := pctx.CreateContract(coin.Contract)
	coinCtx := pctx.WithAddress(coinAddr)
	coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			{Owner: user, Balance: uint64(100)},
		},
	})

	require.Nil(t, err)

	deployerContract := &UserDeployerWhitelist{}
	deployerAddr := pctx.CreateContract(Contract)
	deployerCtx := pctx.WithAddress(deployerAddr)

	deployerContract.Init(contractpb.WrapPluginContext(deployerCtx), &InitRequest{
		Owner:    addr1.MarshalPB(),
		TierInfo: tierInfo,
	})

	coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)), &coin.ApproveRequest{
		Spender: deployerAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(10000)},
	})
	require.Nil(t, err)

	err = deployerContract.AddUserDeployer(contractpb.WrapPluginContext(deployerCtx.WithSender(addr3)),
		&WhitelistUserDeployerRequest{
			DeployerAddr: addr1.MarshalPB(),
			TierId:       0,
		})
	require.NoError(t, err)

	resp1, err := coinContract.BalanceOf(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)), &coin.BalanceOfRequest{
		Owner: deployerAddr.MarshalPB(),
	})
	require.Equal(t, int64(0), resp1.Balance.Value.Int64())
	require.Nil(t, err)

	err = deployerContract.AddUserDeployer(contractpb.WrapPluginContext(deployerCtx.WithSender(addr3)),
		&WhitelistUserDeployerRequest{
			DeployerAddr: addr1.MarshalPB(),
			TierId:       0,
		})
	require.Equal(t, ErrDeployerAlreadyExists, err, "Trying to Add Deployer which Already Exists for User")

	err = deployerContract.AddUserDeployer(contractpb.WrapPluginContext(deployerCtx.WithSender(addr3)),
		&WhitelistUserDeployerRequest{
			DeployerAddr: addr2.MarshalPB(),
			TierId:       1,
		})
	require.Equal(t, ErrInvalidTier, err, "Tier Supplied is Invalid")

	getUserDeployersResponse, err := deployerContract.GetUserDeployers(contractpb.WrapPluginContext(
		deployerCtx.WithSender(addr3)), &GetUserDeployersRequest{})
	require.NoError(t, err)
	require.Equal(t, 1, len(getUserDeployersResponse.Deployers))

	// addr3 is not a deployer so response should be Nil
	getDeployedContractsResponse, err := deployerContract.GetDeployedContracts(contractpb.WrapPluginContext(
		deployerCtx.WithSender(addr3)), &GetDeployedContractsRequest{
		DeployerAddr: addr3.MarshalPB(),
	})
	require.Error(t, err)
	require.Nil(t, getDeployedContractsResponse)

	err = RecordContractDeployment(contractpb.WrapPluginContext(deployerCtx.WithSender(addr3)),
		addr1, contractAddr, vm.VMType_EVM)
	require.Nil(t, err)

	// addr1 is deployer
	getDeployedContractsResponse, err = deployerContract.GetDeployedContracts(contractpb.WrapPluginContext(
		deployerCtx.WithSender(addr3)), &GetDeployedContractsRequest{
		DeployerAddr: addr1.MarshalPB(),
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(getDeployedContractsResponse.ContractAddresses))
}

func createCtx() *plugin.FakeContext {
	return plugin.CreateFakeContext(loom.Address{}, loom.Address{}).WithBlock(loom.BlockHeader{
		ChainID: "default",
		Time:    time.Now().Unix(),
	})
}
