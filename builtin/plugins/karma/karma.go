package karma

import (
	"sort"
	"strings"
	
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/pkg/errors"
)

var (
	configKey = []byte("karma:config:key")
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
	return configKey
}

func GetUserStateKey(owner *types.Address) []byte {
	return []byte("karma:owner:state:" + owner.String())
}

func (k *Karma) createAccount(ctx contract.Context, params *Params) error {
	
	owner := strings.TrimSpace(params.Oracle.String())
	
	//if len(params.Validators) < 1 {
	//	return errors.New("at least one validator is required")
	//}
	
	sort.Slice(params.Sources, func(i, j int) bool {
		return params.Sources[i].Name < params.Sources[j].Name
	})
	
	config := Config{
		MutableOracle:  params.MutableOracle,
		MaxKarma:       params.MaxKarma,
		Oracle:         params.Oracle,
		Sources:        params.Sources,
		LastUpdateTime: ctx.Now().Unix(),
	}
	
	if err := ctx.Set(GetConfigKey(), &config); err != nil {
		return errors.Wrap(err, "Error setting config")
	}
	
	ctx.GrantPermission([]byte(owner), []string{"oracle"})
	
	for _, user :=range params.Users {
		ksu:= &ktypes.KarmaStateUser{
			User: user.User,
			Oracle: params.Oracle,
			SourceStates: make([]*ktypes.KarmaSource, 0),
		}
		for _, source := range user.Sources {
			ksu.SourceStates = append(ksu.SourceStates, source)
		}
		if err := k.UpdateSourcesForUser(ctx, ksu); err != nil {
			return errors.Wrapf(err,"updating source for user %v ", ksu.User)
		}
	}
	
	//for _, v := range params.Validators {
	//	address := loom.LocalAddressFromPublicKey(v.PubKey)
	//	ctx.GrantPermission(address, []string{"validator"})
	//}
	
	return nil
}

func (k *Karma) GetConfig(ctx contract.StaticContext, user *types.Address) (*Config, error) {
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
	
	if karma > config.MaxKarma {
		karma = config.MaxKarma
	}
	
	if karma > config.MaxKarma {
		karma = config.MaxKarma
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
/*
func (k *Karma) isValidator(ctx contract.Context, v *types.Validator) error {
	address := loom.LocalAddressFromPublicKey(v.PubKey)
	var config ktypes.KarmaConfig
	if err := ctx.Get(GetConfigKey(), &config); err != nil {
		return err
	}
	
	if ok, _ := ctx.HasPermission(address, []string{"validator"}); !ok {
		return errors.New("validator unverified")
	}
	
	if ok, _ := ctx.HasPermission(address, []string{"old-validator"}); ok {
		return errors.New("this validator is expired.")
	}
	
	return nil
	
}
*/
func (k *Karma) UpdateSourcesForUser(ctx contract.Context, ksu *ktypes.KarmaStateUser) error {
	err := k.validateOracle(ctx, ksu.Oracle)
	if err != nil {
		return err
	}
	
	var state *State
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

func (k *Karma) UpdateConfig(ctx contract.Context, kpo *ktypes.KarmaParamsValidator) error {
	//err := k.isValidator(ctx, kpo.Validator)
	//if err != nil {
	//	return err
	//}
	
	config, err := k.GetConfig(ctx, nil)
	if err != nil {
		return err
	}
	
	if !config.MutableOracle {
		return errors.New("oracle is not mutable")
	}
	
	sort.Slice(kpo.Params.Sources, func(i, j int) bool {
		return kpo.Params.Sources[i].Name < kpo.Params.Sources[j].Name
	})
	
	newConfig := &Config{
		MutableOracle:  kpo.Params.MutableOracle,
		MaxKarma:       kpo.Params.MaxKarma,
		Oracle:         kpo.Params.Oracle,
		Sources:        kpo.Params.Sources,
		LastUpdateTime: ctx.Now().Unix(),
	}
	
	ctx.GrantPermission([]byte(kpo.Oracle.String()), []string{"old-oracle"})
	
	ctx.GrantPermission([]byte(kpo.Params.Oracle.String()), []string{"oracle"})
	return ctx.Set(GetConfigKey(), newConfig)
}

func (k *Karma) getConfigAfterValidation(ctx contract.Context, ko *types.Validator) (*Config, error) {
	//err := k.isValidator(ctx, ko)
	//if err != nil {
	//	return nil, err
	//}
	config, err := k.GetConfig(ctx, nil)
	if err != nil {
		return nil, err
	}
	return config, err
}

func (k *Karma) UpdateConfigMaxKarma(ctx contract.Context, params *ktypes.KarmaParamsValidatorNewMaxKarma) error {
	config, err := k.getConfigAfterValidation(ctx, params.Validator)
	if err != nil {
		return err
	}
	if !config.MutableOracle {
		return errors.New("oracle is not mutable")
	}
	config.MaxKarma = params.MaxKarma
	return ctx.Set(GetConfigKey(), config)
}

func (k *Karma) UpdateConfigOracleMutability(ctx contract.Context, params *ktypes.KarmaParamsMutableValidator) error {
	config, err := k.getConfigAfterValidation(ctx, params.Validator)
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
	config, err := k.getConfigAfterValidation(ctx, params.Validator)
	if err != nil {
		return err
	}
	if !config.MutableOracle {
		return errors.New("oracle is not mutable")
	}
	config.Oracle = params.NewOracle
	return ctx.Set(GetConfigKey(), config)
}

var Contract plugin.Contract = contract.MakePluginContract(&Karma{})