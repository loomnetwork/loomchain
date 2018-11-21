package karma

import (
	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/pkg/errors"
)

var (
	OracleKey  = []byte("karma:oracle:key")
	SourcesKey = []byte("karma:sources:key")
	RunningCostKey = []byte("karma:running-cost:key")
)

const (
	DeployToken = "deploy-token"
)

type Karma struct {
}

func (k *Karma) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "karma",
		Version: "1.0.0",
	}, nil
}

func (k *Karma) Init(ctx contract.Context, req *ktypes.KarmaInitRequest) error {
	if err := ctx.Set(SourcesKey, &ktypes.KarmaSources{req.Sources}); err != nil {
		return errors.Wrap(err, "Error setting sources")
	}

	if req.Oracle != nil {
		ctx.GrantPermissionTo(loom.UnmarshalAddressPB(req.Oracle), []byte(req.Oracle.String()), "oracle")
		if err := ctx.Set(OracleKey, req.Oracle); err != nil {
			return errors.Wrap(err, "Error setting oracle")
		}
	}

	for _, user := range req.Users {
		ksu := &ktypes.KarmaStateUser{
			User:         user.User,
			Oracle:       req.Oracle,
			SourceStates: make([]*ktypes.KarmaSource, 0),
		}
		for _, source := range user.Sources {
			ksu.SourceStates = append(ksu.SourceStates, source)
		}
		if err := k.validatedUpdateSourcesForUser(ctx, ksu); err != nil {
			return errors.Wrapf(err, "updating source for user %v ", ksu.User)
		}
	}

	return nil
}

func GetUserStateKey(owner *types.Address) []byte {
	return []byte("karma:owner:state:" + owner.String())
}

func (k *Karma) DepositCoin(ctx contract.Context, user *types.Address, amount *loom.BigUInt) (*loom.BigUInt, error) {
	newAmoutn, err := modifyCountForUser(ctx, user, DeployToken, amount.Int64())
	return loom.NewBigUIntFromInt(newAmoutn), err
}

func (k *Karma) WithdrawCoin(ctx contract.Context, user *types.Address, amount *loom.BigUInt) (*loom.BigUInt, error) {
	newAmoutn, err := modifyCountForUser(ctx, user, DeployToken, -1*amount.Int64())
	return loom.NewBigUIntFromInt(newAmoutn), err
}

func modifyCountForUser(ctx contract.Context, user *types.Address, sourceName string, amount int64) (int64, error) {
	stateKey := GetUserStateKey(user)
	if !ctx.Has(stateKey) {
		return 0, errors.Errorf("user %s not found", user.String())
	}

	var userSourceCounts ktypes.KarmaState
	// If user source counts not found, We want to create a new source count
	if err := ctx.Get(stateKey, &userSourceCounts); err != nil && err != contract.ErrNotFound {
		return 0, errors.Wrapf(err, "source counts for user %s", user.String())
	}

	for i, source := range userSourceCounts.SourceStates {
		if source.Name == sourceName {
			if 0 > userSourceCounts.SourceStates[i].Count + amount {
				return 0, errors.Errorf("not enough karma in source %s. found %v, modifying by %v", sourceName, userSourceCounts.SourceStates[i].Count, amount)
			}
			userSourceCounts.SourceStates[i].Count += amount
			if err := ctx.Set(GetUserStateKey(user), &userSourceCounts); err != nil {
				return 0, errors.Wrapf(err, "setting user source counts for %s", user.String())
			}
			return userSourceCounts.SourceStates[i].Count, nil
		}
	}

	// if source for the user does not exist create and set to amount if positive
	if amount < 0 {
		return 0, errors.Errorf("not enough karma in source %s. found 0, modifying by %v", user, amount)
	}
	userSourceCounts.SourceStates = append(userSourceCounts.SourceStates, &ktypes.KarmaSource{
		Name: sourceName,
		Count: amount,
	})
	if err := ctx.Set(GetUserStateKey(user), &userSourceCounts); err != nil {
		return 0, errors.Wrapf(err, "setting user source counts for %s", user.String())
	}
	return amount, nil
}

func (k Karma) SetRunningCost(ctx contract.Context, costPerHour *loom.BigUInt) error {
	caller := ctx.Message().Sender.MarshalPB()
	if err := k.validateOracle(ctx, caller); err != nil {
		return errors.Wrap(err, "validating oracle")
	}
	if err := ctx.Set(RunningCostKey, &types.BigUInt{Value: *costPerHour}); err != nil {
		return errors.Wrap(err, "setting running cost")
	}
	return nil
}

func (k *Karma) GetSources(ctx contract.StaticContext, ko *types.Address) (*ktypes.KarmaSources, error) {
	if ctx.Has(SourcesKey) {
		var sources ktypes.KarmaSources
		if err := ctx.Get(SourcesKey, &sources); err != nil {
			return nil, err
		}
		return &sources, nil
	}
	return &ktypes.KarmaSources{}, nil
}

func (k *Karma) GetUserState(ctx contract.StaticContext, user *types.Address) (*ktypes.KarmaState, error) {
	stateKey := GetUserStateKey(user)
	if ctx.Has(stateKey) {
		var curState ktypes.KarmaState
		if err := ctx.Get(stateKey, &curState); err != nil {
			return nil, err
		}
		return &curState, nil
	}
	return &ktypes.KarmaState{}, nil
}

func (k *Karma) GetTotal(ctx contract.StaticContext, params *types.Address) (*ktypes.KarmaTotal, error) {
	source, err := k.GetSources(ctx, params)
	if err != nil {
		return &ktypes.KarmaTotal{
			Count: 0,
		}, err
	}

	state, err := k.GetUserState(ctx, params)
	if err != nil {
		return &ktypes.KarmaTotal{
			Count: 0,
		}, err
	}

	return &ktypes.KarmaTotal{
		Count: CalculateTotalKarma(*source, *state, ktypes.SourceTarget_ALL),
	}, nil
}

func CalculateTotalKarma(karmaSources ktypes.KarmaSources, karmaStates ktypes.KarmaState, target ktypes.SourceTarget) int64 {
	var karmaValue = int64(0)
	for _, c := range karmaSources.Sources {
		for _, s := range karmaStates.SourceStates {
			if c.Name == s.Name && (c.Target == target || target == ktypes.SourceTarget_ALL) {
				karmaValue += c.Reward * s.Count
			}
		}
	}
	return karmaValue
}

func (k *Karma) validateOracle(ctx contract.Context, ko *types.Address) error {
	if ok, _ := ctx.HasPermission([]byte(ko.String()), []string{"oracle"}); !ok {
		return errors.New("Oracle unverified")
	}

	if ok, _ := ctx.HasPermission([]byte(ko.String()), []string{"old-oracle"}); ok {
		return errors.New("This oracle is expired. Please use latest oracle.")
	}
	return nil
}

func (k *Karma) AppendSourcesForUser(ctx contract.Context, ksu *ktypes.KarmaStateUser) error {
	err := k.validateOracle(ctx, ksu.Oracle)
	if err != nil {
		return err
	}
	return k.validatedUpdateSourcesForUser(ctx, ksu)
}

func (k *Karma) validatedUpdateSourcesForUser(ctx contract.Context, ksu *ktypes.KarmaStateUser) error {
	var state *ktypes.KarmaState
	var err error
	if !ctx.Has(GetUserStateKey(ksu.User)) {
		state = &ktypes.KarmaState{
			SourceStates:   ksu.SourceStates,
			LastUpdateTime: ctx.Now().Unix(),
		}
	} else {
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
	}
	err = ctx.Set(GetUserStateKey(ksu.User), state)

	return err
}

func (k *Karma) DeleteSourcesForUser(ctx contract.Context, ksu *ktypes.KarmaStateKeyUser) error {
	err := k.validateOracle(ctx, ksu.Oracle)
	if err != nil {
		return err
	}

	if !ctx.Has(GetUserStateKey(ksu.User)) {
		return errors.New("user karma sources does not exist")
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

	state.LastUpdateTime = ctx.Now().Unix()
	return ctx.Set(GetUserStateKey(ksu.User), state)
}

func (k *Karma) ResetSources(ctx contract.Context, kpo *ktypes.KarmaSourcesValidator) error {

	if err := k.validateOracle(ctx, kpo.Oracle); err != nil {
		return errors.Wrap(err, "validating oracle")
	}
	if err := ctx.Set(SourcesKey, &ktypes.KarmaSources{kpo.Sources}); err != nil {
		return errors.Wrap(err, "Error setting sources")
	}
	return nil
}

func (k *Karma) UpdateOracle(ctx contract.Context, params *ktypes.KarmaNewOracleValidator) error {
	if ctx.Has(OracleKey) {
		if err := k.validateOracle(ctx, params.OldOracle); err != nil {
			return errors.Wrap(err, "validating oracle")
		}
		ctx.GrantPermission([]byte(params.OldOracle.String()), []string{"old-oracle"})
	}
	ctx.GrantPermission([]byte(params.NewOracle.String()), []string{"oracle"})

	if err := ctx.Set(OracleKey, params.NewOracle); err != nil {
		return errors.Wrap(err, "setting new oracle")
	}
	return nil
}

var Contract plugin.Contract = contract.MakePluginContract(&Karma{})
