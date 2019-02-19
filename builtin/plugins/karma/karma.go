package karma

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/builtin/types/coin"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/pkg/errors"
)

const (
	CoinDeployToken    = "coin-deploy"
	CoinDefaultReward  = 1
	UserStateKeyPrefix = "user_state"
	oracleRole         = "karma_role_oracle"
)

var (
	OracleKey  = []byte("karma:oracle:key")
	SourcesKey = []byte("karma:sources:key")
	ConfigKey  = []byte("config:key")

	ChangeOraclePermission      = []byte("change_oracle")
	ChangeUserSourcesPermission = []byte("change_user_sources")
	SetUpkeepPermission         = []byte("set-upkeep")
	ResetSourcesPermission      = []byte("reset_sources")
	ChangeConfigPermission      = []byte("change-config")

	ErrNotAuthorized = errors.New("sender is not authorized to call this method")
)

// TODO: should take loom.Address instead
func UserStateKey(owner *types.Address) ([]byte, error) {
	key, err := proto.Marshal(owner)
	if err != nil {
		return nil, err
	}
	return util.PrefixKey([]byte(UserStateKeyPrefix), key), nil
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
			Name:   CoinDeployToken,
			Reward: CoinDefaultReward,
			Target: ktypes.KarmaSourceTarget_DEPLOY,
		})
	}

	if err := ctx.Set(NextContractIdKey, &ktypes.KarmaContractId{ContractId: 1}); err != nil {
		return errors.Wrap(err, "Error setting next contract id")
	}

	if err := SetAllowedKarmaSources(ctx, req.Sources); err != nil {
		return errors.Wrap(err, "failed to set allowed karma sources")
	}

	if req.Oracle != nil {
		if err := k.registerOracle(ctx, req.Oracle, nil); nil != err {
			return errors.Wrap(err, "Error setting oracle")
		}
	}

	for _, userKarma := range req.Users {
		userAddr := loom.UnmarshalAddressPB(userKarma.User)
		if err := AddKarma(ctx, userAddr, userKarma.Sources); err != nil {
			return errors.Wrapf(err, "setting karma for user %v", userAddr.String())
		}
	}

	if req.Config == nil {
		if err := ctx.Set(ConfigKey, &ktypes.KarmaConfig{MinKarmaToDeploy: DefaultUpkeepCost}); err != nil {
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
		To:     ctx.ContractAddress().MarshalPB(),
		From:   req.User,
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
		To:     req.User,
		Amount: req.Amount,
	}
	if err := contract.CallMethod(ctx, coinAddr, "Transfer", coinReq, nil); err != nil {
		return errors.Wrap(err, "transferring coin from karma contract")
	}

	amount := req.Amount.Value.Mul(&req.Amount.Value, loom.NewBigUIntFromInt(-1))
	if err := modifyCountForUser(ctx, req.User, CoinDeployToken, &types.BigUInt{Value: *amount}); err != nil {
		return errors.Wrapf(err, "modifying user %v's  upkeep count", req.User.String())
	}
	return nil
}

func (k *Karma) SetConfig(ctx contract.Context, req *ktypes.KarmaConfig) error {
	if hasPermission, _ := ctx.HasPermission(ChangeConfigPermission, []string{oracleRole}); !hasPermission {
		return ErrNotAuthorized
	}
	return SetConfig(ctx, req)
}

func (k *Karma) GetConfig(ctx contract.StaticContext, _ *ktypes.GetConfigRequest) (*ktypes.KarmaConfig, error) {
	return GetConfig(ctx)
}

func SetConfig(ctx contract.Context, cfg *ktypes.KarmaConfig) error {
	if err := ctx.Set(ConfigKey, cfg); err != nil {
		return errors.Wrap(err, "failed to save config")
	}
	return nil
}

func GetConfig(ctx contract.StaticContext) (*ktypes.KarmaConfig, error) {
	var config ktypes.KarmaConfig
	if err := ctx.Get(ConfigKey, &config); err != nil {
		if err == contract.ErrNotFound {
			return &ktypes.KarmaConfig{}, nil
		}
		return nil, err
	}
	return &config, nil
}

// GetOracleAddress returns the address of the oracle, or nil if none is currently set.
func GetOracleAddress(ctx contract.StaticContext) (*loom.Address, error) {
	// TODO: this should be stored in the config
	var oraclePB types.Address
	if err := ctx.Get(OracleKey, &oraclePB); err != nil {
		if err == contract.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	oracleAddr := loom.UnmarshalAddressPB(&oraclePB)
	return &oracleAddr, nil
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

// TODO: request/response types
func (k *Karma) GetUserState(ctx contract.StaticContext, user *types.Address) (*ktypes.KarmaState, error) {
	return GetUserState(ctx, loom.UnmarshalAddressPB(user))
}

func GetUserState(ctx contract.StaticContext, userAddr loom.Address) (*ktypes.KarmaState, error) {
	stateKey, err := UserStateKey(userAddr.MarshalPB())
	if err != nil {
		return nil, err
	}
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
	key, err := UserStateKey(ksu.User)
	if err != nil {
		return err
	}
	return ctx.Set(key, state)
}

func (k *Karma) ResetSources(ctx contract.Context, kpo *ktypes.KarmaSources) error {
	if hasPermission, _ := ctx.HasPermission(ResetSourcesPermission, []string{oracleRole}); !hasPermission {
		return ErrNotAuthorized
	}

	if err := SetAllowedKarmaSources(ctx, kpo.Sources); err != nil {
		return errors.Wrap(err, "failed to set allowed karma sources")
	}
	if err := k.updateKarmaCounts(ctx, ktypes.KarmaSources{Sources: kpo.Sources}); err != nil {
		return errors.Wrap(err, "updating karma counts")
	}
	return nil
}

func SetAllowedKarmaSources(ctx contract.Context, sources []*ktypes.KarmaSourceReward) error {
	return ctx.Set(SourcesKey, &ktypes.KarmaSources{Sources: sources})
}

func (k *Karma) UpdateOracle(ctx contract.Context, params *ktypes.KarmaNewOracle) error {
	if hasPermission, _ := ctx.HasPermission(ChangeOraclePermission, []string{oracleRole}); !hasPermission {
		return ErrNotAuthorized
	}

	currentOracle := ctx.Message().Sender
	return k.registerOracle(ctx, params.NewOracle, &currentOracle)
}

func (k *Karma) AddKarma(ctx contract.Context, req *ktypes.AddKarmaRequest) error {
	if hasPermission, _ := ctx.HasPermission(ChangeUserSourcesPermission, []string{oracleRole}); !hasPermission {
		return ErrNotAuthorized
	}

	return AddKarma(ctx, loom.UnmarshalAddressPB(req.User), req.KarmaSources)
}

// TODO: probably should rename ktypes.KarmaSource -> ktypes.KarmaAmount
func AddKarma(ctx contract.Context, userAddr loom.Address, karmaAmounts []*ktypes.KarmaSource) error {
	state, err := GetUserState(ctx, userAddr)
	if err != nil {
		return err
	}

	for i := range karmaAmounts {
		source := karmaAmounts[i]
		exists := false
		for j := range state.SourceStates {
			if state.SourceStates[j].Name == source.Name {
				total := &state.SourceStates[j].Count.Value
				total.Add(total, &source.Count.Value)
				exists = true
				break
			}
		}
		if !exists {
			state.SourceStates = append(state.SourceStates, source)
		}
	}

	// TODO: don't like how this sources list affects the karma totals for the user,
	//       if the sources list changes gotta remember to recompute all the totals
	var karmaSources ktypes.KarmaSources
	if err := ctx.Get(SourcesKey, &karmaSources); err != nil {
		return errors.Wrap(err, "failed to load list of allowed karma sources")
	}
	state.DeployKarmaTotal, state.CallKarmaTotal = CalculateTotalKarma(karmaSources, *state)

	userStateKey, err := UserStateKey(userAddr.MarshalPB())
	if err != nil {
		return err
	}
	return ctx.Set(userStateKey, state)
}

func (k *Karma) GetUserKarma(ctx contract.StaticContext, userTarget *ktypes.KarmaUserTarget) (*ktypes.KarmaTotal, error) {
	total, err := GetUserKarma(ctx, loom.UnmarshalAddressPB(userTarget.User), userTarget.Target)
	if err != nil {
		return nil, err
	}
	return &ktypes.KarmaTotal{Count: &types.BigUInt{Value: *total}}, nil
}

func GetUserKarma(ctx contract.StaticContext, userAddr loom.Address, target ktypes.KarmaSourceTarget) (*common.BigUInt, error) {
	userState, err := GetUserState(ctx, userAddr)
	if err != nil {
		return nil, err
	}
	if userState.DeployKarmaTotal == nil {
		return common.BigZero(), nil
	}
	if userState.CallKarmaTotal == nil {
		return common.BigZero(), nil
	}

	switch target {
	case ktypes.KarmaSourceTarget_DEPLOY:
		return &userState.DeployKarmaTotal.Value, nil
	case ktypes.KarmaSourceTarget_CALL:
		return &userState.CallKarmaTotal.Value, nil
	default:
		return nil, fmt.Errorf("unknown karma target %v", target)
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
	stateKey, err := UserStateKey(user)
	if err != nil {
		return err
	}
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
			userSourceCounts.SourceStates[i].Count = &types.BigUInt{Value: *newAmount}
			found = true
			break
		}
	}

	if !found {
		// if source for the user does not exist create and set to amount if positive
		if amount.Value.Cmp(common.BigZero()) < 0 {
			return errors.Errorf("not enough karma in source %s. found 0, modifying by %v", user, amount)
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
	key, err := UserStateKey(user)
	if err != nil {
		return err
	}
	if err := ctx.Set(key, &userSourceCounts); err != nil {
		return errors.Wrapf(err, "setting user source counts for %s", user.String())
	}
	return nil
}

var Contract plugin.Contract = contract.MakePluginContract(&Karma{})
