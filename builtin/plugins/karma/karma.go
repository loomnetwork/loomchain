package karma

import (
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/pkg/errors"
)

var (
	oracleKey  = []byte("karma:oracle:key")
	sourcesKey = []byte("karma:sources:key")
)

type (
	State = ktypes.KarmaState
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
	if err := ctx.Set(sourcesKey, &ktypes.KarmaSources{req.Sources}); err != nil {
		return errors.Wrap(err, "kama: Error setting sources")
	}

	if req.Oracle != nil {
		ctx.GrantPermission([]byte(req.Oracle.String()), []string{"oracle"})
		if err := ctx.Set(oracleKey, req.Oracle); err != nil {
			return errors.Wrap(err, "kara: Error setting oracle")
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

func (k *Karma) GetSources(ctx contract.StaticContext, ko *types.Address) (*ktypes.KarmaSources, error) {
	if ctx.Has(sourcesKey) {
		var sources ktypes.KarmaSources
		if err := ctx.Get(sourcesKey, &sources); err != nil {
			return nil, err
		}
		return &sources, nil
	}
	return &ktypes.KarmaSources{}, nil
}

func (k *Karma) GetState(ctx contract.StaticContext, user *types.Address) (*State, error) {
	stateKey := GetUserStateKey(user)
	if ctx.Has(stateKey) {
		var curState State
		if err := ctx.Get(stateKey, &curState); err != nil {
			return nil, err
		}
		return &curState, nil
	}
	return &State{}, nil
}

func (k *Karma) GetTotal(ctx contract.StaticContext, params *types.Address) (*ktypes.KarmaTotal, error) {
	source, err := k.GetSources(ctx, params)
	if err != nil {
		return &ktypes.KarmaTotal{
			Count: 0,
		}, err
	}

	state, err := k.GetState(ctx, params)
	if err != nil {
		return &ktypes.KarmaTotal{
			Count: 0,
		}, err
	}

	var karma int64 = 0
	for _, c := range source.Sources {
		for _, s := range state.SourceStates {
			if c.Name == s.Name {
				karma += c.Reward * s.Count
			}
		}
	}

	return &ktypes.KarmaTotal{
		Count: karma,
	}, nil
}

func (k *Karma) IsOracle(ctx contract.Context, ko *types.Address) (bool, error) {
	return nil == k.validateOracle(ctx, ko), nil
}

func (k *Karma) validateOracle(ctx contract.Context, ko *types.Address) error {
	if ok, _ := ctx.HasPermission([]byte(ko.String()), []string{"oracle"}); !ok {
		return errors.New("karma: Oracle unverified")
	}

	if ok, _ := ctx.HasPermission([]byte(ko.String()), []string{"old-oracle"}); ok {
		return errors.New("karma: This oracle is expired. Please use latest oracle.")
	}

	return nil
}

func (k *Karma) UpdateSourcesForUser(ctx contract.Context, ksu *ktypes.KarmaStateUser) error {
	err := k.validateOracle(ctx, ksu.Oracle)
	if err != nil {
		return err
	}
	return k.validatedUpdateSourcesForUser(ctx, ksu)
}

func (k *Karma) validatedUpdateSourcesForUser(ctx contract.Context, ksu *ktypes.KarmaStateUser) error {
	var state *State
	var err error
	if !ctx.Has(GetUserStateKey(ksu.User)) {
		state = &State{
			SourceStates:   ksu.SourceStates,
			LastUpdateTime: ctx.Now().Unix(),
		}
	} else {
		state, err = k.GetState(ctx, ksu.User)
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

	state, err := k.GetState(ctx, ksu.User)
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

func (k *Karma) UpdateSources(ctx contract.Context, kpo *ktypes.KarmaSourcesValidator) error {
	if err := k.validateOracle(ctx, kpo.Oracle); err != nil {
		return errors.Wrap(err, "karma: validating oracle")
	}
	if err := ctx.Set(sourcesKey, &ktypes.KarmaSources{kpo.Sources}); err != nil {
		return errors.Wrap(err, "kama: Error setting sources")
	}
	return nil
}

func (k *Karma) UpdateOracle(ctx contract.Context, params *ktypes.KarmaNewOracleValidator) error {
	if ctx.Has(oracleKey) {
		if err := k.validateOracle(ctx, params.OldOracle); err != nil {
			return errors.Wrap(err, "karma: validating oracle")
		}
		ctx.GrantPermission([]byte(params.OldOracle.String()), []string{"old-oracle"})
	}
	ctx.GrantPermission([]byte(params.NewOracle.String()), []string{"oracle"})

	if err := ctx.Set(oracleKey, params.NewOracle); err != nil {
		return errors.Wrap(err, "karma: setting new oracle")
	}
	return nil
}

var Contract plugin.Contract = contract.MakePluginContract(&Karma{})
