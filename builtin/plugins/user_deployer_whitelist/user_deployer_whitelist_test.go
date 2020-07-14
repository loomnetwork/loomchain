package user_deployer_whitelist

import (
	"testing"
	"time"

	"github.com/loomnetwork/go-loom"
	udwtypes "github.com/loomnetwork/go-loom/builtin/types/user_deployer_whitelist"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/loomnetwork/loomchain/builtin/plugins/deployer_whitelist"
	"github.com/loomnetwork/loomchain/features"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	addr1        = loom.MustParseAddress("default:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	addr3        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c5")
	addr4        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c7")
	addr5        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c9")
	addr6        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e0112")
	contractAddr = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01ab")
	user         = addr3.MarshalPB()
	chainID      = "default"
)

func TestUserDeployerWhitelistContract(t *testing.T) {
	div := loom.NewBigUIntFromInt(10)
	div.Exp(div, loom.NewBigUIntFromInt(18), nil)
	fee := uint64(100)
	whitelistingfees := loom.NewBigUIntFromInt(int64(fee))
	whitelistingfees = whitelistingfees.Mul(whitelistingfees, div)
	tier := &udwtypes.TierInfo{
		TierID:     udwtypes.TierID_DEFAULT,
		Fee:        fee,
		Name:       "Tier1",
		BlockRange: 10,
		MaxTxs:     20,
	}
	tierList := []*udwtypes.TierInfo{}
	tierList = append(tierList, tier)
	pctx := createCtx()
	pctx.SetFeature(features.CoinVersion1_1Feature, true)
	deployContract := &deployer_whitelist.DeployerWhitelist{}
	deployerAddr := pctx.CreateContract(deployer_whitelist.Contract)
	dctx := pctx.WithAddress(deployerAddr)
	err := deployContract.Init(contractpb.WrapPluginContext(dctx), &deployer_whitelist.InitRequest{
		Owner: addr4.MarshalPB(),
		Deployers: []*Deployer{
			&Deployer{
				Address: addr5.MarshalPB(),
				Flags:   uint32(1),
			},
		},
	})

	require.Nil(t, err)

	coinContract := &coin.Coin{}
	coinAddr := pctx.CreateContract(coin.Contract)
	coinCtx := pctx.WithAddress(coinAddr)
	err = coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			{Owner: user, Balance: uint64(300)},
		},
	})

	require.Nil(t, err)

	deployerContract := &UserDeployerWhitelist{}
	userWhitelistDeployerAddr := pctx.CreateContract(Contract)
	deployerCtx := pctx.WithAddress(userWhitelistDeployerAddr)

	err = deployerContract.Init(contractpb.WrapPluginContext(deployerCtx), &InitRequest{
		Owner:    nil,
		TierInfo: tierList,
	})

	require.EqualError(t, ErrOwnerNotSpecified, err.Error(), "Owner is not specified at the time of initialization")

	err = deployerContract.Init(contractpb.WrapPluginContext(deployerCtx), &InitRequest{
		Owner:    addr4.MarshalPB(),
		TierInfo: nil,
	})

	require.EqualError(t, ErrMissingTierInfo, err.Error(), "Tier info is not specified at the time of initialization")

	err = deployerContract.Init(contractpb.WrapPluginContext(deployerCtx), &InitRequest{
		Owner:    addr4.MarshalPB(),
		TierInfo: tierList,
	})

	require.Nil(t, err)

	//Get the initial balance for the user
	resp1, err := coinContract.BalanceOf(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)),
		&coin.BalanceOfRequest{
			Owner: addr3.MarshalPB(),
		})

	approvalAmount := sciNot(1000, 18)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr3)), &coin.ApproveRequest{
		Spender: userWhitelistDeployerAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *approvalAmount},
	})

	require.Nil(t, err)

	// When no deployer is attached to the user, then it should return 0 deployers instead of an error
	getUserDeployersResponse, err := deployerContract.GetUserDeployers(contractpb.WrapPluginContext(
		deployerCtx), &GetUserDeployersRequest{UserAddr: addr3.MarshalPB()})
	require.NoError(t, err)
	require.Equal(t, 0, len(getUserDeployersResponse.Deployers))
	// add a deployer addr1 by addr3
	err = deployerContract.AddUserDeployer(contractpb.WrapPluginContext(deployerCtx.WithSender(addr3)),
		&WhitelistUserDeployerRequest{
			DeployerAddr: addr1.MarshalPB(),
			TierID:       0,
		})

	require.Nil(t, err)

	// checking the balance after addr1 gets deployed
	resp2, err := coinContract.BalanceOf(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)),
		&coin.BalanceOfRequest{
			Owner: addr3.MarshalPB(),
		})

	require.Nil(t, err)
	//Whitelisting fees are debited, and the final LOOM balance of the user is initial balance - whitelisting fees
	assert.Equal(t, whitelistingfees.Uint64(), resp1.Balance.Value.Uint64()-resp2.Balance.Value.Uint64())

	//Error Cases
	//Trying to add a duplicate deployer
	err = deployerContract.AddUserDeployer(contractpb.WrapPluginContext(deployerCtx.WithSender(addr3)),
		&WhitelistUserDeployerRequest{
			DeployerAddr: addr1.MarshalPB(),
			TierID:       0,
		})

	require.EqualError(t, ErrDeployerAlreadyExists, err.Error(), "Trying to add a deployer which already exists for user")

	getUserDeployersResponse, err = deployerContract.GetUserDeployers(contractpb.WrapPluginContext(
		deployerCtx), &GetUserDeployersRequest{UserAddr: addr3.MarshalPB()})
	require.NoError(t, err)
	require.Equal(t, 1, len(getUserDeployersResponse.Deployers))

	// When no contracts are attached, it should return 0 contracts instead of an error
	getDeployedContractsResponse, err := deployerContract.GetDeployedContracts(contractpb.WrapPluginContext(
		deployerCtx.WithSender(addr3)), &GetDeployedContractsRequest{
		DeployerAddr: addr1.MarshalPB(),
	})
	require.NoError(t, err)
	require.Equal(t, 0, len(getDeployedContractsResponse.ContractAddresses))

	err = RecordEVMContractDeployment(contractpb.WrapPluginContext(deployerCtx.WithSender(addr3)),
		addr1, contractAddr)
	require.Nil(t, err)
	// When one contract is deployed, length = 1
	getDeployedContractsResponse, err = deployerContract.GetDeployedContracts(contractpb.WrapPluginContext(
		deployerCtx.WithSender(addr3)), &GetDeployedContractsRequest{
		DeployerAddr: addr1.MarshalPB(),
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(getDeployedContractsResponse.ContractAddresses))

	//Modify tier info
	err = deployerContract.SetTierInfo(contractpb.WrapPluginContext(deployerCtx.WithSender(addr4)),
		&SetTierInfoRequest{
			Name:   "Tier2",
			TierID: udwtypes.TierID_DEFAULT,
			Fee: &types.BigUInt{
				Value: *whitelistingfees,
			},
			BlockRange: 10,
			MaxTxs:     20,
		})

	require.NoError(t, err)

	//Error test case: modify tier info for an unauthorized user
	err = deployerContract.SetTierInfo(contractpb.WrapPluginContext(deployerCtx.WithSender(addr5)),
		&SetTierInfoRequest{
			Name:   "Tier2",
			TierID: udwtypes.TierID_DEFAULT,
			Fee: &types.BigUInt{
				Value: *whitelistingfees,
			},
			BlockRange: 10,
			MaxTxs:     20,
		})

	require.EqualError(t, ErrNotAuthorized, err.Error(), "Can be modified only by the owner")

	//Get Tier Info
	resp, err := deployerContract.GetTierInfo(contractpb.WrapPluginContext(deployerCtx),
		&udwtypes.GetTierInfoRequest{
			TierID: udwtypes.TierID_DEFAULT,
		})
	fees := sciNot(100, 18)
	require.NoError(t, err)
	require.Equal(t, fees, &resp.Tier.Fee.Value)
	require.Equal(t, "Tier2", resp.Tier.Name)
	// when the deployer is not in ctx it should not return an error, but an empty list of contracts
	getDeployedContractsResponse, err = deployerContract.GetDeployedContracts(contractpb.WrapPluginContext(
		deployerCtx.WithSender(addr3)), &GetDeployedContractsRequest{
		DeployerAddr: addr3.MarshalPB(),
	})
	require.NoError(t, err)
	require.Equal(t, 0, len(getDeployedContractsResponse.ContractAddresses))

	// add a deployer addr6 by addr3
	err = deployerContract.AddUserDeployer(contractpb.WrapPluginContext(deployerCtx.WithSender(addr3)),
		&WhitelistUserDeployerRequest{
			DeployerAddr: addr6.MarshalPB(),
			TierID:       0,
		})

	require.Nil(t, err)

	// removing the deployer addr6 by add1 should error with :ErrDeployerDoesNotExist
	err = deployerContract.RemoveUserDeployer(contractpb.WrapPluginContext(deployerCtx.WithSender(addr1)),
		&udwtypes.RemoveUserDeployerRequest{
			DeployerAddr: addr6.MarshalPB(),
		})

	require.EqualError(t, ErrDeployerDoesNotExist, err.Error(), "Deployer Does not exist corresponding to the user")
	// as the above deployer is not removed, GetUserDeployers should return 2
	getUserDeployersResponse, err = deployerContract.GetUserDeployers(contractpb.WrapPluginContext(
		deployerCtx), &GetUserDeployersRequest{UserAddr: addr3.MarshalPB()})
	require.NoError(t, err)
	require.Equal(t, 2, len(getUserDeployersResponse.Deployers))
	// removing the deployer addr6 by addr3 should not error
	err = deployerContract.RemoveUserDeployer(contractpb.WrapPluginContext(deployerCtx.WithSender(addr3)),
		&udwtypes.RemoveUserDeployerRequest{
			DeployerAddr: addr6.MarshalPB(),
		})

	require.Nil(t, err)
	// as the above deployer is removed, GetUserDeployers should return 1
	getUserDeployersResponse, err = deployerContract.GetUserDeployers(contractpb.WrapPluginContext(
		deployerCtx), &GetUserDeployersRequest{UserAddr: addr3.MarshalPB()})
	require.NoError(t, err)
	require.Equal(t, 1, len(getUserDeployersResponse.Deployers))

}

func sciNot(m, n int64) *loom.BigUInt {
	ret := loom.NewBigUIntFromInt(10)
	ret.Exp(ret, loom.NewBigUIntFromInt(n), nil)
	ret.Mul(ret, loom.NewBigUIntFromInt(m))
	return ret
}

func createCtx() *plugin.FakeContext {
	return plugin.CreateFakeContext(loom.Address{}, loom.Address{}).WithBlock(loom.BlockHeader{
		ChainID: "default",
		Time:    time.Now().Unix(),
	})
}
