package karma

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/pkg/errors"
	"github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/go-loom/common"
)

const (
	CoinDeployToken         = "coin-deploy"
	CoinDefaultReward       = 1
	UserStateKeyPrefix      = "user_state"
	oracleRole              = "karma_role_oracle"
	DefaultUpkeepCost       = 1
	DefaultUpkeepPeriod     = 3600
)

var (
	OracleKey      = []byte("karma:oracle:key")
	SourcesKey     = []byte("karma:sources:key")
	UpkeepKey      = []byte("karma:upkeep:params:kep")
	ActivePrefix   = []byte("active")
	InactivePrefix = []byte("inactive")
	ConfigKey      = []byte("config:key")

	ChangeOraclePermission      = []byte("change_oracle")
	ChangeUserSourcesPermission = []byte("change_user_sources")
	SetUpkeepPermission         = []byte("set-upkeep")
	ResetSourcesPermission      = []byte("reset_sources")
	ChangeConfigPermission      = []byte("change-config")

	defaultUpkeep = &ktypes.KarmaUpkeepParams{
		Cost:   DefaultUpkeepCost,
		Period: DefaultUpkeepPeriod,
	}
	ErrNotAuthorized = errors.New("sender is not authorized to call this method")
)

func ContractActiveRecordKey(contractAddr loom.Address) []byte {
	return util.PrefixKey(ActivePrefix, contractAddr.Bytes())
}

func ContractInactiveRecordKey(contractAddr loom.Address) []byte {
	return util.PrefixKey(InactivePrefix, contractAddr.Bytes())
}

func UserStateKey(owner *types.Address) []byte {
	return util.PrefixKey([]byte(UserStateKeyPrefix), []byte(owner.String()))
}

type Karma struct {
}

func (k *Karma) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "karma",
		Version: "1.0.0",
	}, nil
}

func (k *Karma) Init(ctx contract.Context, req *ktypes.KarmaInitRequest) error {
	foundCoinSource := false
	for _, source := range req.Sources {
		if source.Name == CoinDeployToken {
			foundCoinSource = true
			break
		}
	}
	if !foundCoinSource {
		req.Sources = append(req.Sources, &ktypes.KarmaSourceReward{
			Name: CoinDeployToken,
			Reward: CoinDefaultReward,
		})
	}

	if err := ctx.Set(SourcesKey, &ktypes.KarmaSources{Sources: req.Sources}); err != nil {
		return errors.Wrap(err, "Error setting sources")
	}

	if req.Oracle != nil {
		if err := k.registerOracle(ctx, req.Oracle, nil); nil != err {
			return errors.Wrap(err, "Error setting oracle")
		}
	}

	for _, user := range req.Users {
		ksu := &ktypes.KarmaStateUser{
			User:         user.User,
			SourceStates: make([]*ktypes.KarmaSource, 0),
		}
		for _, source := range user.Sources {
			ksu.SourceStates = append(ksu.SourceStates, source)
		}
		if err := k.validatedUpdateSourcesForUser(ctx, ksu); err != nil {
			return errors.Wrapf(err, "updating source for user %v ", ksu.User)
		}
	}

	if req.Config == nil {
		if err := ctx.Set(ConfigKey, &ktypes.KarmaConfig{ MinKarmaToDeploy: DefaultUpkeepCost }); err != nil {
			return errors.Wrap(err, "setting config params")
		}
	} else {
		if err := ctx.Set(ConfigKey, req.Config); err != nil {
			return errors.Wrap(err, "setting config params")
		}
	}

	if req.Upkeep == nil {
		if err := ctx.Set(UpkeepKey, defaultUpkeep); err != nil {
			return errors.Wrap(err, "setting upkeep params")
		}
	} else {
		if err := ctx.Set(UpkeepKey, req.Upkeep); err != nil {
			return errors.Wrap(err, "setting upkeep params")
		}
	}

	return nil
}

func (k *Karma) DepositCoin(ctx contract.Context, req *ktypes.KarmaUserAmount) error {
	coinAddr, err := ctx.Resolve("coin")
	if err != nil {
		return errors.Wrap(err, "address of coin contract")
	}

	coinReq := &coin.TransferFromRequest{
		To: 	ctx.ContractAddress().MarshalPB(),
		From: 	req.User,
		Amount: req.Amount,
	}
	if err := contract.CallMethod(ctx, coinAddr, "TransferFrom", coinReq, nil); err != nil {
		return errors.Wrap(err, "transferring coin to karma contract")
	}

	if err := modifyCountForUser(ctx, req.User, CoinDeployToken, req.Amount); err != nil {
		return errors.Wrapf(err, "modifying user %v's upkeep count", req.User.String())
	}
	return nil
}

func (k *Karma) WithdrawCoin(ctx contract.Context, req *ktypes.KarmaUserAmount) error {
	coinAddr, err := ctx.Resolve("coin")
	if err != nil {
		return errors.Wrap(err, "address of coin contract")
	}

	coinReq := &coin.TransferRequest{
		To:  req.User,
		Amount: req.Amount,
	}
	if err := contract.CallMethod(ctx, coinAddr, "Transfer", coinReq, nil); err != nil {
		return errors.Wrap(err,"transferring coin from karma contract")
	}

	amount := req.Amount.Value.Mul(&req.Amount.Value, loom.NewBigUIntFromInt(-1))
	if err := modifyCountForUser(ctx, req.User, CoinDeployToken, &types.BigUInt{ Value: *amount }); err != nil {
		return errors.Wrapf(err, "modifying user %v's  upkeep count", req.User.String())
	}
	return nil
}

func (k *Karma) SetConfig(ctx contract.Context, req *ktypes.KarmaConfig) error {
	if hasPermission, _ := ctx.HasPermission(ChangeConfigPermission, []string{oracleRole}); !hasPermission {
		return ErrNotAuthorized
	}

	if err := ctx.Set(ConfigKey, req); err != nil {
		return errors.Wrap(err, "Error setting config")
	}
	return nil
}

func (k *Karma) GetConfig(ctx contract.StaticContext, _ *ktypes.GetConfigRequest) (*ktypes.KarmaConfig, error) {
	var config ktypes.KarmaConfig
	if err := ctx.Get(ConfigKey, &config); err != nil {
		if err == contract.ErrNotFound {
			return &ktypes.KarmaConfig{}, nil
		}
		return nil, err
	}
	return &config, nil
}

func (k *Karma) SetUpkeepParams(ctx contract.Context, params *ktypes.KarmaUpkeepParams) error {
	if hasPermission, _ := ctx.HasPermission(SetUpkeepPermission, []string{oracleRole}); !hasPermission {
		return ErrNotAuthorized
	}
	var oldParams ktypes.KarmaUpkeepParams
	if err := ctx.Get(UpkeepKey, &oldParams); err != nil {
		return errors.Wrap(err, "get upkeep params from db")
	}
	if params.Cost == 0 {
		params.Cost = oldParams.Cost
	}
	if params.Period == 0 {
		params.Period = oldParams.Period
	}

	if err := ctx.Set(UpkeepKey, params); err != nil {
		return errors.Wrap(err, "setting upkeep params")
	}
	return nil
}

func (k *Karma) GetUpkeepParms(ctx contract.StaticContext, ko *types.Address) (*ktypes.KarmaUpkeepParams, error) {
	var upkeep ktypes.KarmaUpkeepParams
	if err := ctx.Get(UpkeepKey, &upkeep); err != nil {
		return nil, errors.Wrap(err, "get upkeep params from db")
	}
	return &upkeep, nil
}

func (k *Karma) SetActive(ctx contract.Context, contract *types.Address) error {
	addr := loom.UnmarshalAddressPB(contract)
	var record ktypes.ContractRecord
	if err := ctx.Get(ContractInactiveRecordKey(addr), &record); err != nil {
		return errors.Wrapf(err, "getting record for %s", addr.String())
	}
	ctx.Delete(ContractInactiveRecordKey(addr))
	if err := ctx.Set(ContractActiveRecordKey(addr), &record); err != nil {
		return errors.Wrapf(err, "setting record %v for %s", record, addr.String())
	}
	return nil
}

func (k *Karma) SetInactive(ctx contract.Context, contract *types.Address) error {
	addr := loom.UnmarshalAddressPB(contract)
	var record ktypes.ContractRecord
	if err := ctx.Get(ContractActiveRecordKey(addr), &record); err != nil {
		return errors.Wrapf(err, "getting record for %s", addr.String())
	}
	ctx.Delete(ContractActiveRecordKey(addr))
	if err := ctx.Set(ContractInactiveRecordKey(addr), &record); err != nil {
		return errors.Wrapf(err, "setting record %v for %s", record, addr.String())
	}
	return nil
}

func (k *Karma) IsActive(ctx contract.StaticContext, contract *types.Address) (bool, error) {
	return ctx.Has(ContractActiveRecordKey(loom.UnmarshalAddressPB(contract))), nil
}

func AddOwnedContract(state loomchain.State, owner loom.Address, contract loom.Address, block int64, nonce uint64) error {
	record, err := proto.Marshal(&ktypes.ContractRecord{
		Owner:         	owner.MarshalPB(),
		Address:       	contract.MarshalPB(),
		CreationBlock: 	block,
		Nonce: 			int64(nonce),
	})
	if err != nil {
		return errors.Wrapf(err, "marshal record %v", record)
	}
	state.Set(ContractActiveRecordKey(contract), record)
	return nil
}

func SetInactive(state loomchain.State, contract loom.Address) error {
	record := state.Get(ContractActiveRecordKey(contract))
	if len(record) == 0 {
		return errors.Errorf("contract not found %v", contract.String())
	}
	state.Delete(ContractActiveRecordKey(contract))
	state.Set(ContractInactiveRecordKey(contract), record)
	return nil
}

func GetActiveContractRecords(state loomchain.State) ([]*ktypes.ContractRecord, error) {
	var records []*ktypes.ContractRecord
	activeRecords := state.Range(ActivePrefix)
	for _, kv := range activeRecords {
		var record ktypes.ContractRecord
		if err := proto.Unmarshal(kv.Value, &record); err != nil {
			return nil, errors.Wrapf(err, "unmarshal record %v", kv.Value)
		}
		records = append(records, &record)
	}
	return records, nil
}

func (k *Karma) GetSources(ctx contract.StaticContext, _ *ktypes.GetSourceRequest) (*ktypes.KarmaSources, error) {
	var sources ktypes.KarmaSources
	if err := ctx.Get(SourcesKey, &sources); err != nil {
		if err == contract.ErrNotFound {
			return &ktypes.KarmaSources{}, nil
		}
		return nil, err
	}
	return &sources, nil
}

func (k *Karma) GetUserState(ctx contract.StaticContext, user *types.Address) (*ktypes.KarmaState, error) {
	stateKey := UserStateKey(user)
	var curState ktypes.KarmaState
	if err := ctx.Get(stateKey, &curState); err != nil {
		if err == contract.ErrNotFound {
			return &ktypes.KarmaState{}, nil
		}
		return nil, err
	}
	return &curState, nil
}

func (k *Karma) DeleteSourcesForUser(ctx contract.Context, ksu *ktypes.KarmaStateKeyUser) error {
	if hasPermission, _ := ctx.HasPermission(ChangeUserSourcesPermission, []string{oracleRole}); !hasPermission {
		return ErrNotAuthorized
	}

	state, err := k.GetUserState(ctx, ksu.User)
	if err != nil {
		return err
	}

	for k := range ksu.StateKeys {
		for index, s := range state.SourceStates {
			if s.Name == ksu.StateKeys[k] {
				state.SourceStates = append(state.SourceStates[:index], state.SourceStates[index+1:]...)
			}
		}
	}

	var karmaSources ktypes.KarmaSources
	if err := ctx.Get(SourcesKey, &karmaSources); err != nil {
		return err
	}
	state.DeployKarmaTotal, state.CallKarmaTotal = CalculateTotalKarma(karmaSources, *state)
	state.LastUpdateTime = ctx.Now().Unix()
	return ctx.Set(UserStateKey(ksu.User), state)
}

func (k *Karma) ResetSources(ctx contract.Context, kpo *ktypes.KarmaSources) error {
	if hasPermission, _ := ctx.HasPermission(ResetSourcesPermission, []string{oracleRole}); !hasPermission {
		return ErrNotAuthorized
	}

	if err := ctx.Set(SourcesKey, &ktypes.KarmaSources{Sources: kpo.Sources}); err != nil {
		return errors.Wrap(err, "Error setting sources")
	}
	if err := k.updateKarmaCounts(ctx, ktypes.KarmaSources{Sources: kpo.Sources}); err != nil {
		return errors.Wrap(err, "updating karma counts")
	}
	return nil
}

func (k *Karma) UpdateOracle(ctx contract.Context, params *ktypes.KarmaNewOracle) error {
	if hasPermission, _ := ctx.HasPermission(ChangeOraclePermission, []string{oracleRole}); !hasPermission {
		return ErrNotAuthorized
	}

	currentOracle := ctx.Message().Sender
	return k.registerOracle(ctx, params.NewOracle, &currentOracle)
}

func (k *Karma) AppendSourcesForUser(ctx contract.Context, ksu *ktypes.KarmaStateUser) error {
	if hasPermission, _ := ctx.HasPermission(ChangeUserSourcesPermission, []string{oracleRole}); !hasPermission {
		return ErrNotAuthorized
	}
	return k.validatedUpdateSourcesForUser(ctx, ksu)
}

func (k *Karma) GetUserKarma(ctx contract.StaticContext, userTarget *ktypes.KarmaUserTarget) (*ktypes.KarmaTotal, error) {
	userState, err := k.GetUserState(ctx, userTarget.User)
	if err != nil {
		return nil, err
	}
	if userState.DeployKarmaTotal == nil {
		userState.DeployKarmaTotal = &types.BigUInt{Value: *common.BigZero()}
	}
	if userState.CallKarmaTotal == nil {
		userState.CallKarmaTotal = &types.BigUInt{Value: *common.BigZero()}
	}
	switch userTarget.Target {
	case ktypes.KarmaSourceTarget_DEPLOY:
		return &ktypes.KarmaTotal{Count: userState.DeployKarmaTotal}, nil
	case ktypes.KarmaSourceTarget_CALL:
		return &ktypes.KarmaTotal{Count: userState.CallKarmaTotal}, nil
	default:
		return nil, fmt.Errorf("unknown karma type %v", userTarget.Target)
	}
}

func (c *Karma) registerOracle(ctx contract.Context, pbOracle *types.Address, currentOracle *loom.Address) error {
	if pbOracle == nil {
		return fmt.Errorf("oracle address cannot be null")
	}

	newOracleAddr := loom.UnmarshalAddressPB(pbOracle)
	if newOracleAddr.IsEmpty() {
		return fmt.Errorf("oracle address cannot be empty")
	}

	if currentOracle != nil {
		ctx.RevokePermissionFrom(*currentOracle, ChangeOraclePermission, oracleRole)
		ctx.RevokePermissionFrom(*currentOracle, ChangeUserSourcesPermission, oracleRole)
		ctx.RevokePermissionFrom(*currentOracle, ResetSourcesPermission, oracleRole)
		ctx.RevokePermissionFrom(*currentOracle, ChangeConfigPermission, oracleRole)
		ctx.RevokePermissionFrom(*currentOracle, SetUpkeepPermission, oracleRole)
	}

	ctx.GrantPermissionTo(newOracleAddr, ChangeOraclePermission, oracleRole)
	ctx.GrantPermissionTo(newOracleAddr, ChangeUserSourcesPermission, oracleRole)
	ctx.GrantPermissionTo(newOracleAddr, ResetSourcesPermission, oracleRole)
	ctx.GrantPermissionTo(newOracleAddr, SetUpkeepPermission, oracleRole)
	ctx.GrantPermissionTo(newOracleAddr, ChangeConfigPermission, oracleRole)
	if err := ctx.Set(OracleKey, pbOracle); err != nil {
		return errors.Wrap(err, "setting new oracle")
	}
	return nil
}

func CalculateTotalKarma(karmaSources ktypes.KarmaSources, karmaStates ktypes.KarmaState) (*types.BigUInt, *types.BigUInt) {
	deployKarma := types.BigUInt{Value: *common.BigZero()}
	callKarma := types.BigUInt{Value: *common.BigZero()}
	for _, c := range karmaSources.Sources {
		for _, s := range karmaStates.SourceStates {
			if c.Name == s.Name && (c.Target == ktypes.KarmaSourceTarget_DEPLOY) {
				reward := loom.NewBigUIntFromInt(c.Reward)
				deployKarma.Value.Add(&deployKarma.Value, reward.Mul(reward, &s.Count.Value))
			}
			if c.Name == s.Name && (c.Target == ktypes.KarmaSourceTarget_CALL) {
				reward := loom.NewBigUIntFromInt(c.Reward)
				callKarma.Value.Add(&callKarma.Value, reward.Mul(reward, &s.Count.Value))
			}
		}
	}
	return &deployKarma, &callKarma
}

func (k *Karma) updateKarmaCounts(ctx contract.Context, sources ktypes.KarmaSources) error {
	userRange := ctx.Range([]byte(UserStateKeyPrefix))
	for _, userKV := range userRange {
		var karmaStates ktypes.KarmaState
		if err := proto.Unmarshal(userKV.Value, &karmaStates); err != nil {
			return errors.Wrap(err, "unmarshal karma user state")
		}
		karmaStates.DeployKarmaTotal, karmaStates.CallKarmaTotal = CalculateTotalKarma(sources, karmaStates)
		userStateKey := util.PrefixKey([]byte(UserStateKeyPrefix), userKV.Key)
		if err := ctx.Set(userStateKey, &karmaStates); err != nil {
			return errors.Wrap(err, "setting user state karma")
		}
	}
	return nil
}

func modifyCountForUser(ctx contract.Context, user *types.Address, sourceName string, amount *types.BigUInt) error {
	stateKey := UserStateKey(user)

	var userSourceCounts ktypes.KarmaState
	// If user source counts not found, We want to create a new source count
	if err := ctx.Get(stateKey, &userSourceCounts); err != nil && err != contract.ErrNotFound {
		return errors.Wrapf(err, "source counts for user %s", user.String())
	}

	found := false
	for i, source := range userSourceCounts.SourceStates {
		if source.Name == sourceName {
			newAmount := common.BigZero()
			newAmount.Add(&userSourceCounts.SourceStates[i].Count.Value, &amount.Value)
			if newAmount.Cmp(common.BigZero()) < 0 {
				return errors.Errorf("not enough karma in source %s. found %v, modifying by %v", sourceName, userSourceCounts.SourceStates[i].Count, amount)
			}
			userSourceCounts.SourceStates[i].Count = &types.BigUInt{ Value: *newAmount }
			found = true
			break
		}
	}

	if !found {
		// if source for the user does not exist create and set to amount if positive
		if amount.Value.Cmp(common.BigZero()) < 0 {
			return  errors.Errorf("not enough karma in source %s. found 0, modifying by %v", user, amount)
		}
		userSourceCounts.SourceStates = append(userSourceCounts.SourceStates, &ktypes.KarmaSource{
			Name:  sourceName,
			Count: amount,
		})
	}

	var karmaSources ktypes.KarmaSources
	if err := ctx.Get(SourcesKey, &karmaSources); err != nil {
		return err
	}
	userSourceCounts.DeployKarmaTotal, userSourceCounts.CallKarmaTotal = CalculateTotalKarma(karmaSources, userSourceCounts)

	if err := ctx.Set(UserStateKey(user), &userSourceCounts); err != nil {
		return  errors.Wrapf(err, "setting user source counts for %s", user.String())
	}
	return  nil
}

func (k *Karma) validatedUpdateSourcesForUser(ctx contract.Context, ksu *ktypes.KarmaStateUser) error {
	var state *ktypes.KarmaState
	var err error

	state, err = k.GetUserState(ctx, ksu.User)
	if err != nil {
		return err
	}

	for _, v := range ksu.SourceStates {
		var flag = false
		for index := range state.SourceStates {
			if state.SourceStates[index].Name == v.Name {
				state.SourceStates[index].Count = v.Count
				flag = true
			}
		}
		if !flag {
			state.SourceStates = append(state.SourceStates, v)
		}

	}
	state.LastUpdateTime = ctx.Now().Unix()


	var karmaSources ktypes.KarmaSources
	if err := ctx.Get(SourcesKey, &karmaSources); err != nil {
		return err
	}
	state.DeployKarmaTotal, state.CallKarmaTotal = CalculateTotalKarma(karmaSources, *state)

	return ctx.Set(UserStateKey(ksu.User), state)
}

var Contract plugin.Contract = contract.MakePluginContract(&Karma{})
