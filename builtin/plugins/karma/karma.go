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
)

const (
	DeployToken        = "deploy-token"
	UserStateKeyPrefix = "user_state:"
	oracleRole         = "karma_role_oracle"
)

var (
	OracleKey  = []byte("karma:oracle:key")
	SourcesKey = []byte("karma:sources:key")

	ChangeOraclePermission      = []byte("change_oracle")
	ChangeUserSourcesPermission = []byte("change_user_sources")
	ResetSourcesPermission      = []byte("reset_sources")

	ErrNotAuthorized = errors.New("sender is not authorized to call this method")
)

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

	return nil
}

func (k *Karma) DepositCoin(ctx contract.Context, req *ktypes.KarmaUserAmount) error {
	_, err := modifyCountForUser(ctx, req.User, DeployToken, req.Amount.Value.Int64())
	if err := k.updateUserKarmaState(ctx, req.User); err != nil {
		return err
	}

	karmaAddr, err := ctx.Resolve("karma")
	coinReq := &coin.TransferFromRequest{
		To: 	karmaAddr.MarshalPB(),
		From: 	req.User,
		Amount: req.Amount,
	}
	coinAddr, err := ctx.Resolve("coin")

	if err != nil {
		return errors.Wrap(err, "address of coin contract")
	}
	err = contract.CallMethod(ctx, coinAddr, "TransferFrom", coinReq, nil)
	if err != nil {
		return errors.Wrapf(err, "transfer %v to %v", req.Amount, req.User)
	}

	return err
}

func (k *Karma) WithdrawCoin(ctx contract.Context, req *ktypes.KarmaUserAmount) error {
	_, err := modifyCountForUser(ctx, req.User, DeployToken, -1*req.Amount.Value.Int64())
	if err := k.updateUserKarmaState(ctx, req.User); err != nil {
		return err
	}

	karmaAddr, err := ctx.Resolve("karma")
    coinReq := &coin.TransferFromRequest{
        To: 	req.User,
        From: 	karmaAddr.MarshalPB(),
        Amount: req.Amount,
    }
    coinAddr, err := ctx.Resolve("coin")
    if err != nil {
        return errors.Wrap(err, "address of coin contract")
    }
    err = contract.CallMethod(ctx, coinAddr, "TransferFromRequest", coinReq, nil)
    if err != nil {
        return errors.Wrapf(err, "transfer %v to %v", req.Amount, req.User)
    }

	return err
}

func (k *Karma) GetSources(ctx contract.StaticContext, ko *types.Address) (*ktypes.KarmaSources, error) {
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
	switch userTarget.Target {
	case ktypes.KarmaSourceTarget_DEPLOY:
		return &ktypes.KarmaTotal{Count: userState.DeployKarmaTotal}, nil
	case ktypes.KarmaSourceTarget_CALL:
		return &ktypes.KarmaTotal{Count: userState.CallKarmaTotal}, nil
	case ktypes.KarmaSourceTarget_ALL:
		return &ktypes.KarmaTotal{Count: userState.DeployKarmaTotal + userState.CallKarmaTotal}, nil
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
	}

	ctx.GrantPermissionTo(newOracleAddr, ChangeOraclePermission, oracleRole)
	ctx.GrantPermissionTo(newOracleAddr, ChangeUserSourcesPermission, oracleRole)
	ctx.GrantPermissionTo(newOracleAddr, ResetSourcesPermission, oracleRole)
	if err := ctx.Set(OracleKey, pbOracle); err != nil {
		return errors.Wrap(err, "setting new oracle")
	}
	return nil
}

func CalculateTotalKarma(karmaSources ktypes.KarmaSources, karmaStates ktypes.KarmaState) (int64, int64) {
	var deployKarma, callKarma int64
	for _, c := range karmaSources.Sources {
		for _, s := range karmaStates.SourceStates {
			if c.Name == s.Name && (c.Target == ktypes.KarmaSourceTarget_DEPLOY || c.Target == ktypes.KarmaSourceTarget_ALL) {
				deployKarma += c.Reward * s.Count
			}
			if c.Name == s.Name && (c.Target == ktypes.KarmaSourceTarget_CALL || c.Target == ktypes.KarmaSourceTarget_ALL) {
				callKarma += c.Reward * s.Count
			}
		}
	}
	return deployKarma, callKarma
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

func (k *Karma) updateUserKarmaState(ctx contract.Context, user *types.Address) error {
	var karmaSources ktypes.KarmaSources
	if err := ctx.Get(SourcesKey, &karmaSources); err != nil {
		return err
	}

	userState, err := k.GetUserState(ctx, user)
	if err != nil {
		return err
	}
	userState.DeployKarmaTotal, userState.CallKarmaTotal = CalculateTotalKarma(karmaSources, *userState)
	if err := ctx.Set(UserStateKey(user), userState); err != nil {
		return errors.Wrap(err, "setting user state karma")
	}
	return nil
}

func modifyCountForUser(ctx contract.Context, user *types.Address, sourceName string, amount int64) (int64, error) {
	stateKey := UserStateKey(user)

	var userSourceCounts ktypes.KarmaState
	// If user source counts not found, We want to create a new source count
	if err := ctx.Get(stateKey, &userSourceCounts); err != nil && err != contract.ErrNotFound {
		return 0, errors.Wrapf(err, "source counts for user %s", user.String())
	}

	for i, source := range userSourceCounts.SourceStates {
		if source.Name == sourceName {
			if 0 > userSourceCounts.SourceStates[i].Count+amount {
				return 0, errors.Errorf("not enough karma in source %s. found %v, modifying by %v", sourceName, userSourceCounts.SourceStates[i].Count, amount)
			}
			userSourceCounts.SourceStates[i].Count += amount
			if err := ctx.Set(UserStateKey(user), &userSourceCounts); err != nil {
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
		Name:  sourceName,
		Count: amount,
	})
	if err := ctx.Set(UserStateKey(user), &userSourceCounts); err != nil {
		return 0, errors.Wrapf(err, "setting user source counts for %s", user.String())
	}
	return amount, nil
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
