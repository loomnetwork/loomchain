package karma

import (
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	ktypes "github.com/loomnetwork/loomchain/builtin/plugins/karma/types"
	"github.com/pkg/errors"
	"strings"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/types"
)

var (
	configKey			= []byte("karma:config:key")
)

type (
	Params      = ktypes.KarmaParams
	Config      = ktypes.KarmaConfig
	Source      = ktypes.KarmaSource
	State      	= ktypes.KarmaState
	InitRequest = ktypes.KarmaInitRequest
)

type Karma struct {
}

func (k *Karma) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "Karma",
		Version: "1.0.0",
	}, nil
}

func (k *Karma) Init(ctx contract.Context, req *InitRequest) error {
	return k.createAccount(ctx, req.Params)
}

func GetConfigKey() []byte {
	return configKey
}

func GetUserStateKey(owner string) []byte {
	return []byte("karma:owner:state:" + owner)
}

func (k *Karma) createAccount(ctx contract.Context, params *Params) error {
	karmaOwner := &ktypes.KarmaUser{
		params.OraclePublicAddress,
	}
	owner := strings.TrimSpace(karmaOwner.Address)

	if len(params.Validators) < 1 {
		return errors.New("at least one validator is required")
	}
	config := Config{
		MutableOracle:					params.MutableOracle,
		MaxKarma: 						params.MaxKarma,
		Oracle:				 			karmaOwner,
		Sources: 						params.Sources,
		LastUpdateTime: 				ctx.Now().Unix(),
	}

	ctx.Set(GetConfigKey(), &config)
	if err := ctx.Set(GetConfigKey(), &config); err != nil {
		return errors.Wrap(err, "Error setting state")
	}

	ctx.GrantPermission([]byte(owner), []string{"oracle"})
	for _, v := range params.Validators {
		address := loom.LocalAddressFromPublicKey(v.PubKey)
		ctx.GrantPermission(address, []string{"validator"})
	}

	return nil
}

func (k *Karma) GetConfig(ctx contract.StaticContext,  user *ktypes.KarmaUser) (*Config, error) {
	if ctx.Has(GetConfigKey()) {
		var curConfig Config
		if err := ctx.Get(GetConfigKey(), &curConfig); err != nil {
			return nil, err
		}
		return &curConfig, nil
	}
	return &Config{}, nil
}

func (k *Karma) GetState(ctx contract.StaticContext,  user *ktypes.KarmaUser) (*State, error) {
	stateKey := GetUserStateKey(user.Address)
	if ctx.Has(stateKey) {
		var curState State
		if err := ctx.Get(stateKey, &curState); err != nil {
			return nil, err
		}
		return &curState, nil
	}
	return &State{}, nil
}

func (k *Karma) GetTotal(ctx contract.StaticContext,  params *ktypes.KarmaUser) (*ktypes.KarmaTotal, error) {
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
	for key := range config.Sources {
		if value, ok := state.SourceStates[key]; ok {
			karma += config.Sources[key] * value
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

func (k *Karma) validateOracle(ctx contract.Context,  ko *ktypes.KarmaUser) (error) {
	owner := strings.TrimSpace(ko.Address)
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

func (k *Karma) isValidator(ctx contract.Context,  v *types.Validator) (error) {
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

func (k *Karma) UpdateSourcesForUser(ctx contract.Context,  ksu *ktypes.KarmaStateUser) (error) {
	err := k.validateOracle(ctx, ksu.Oracle)
	if err != nil {
		return err
	}

	var state *State
	user := strings.TrimSpace(ksu.User.Address)
	if !ctx.Has(GetUserStateKey(user)) {
		state = &State{
			SourceStates: 					ksu.SourceStates,
			LastUpdateTime: 				ctx.Now().Unix(),
		}
	}else{
		state, err = k.GetState(ctx, ksu.User)
		if err != nil {
			return err
		}

		for k, v := range ksu.SourceStates {
			state.SourceStates[k] = v
		}
		state.LastUpdateTime = ctx.Now().Unix()
	}


	err = ctx.Set(GetUserStateKey(ksu.User.Address), state)

	return err
}

func (k *Karma) DeleteSourcesForUser(ctx contract.Context,  ksu *ktypes.KarmaStateKeyUser) (error) {
	err := k.validateOracle(ctx, ksu.Oracle)
	if err != nil {
		return err
	}

	user := strings.TrimSpace(ksu.User.Address)
	if !ctx.Has(GetUserStateKey(user)) {
		return errors.New("user karma sources does not exist")
	}

	state, err := k.GetState(ctx, ksu.User)
	if err != nil {
		return err
	}

	for k := range ksu.StateKeys {
		delete(state.SourceStates, ksu.StateKeys[k])
	}

	state.LastUpdateTime = ctx.Now().Unix()
	return ctx.Set(GetUserStateKey(ksu.User.Address), state)
}

func (k *Karma) UpdateConfig(ctx contract.Context,  kpo *ktypes.KarmaParamsValidator) (error) {
	err := k.isValidator(ctx, kpo.Validator)
	if err != nil {
		return err
	}

	newConfig := &Config{
		MutableOracle:					kpo.Params.MutableOracle,
		MaxKarma: 						kpo.Params.MaxKarma,
		Oracle:				 			&ktypes.KarmaUser{
			kpo.Params.OraclePublicAddress,
		},
		Sources: 						kpo.Params.Sources,
		LastUpdateTime: 				ctx.Now().Unix(),
	}

	oldOwner := strings.TrimSpace(kpo.Oracle.Address)
	ctx.GrantPermission([]byte(oldOwner), []string{"old-oracle"})

	newOwner := strings.TrimSpace(kpo.Params.OraclePublicAddress)
	ctx.GrantPermission([]byte(newOwner), []string{"oracle"})
	return ctx.Set(GetConfigKey(), newConfig)
}

func (k *Karma) getConfigAfterValidation(ctx contract.Context,  ko *types.Validator) (*Config, error) {
	err := k.isValidator(ctx, ko)
	if err != nil {
		return nil, err
	}
	config, err := k.GetConfig(ctx, nil)
	if err != nil {
		return nil, err
	}
	return config, err
}

func (k *Karma) UpdateConfigMaxKarma(ctx contract.Context,  params *ktypes.KarmaParamsValidatorNewMaxKarma) (error) {
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

func (k *Karma) UpdateConfigOracleMutability(ctx contract.Context,  params *ktypes.KarmaParamsMutableValidator) (error) {
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

func (k *Karma) UpdateConfigOracle(ctx contract.Context,  params *ktypes.KarmaParamsValidatorNewOracle) (error) {
	config, err := k.getConfigAfterValidation(ctx, params.Validator)
	if err != nil {
		return err
	}
	if !config.MutableOracle {
		return errors.New("oracle is not mutable")
	}
	config.Oracle = &ktypes.KarmaUser{
		Address:params.NewOraclePublicAddress,
	}
	return ctx.Set(GetConfigKey(), config)
}

var Contract plugin.Contract = contract.MakePluginContract(&Karma{})
