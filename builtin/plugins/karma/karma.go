package karma

import (
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	ktypes "github.com/loomnetwork/loomchain/builtin/plugins/karma/types"

	"github.com/pkg/errors"
	"strings"
)

var (
	configKey      = []byte("karma:config:key")
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

func (k *Karma) getConfigKey() []byte {
	return configKey
}

func (k *Karma) getUserStateKey(owner string) []byte {
	return []byte("karma:owner:state:" + owner)
}

func (k *Karma) createAccount(ctx contract.Context, params *Params) error {
	karmaOwner := &ktypes.KarmaUser{
		params.OraclePublicAddress,
	}

	owner := strings.TrimSpace(karmaOwner.Address)
	// confirm owner doesnt exist already
	if ctx.Has(k.getConfigKey()) {
		return errors.New("Owner already exists")
	}

	config := Config{
		MaxKarma: 						params.MaxKarma,
		Oracle:				 			karmaOwner,
		Sources: 						params.Sources,
		LastUpdateTime: 				ctx.Now().Unix(),
	}

	if err := ctx.Set(k.getConfigKey(), &config); err != nil {
		return errors.Wrap(err, "Error setting state")
	}

	ctx.GrantPermission([]byte(owner), []string{"oracle"})
	return nil
}

func (k *Karma) GetConfig(ctx contract.StaticContext,  user *ktypes.KarmaUser) (*Config, error) {
	if ctx.Has(k.getConfigKey()) {
		var curConfig Config
		if err := ctx.Get(k.getConfigKey(), &curConfig); err != nil {
			return nil, err
		}
		return &curConfig, nil
	}
	return &Config{}, nil
}

func (k *Karma) GetState(ctx contract.StaticContext,  user *ktypes.KarmaUser) (*State, error) {
	stateKey := k.getUserStateKey(user.Address)
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

	var karma float64 = 0
	for key := range config.Sources {
		if value, ok := state.SourceStates[key]; ok {
			karma += config.Sources[key] * value
		}
		//TODO: This is for debuging without oracles
		/*else{
			karma += config.Sources[key]
		}*/
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
	if err := ctx.Get(k.getConfigKey(), &config); err != nil {
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

func (k *Karma) AddNewSourcesForUser(ctx contract.Context,  ksu *ktypes.KarmaStateUser) (error) {
	err := k.validateOracle(ctx, ksu.Oracle)
	if err != nil {
		return err
	}

	user := strings.TrimSpace(ksu.User.Address)
	// confirm user doesnt exist already
	if ctx.Has(k.getUserStateKey(user)) {
		return errors.New("User karma sources already exists. Use UpdateSourcesForUser call change values.")
	}

	state := State{
		SourceStates: 					ksu.SourceStates,
		LastUpdateTime: 				ctx.Now().Unix(),
	}
	if err := ctx.Set(k.getUserStateKey(user), &state); err != nil {
		return errors.Wrap(err, "Error setting karma state")
	}
	return nil
}

func (k *Karma) UpdateSourcesForUser(ctx contract.Context,  ksu *ktypes.KarmaStateUser) (error) {
	err := k.validateOracle(ctx, ksu.Oracle)
	if err != nil {
		return err
	}

	user := strings.TrimSpace(ksu.User.Address)
	if !ctx.Has(k.getUserStateKey(user)) {
		return errors.New("User karma sources does not exist")
	}

	state, err := k.GetState(ctx, ksu.User)
	if err != nil {
		return err
	}

	for k, v := range ksu.SourceStates {
		state.SourceStates[k] = v
	}

	state.LastUpdateTime = ctx.Now().Unix()
	return ctx.Set(k.getUserStateKey(ksu.User.Address), state)
}

func (k *Karma) DeleteSourcesForUser(ctx contract.Context,  ksu *ktypes.KarmaStateKeyUser) (error) {
	err := k.validateOracle(ctx, ksu.Oracle)
	if err != nil {
		return err
	}

	user := strings.TrimSpace(ksu.User.Address)
	if !ctx.Has(k.getUserStateKey(user)) {
		return errors.New("User karma sources does not exist")
	}

	state, err := k.GetState(ctx, ksu.User)
	if err != nil {
		return err
	}

	for k := range ksu.StateKeys {
		delete(state.SourceStates, ksu.StateKeys[k])
	}

	state.LastUpdateTime = ctx.Now().Unix()
	return ctx.Set(k.getUserStateKey(ksu.User.Address), state)
}

func (k *Karma) UpdateConfig(ctx contract.Context,  kpo *ktypes.KarmaParamsOracle) (error) {
	err := k.validateOracle(ctx, kpo.Oracle)
	if err != nil {
		return err
	}

	newConfig := &Config{
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
	return ctx.Set(k.getConfigKey(), newConfig)
}

func (k *Karma) getConfigAfterOracleValidation(ctx contract.Context,  ko *ktypes.KarmaUser) (*Config, error) {
	err := k.validateOracle(ctx, ko)
	if err != nil {
		return nil, err
	}
	config, err := k.GetConfig(ctx, ko)
	if err != nil {
		return nil, err
	}
	return config, err
}

func (k *Karma) UpdateConfigMaxKarma(ctx contract.Context,  params *ktypes.KarmaParamsOracleNewMaxKarma) (error) {
	config, err := k.getConfigAfterOracleValidation(ctx, params.Oracle)
	if err != nil {
		return err
	}
	config.MaxKarma = params.MaxKarma
	return ctx.Set(k.getConfigKey(), config)
}

func (k *Karma) UpdateConfigOracle(ctx contract.Context,  params *ktypes.KarmaParamsOracleNewOracle) (error) {
	config, err := k.getConfigAfterOracleValidation(ctx, params.Oracle)
	if err != nil {
		return err
	}
	config.Oracle = &ktypes.KarmaUser{
		Address:params.NewOraclePublicAddress,
	}
	return ctx.Set(k.getConfigKey(), config)
}

var Contract plugin.Contract = contract.MakePluginContract(&Karma{})
