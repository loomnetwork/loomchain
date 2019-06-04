package user_deployer_whitelist

import (
	"testing"
	"time"

	"github.com/loomnetwork/go-loom"
	udwtypes "github.com/loomnetwork/go-loom/builtin/types/user_deployer_whitelist"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/loomnetwork/loomchain/builtin/plugins/deployer_whitelist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	addr1        = loom.MustParseAddress("default:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	addr3        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c5")
	addr4        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c7")
	addr5        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c9")
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
		TierID: udwtypes.TierID_DEFAULT,
		Fee:    fee,
		Name:   "Tier1",
	}
	tierList := []*udwtypes.TierInfo{}
	tierList = append(tierList, tier)
	pctx := createCtx()
	pctx.SetFeature(loomchain.CoinVersion1_1Feature, true)
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
			{Owner: user, Balance: uint64(100)},
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

	require.EqualError(t, ErrOwnerNotSpecified, err.Error(), "Owner Not specified at the time of Initialization")

	err = deployerContract.Init(contractpb.WrapPluginContext(deployerCtx), &InitRequest{
		Owner:    addr4.MarshalPB(),
		TierInfo: nil,
	})

	require.EqualError(t, ErrMissingTierInfo, err.Error(), "Tier Info Not specified at the time of Initialization")

	err = deployerContract.Init(contractpb.WrapPluginContext(deployerCtx), &InitRequest{
		Owner:    addr4.MarshalPB(),
		TierInfo: tierList,
	})

	require.Nil(t, err)

	//Get initial balance of user
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

	// When no deployer attached to user then should return 0 deployers instead of error
	getUserDeployersResponse, err := deployerContract.GetUserDeployers(contractpb.WrapPluginContext(
		deployerCtx), &GetUserDeployersRequest{UserAddr: addr3.MarshalPB()})
	require.NoError(t, err)
	require.Equal(t, 0, len(getUserDeployersResponse.Deployers))

	err = deployerContract.AddUserDeployer(contractpb.WrapPluginContext(deployerCtx.WithSender(addr3)),
		&WhitelistUserDeployerRequest{
			DeployerAddr: addr1.MarshalPB(),
			TierID:       0,
		})

	require.Nil(t, err)

	resp2, err := coinContract.BalanceOf(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)),
		&coin.BalanceOfRequest{
			Owner: addr3.MarshalPB(),
		})

	require.Nil(t, err)
	//Whitelisted fees is debited and final balance of user's loom coin is initial balance - whitelisting fees
	assert.Equal(t, whitelistingfees.Uint64(), resp1.Balance.Value.Uint64()-resp2.Balance.Value.Uint64())

	//Error Cases
	//Trying to Add Duplicate Deployer
	err = deployerContract.AddUserDeployer(contractpb.WrapPluginContext(deployerCtx.WithSender(addr3)),
		&WhitelistUserDeployerRequest{
			DeployerAddr: addr1.MarshalPB(),
			TierID:       0,
		})

	require.EqualError(t, ErrDeployerAlreadyExists, err.Error(), "Trying to Add Deployer which Already Exists for User")

	//Invalid Tier Id specified
	err = deployerContract.AddUserDeployer(contractpb.WrapPluginContext(deployerCtx.WithSender(addr3)),
		&WhitelistUserDeployerRequest{
			DeployerAddr: addr2.MarshalPB(),
			TierID:       1,
		})

	require.EqualError(t, ErrInvalidTier, err.Error(), "Tier Supplied is Invalid")

	getUserDeployersResponse, err = deployerContract.GetUserDeployers(contractpb.WrapPluginContext(
		deployerCtx), &GetUserDeployersRequest{UserAddr: addr3.MarshalPB()})
	require.NoError(t, err)
	require.Equal(t, 1, len(getUserDeployersResponse.Deployers))

	// When no contracts attached should return 0 contracts instead of error
	getDeployedContractsResponse, err := deployerContract.GetDeployedContracts(contractpb.WrapPluginContext(
		deployerCtx.WithSender(addr3)), &GetDeployedContractsRequest{
		DeployerAddr: addr1.MarshalPB(),
	})
	require.NoError(t, err)
	require.Equal(t, 0, len(getDeployedContractsResponse.ContractAddresses))

	err = RecordEVMContractDeployment(contractpb.WrapPluginContext(deployerCtx.WithSender(addr3)),
		addr1, contractAddr)
	require.Nil(t, err)
	// addr1 is deployer
	getDeployedContractsResponse, err = deployerContract.GetDeployedContracts(contractpb.WrapPluginContext(
		deployerCtx.WithSender(addr3)), &GetDeployedContractsRequest{
		DeployerAddr: addr1.MarshalPB(),
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(getDeployedContractsResponse.ContractAddresses))
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
