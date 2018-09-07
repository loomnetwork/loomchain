package karma

import (
	`github.com/loomnetwork/go-loom`
	"sort"
	"strings"
	
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/pkg/errors"
)

var (
	ConfigKey = []byte("karma:config:key")
	OracleKey = []byte("karma:oracle:key")
)

type (
	Params        = ktypes.KarmaParams
	Config        = ktypes.KarmaConfig
	Source        = ktypes.KarmaSource
	SourceReward  = ktypes.KarmaSourceReward
	State         = ktypes.KarmaState
	InitRequest   = ktypes.KarmaInitRequest
	AddressSource = ktypes.KarmaAddressSource

)

type Karma struct {
}

func (k *Karma) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "karma",
		Version: "1.0.0",
	}, nil
}

func (k *Karma) Init(ctx contract.Context, req *InitRequest) error {
	return k.createAccount(ctx, req.Params)
}

func GetConfigKey() []byte {
	return ConfigKey
}

func GetUserStateKey(owner *types.Address) []byte {
	return []byte("karma:owner:state:" + owner.String())
}

func (k *Karma) createAccount(ctx contract.Context, params *Params) error {
	
	owner := strings.TrimSpace(params.Oracle.String())
	
	var config Config
	if params.Config != nil {
		sort.Slice(params.Config.Sources, func(i, j int) bool {
			return params.Config.Sources[i].Name < params.Config.Sources[j].Name
		})
		
		config = Config{
			Enabled:                params.Config.Enabled,
			MutableOracle:          params.Config.MutableOracle,
			Sources:                params.Config.Sources,
			SessionMaxAccessCount:  params.Config.SessionMaxAccessCount,
			SessionDuration:        params.Config.SessionDuration,
			DeployEnabled:          params.Config.DeployEnabled,
			CallEnabled:            params.Config.CallEnabled,
			LastUpdateTime: ctx.Now().Unix(),
		}
		
	}
	
	if err := ctx.Set(GetConfigKey(), &config); err != nil {
		return errors.Wrap(err, "Error setting config")
	}
	ctx.GrantPermissionTo(loom.UnmarshalAddressPB(params.Oracle), []byte(owner), "oracle")
	
	if err := ctx.Set(OracleKey, params.Oracle); err != nil {
		return errors.Wrap(err, "Error setting oracle")
	}
	
	for _, user :=range params.Users {
		ksu:= &ktypes.KarmaStateUser{
			User: user.User,
			Oracle: params.Oracle,
			SourceStates: make([]*ktypes.KarmaSource, 0),
		}
		for _, source := range user.Sources {
			ksu.SourceStates = append(ksu.SourceStates, source)
		}
		if err := k.validatedUpdateSourcesForUser(ctx, ksu); err != nil {
			return errors.Wrapf(err,"updating source for user %v ", ksu.User)
		}
	}

	return nil
}

func (k *Karma) GetConfig(ctx contract.StaticContext, ko *types.Address) (*Config, error) {
	if ctx.Has(GetConfigKey()) {
		var curConfig Config
		if err := ctx.Get(GetConfigKey(), &curConfig); err != nil {
			return nil, err
		}
		return &curConfig, nil
	}
	return &Config{}, nil
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
	config, err := k.GetConfig(ctx, params)
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
	for _, c := range config.Sources {
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

func (k *Karma) validateOracle(ctx contract.Context, ko *types.Address) error {
	owner := strings.TrimSpace(ko.String())
	var config ktypes.KarmaConfig
	if err := ctx.Get(GetConfigKey(), &config); err != nil {
		return err
	}
	
	if ok, _ := ctx.HasPermission([]byte(owner), []string{"oracle"}); !ok {
		return errors.New("Oracle unverified")
	}
	
	if ok, _ := ctx.HasPermission([]byte(owner), []string{"old-oracle"}); ok {
		return errors.New("This oracle is expired. Please use latest oracle.")
	}
	
	return nil
	
}

func (k *Karma) UpdateSourcesForUser(ctx contract.Context, ksu *ktypes.KarmaStateUser) error {
	err := k.validateOracle(ctx, ksu.Oracle)
	if err != nil {
		return err
	}
	return k.validatedUpdateSourcesForUser(ctx, ksu);
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
	
	sort.Slice(state.SourceStates, func(i, j int) bool {
		return state.SourceStates[i].Name < state.SourceStates[j].Name
	})
	
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

func (k *Karma) UpdateConfig(ctx contract.Context, kpo *ktypes.KarmaConfigValidator) error {
	config, err := k.GetConfig(ctx, nil)
	if err != nil {
		return err
	}
	
	if !config.MutableOracle {
		return errors.New("oracle is not mutable")
	}
	
	newConfig := &Config{
		Enabled:                kpo.Enabled,
		SessionMaxAccessCount:  kpo.SessionMaxAccessCount,
		SessionDuration:        kpo.SessionDuration,
		DeployEnabled:          kpo.DeployEnabled,
		CallEnabled:            kpo.CallEnabled,
		LastUpdateTime: ctx.Now().Unix(),
	}
	ctx.GrantPermission([]byte(kpo.Oracle.String()), []string{"oracle"})
	return ctx.Set(GetConfigKey(), newConfig)
}

func (k *Karma) getConfigAfterValidation(ctx contract.Context, ko *types.Address) (*Config, error) {
	if err := k.validateOracle(ctx, ko); err != nil {
		return nil, errors.Wrap(err, "validating oracle")
	}
	
	config, err := k.GetConfig(ctx, nil)
	if err != nil {
		return nil, err
	}
	return config, err
}

func (k *Karma) UpdateConfigOracleMutability(ctx contract.Context, params *ktypes.KarmaParamsMutableValidator) error {
	config, err := k.getConfigAfterValidation(ctx, params.Oracle)
	if err != nil {
		return err
	}
	if !config.MutableOracle {
		return errors.New("oracle is not mutable")
	}
	config.MutableOracle = params.MutableOracle
	return ctx.Set(GetConfigKey(), config)
}

func (k *Karma) UpdateConfigOracle(ctx contract.Context, params *ktypes.KarmaParamsValidatorNewOracle) error {
	config, err := k.getConfigAfterValidation(ctx, params.OldOracle)
	if err != nil {
		return err
	}
	if !config.MutableOracle {
		return errors.New("oracle is not mutable")
	}

	ctx.GrantPermissionTo(loom.UnmarshalAddressPB(params.NewOracle), []byte(params.NewOracle.String()), "oracle")
	if err := ctx.Set(OracleKey, params.NewOracle); err != nil {
		return errors.Wrap(err, "Error setting oracle")
	}
	return ctx.Set(GetConfigKey(), config)
}

var Contract plugin.Contract = contract.MakePluginContract(&Karma{})