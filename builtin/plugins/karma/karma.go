package karma

import (
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	ktypes "github.com/loomnetwork/loomchain/builtin/plugins/karma/types"
	"github.com/loomnetwork/go-loom/util"
)

var (
	stateKey      = []byte("karmaState")
	configKey      = []byte("karmaConfig")
)

type (
	Params      = ktypes.KarmaParams
	InitRequest = ktypes.KarmaInitRequest
	Config      = ktypes.KarmaConfig
	State      	= ktypes.KarmaState
)

func getStateKey(owner string) []byte {
	return util.PrefixKey(stateKey, []byte(owner))
}

type Karma struct {
}

func (k *Karma) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "Karma",
		Version: "1.0.0",
	}, nil
}

func (k *Karma) Init(ctx contract.Context, req *InitRequest) error {
	params := req.Params
	config := &Config{
		SmsKarma:           params.SmsKarma,
		OauthKarma:        	params.OauthKarma,
		TokenKarma: 		params.TokenKarma,
		LastUpdateTime: 	ctx.Now().Unix(),
	}
	return ctx.Set(configKey, config)
}

func (k *Karma) GetConfig(ctx contract.StaticContext,  params *ktypes.KarmaOwner) (*Config, error) {
	if ctx.Has(configKey) {
		var curConfig Config
		if err := ctx.Get(configKey, &curConfig); err != nil {
			return nil, err
		}
		return &curConfig, nil
	}
	return &Config{}, nil
}

func (k *Karma) GetState(ctx contract.StaticContext,  params *ktypes.KarmaOwner) (*State, error) {
	if ctx.Has(stateKey) {
		var curState State
		if err := ctx.Get(stateKey, &curState); err != nil {
			return nil, err
		}
		return &curState, nil
	}
	return &State{}, nil
}

func (k *Karma) GetTotalKarma(ctx contract.StaticContext,  params *ktypes.KarmaOwner) (*ktypes.KarmaTotal, error) {
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

	if state.SmsKarma {
		karma += config.SmsKarma
	}

	if state.OauthKarma {
		karma += config.OauthKarma
	}

	//TODO: figure out a way to get loom token counts
	if state.TokenCount > 0 {
		karma += state.TokenCount * config.TokenKarma
	}

	return &ktypes.KarmaTotal{
		Count: karma,
	}, nil
}



func (k *Karma) SetSmsKarma(ctx contract.Context,  params *ktypes.KarmaOwner, status bool) (error) {
	state, err := k.GetState(ctx, params)
	if err != nil {
		return err
	}
	state.SmsKarma = status
	state.LastUpdateTime = ctx.Now().Unix()
	return ctx.Set(getStateKey(params.Owner), state)
}

func (k *Karma) EnableSmsKarma(ctx contract.Context,  params *ktypes.KarmaOwner) (error) {
	return k.SetSmsKarma(ctx, params, true)
}

func (k *Karma) DisableSmsKarma(ctx contract.Context,  params *ktypes.KarmaOwner) (error) {
	return k.SetSmsKarma(ctx, params, false)
}

func (k *Karma) SetOauthKarma(ctx contract.Context,  params *ktypes.KarmaOwner, status bool) (error) {
	state, err := k.GetState(ctx, params)
	if err != nil {
		return err
	}
	state.OauthKarma = status
	state.LastUpdateTime = ctx.Now().Unix()
	return ctx.Set(getStateKey(params.Owner), state)
}

func (k *Karma) EnableOauthKarma(ctx contract.Context,  params *ktypes.KarmaOwner) (error) {
	return k.SetSmsKarma(ctx, params, true)
}

func (k *Karma) DisableOauthKarma(ctx contract.Context,  params *ktypes.KarmaOwner) (error) {
	return k.SetSmsKarma(ctx, params, false)
}

func (k *Karma) SetTokenKarma(ctx contract.Context,  params *ktypes.KarmaOwnerToken) (error) {
	state, err := k.GetState(ctx, params.Owner)
	if err != nil {
		return err
	}
	state.TokenCount = params.TokenCount
	state.LastUpdateTime = ctx.Now().Unix()
	return ctx.Set(getStateKey(params.Owner.Owner), state)
}



var Contract plugin.Contract = contract.MakePluginContract(&Karma{})
