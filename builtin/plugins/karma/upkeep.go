package karma

import (
	"math/big"
	"sort"

	"github.com/golang/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/common"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
)

const (
	DefaultUpkeepCost   = 1
	DefaultUpkeepPeriod = 3600
)

var (
	upkeepStateKey    = []byte("upkeepState")
	UpkeepKey         = []byte("karma:upkeep:params:key")
	RecordPrefix      = []byte("contract-record")
	ActivePrefix      = []byte("active")
	NextContractIdKey = []byte("next-contract-id")

	defaultUpkeep = &ktypes.KarmaUpkeepParams{
		Cost:   DefaultUpkeepCost,
		Period: DefaultUpkeepPeriod,
	}
)

func ContractRecordKey(contractAddr loom.Address) []byte {
	return util.PrefixKey(RecordPrefix, contractAddr.Bytes())
}

func ContractActiveKey(owner loom.Address, contractId uint64) ([]byte, error) {
	key, err := proto.Marshal(&ktypes.KarmaContractId{ContractId: contractId})
	if err != nil {
		return nil, errors.Wrapf(err, "marshal contract id %v", contractId)
	}
	return util.PrefixKey(ActivePrefix, owner.Bytes(), key), nil
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

// TODO: only the contract owner or the oracle should be allowed to activate a contract
func (k *Karma) SetActive(ctx contract.Context, contract *types.Address) error {
	record, err := GetContractRecord(ctx, loom.UnmarshalAddressPB(contract))
	if err != nil {
		return err
	}
	if record.Owner == nil {
		return errors.New("owner not found")
	}

	activeContractKey, err := ContractActiveKey(loom.UnmarshalAddressPB(record.Owner), record.ContractId)
	if err != nil {
		return errors.Wrapf(err, "making contract id key from %v", record.ContractId)
	}
	if ctx.Has(activeContractKey) {
		return errors.Wrapf(err, "contract %v already active", loom.UnmarshalAddressPB(record.Address).String())
	}
	// TODO: Should check if the user has sufficient karma to cover the upkeep of all contracts plus
	//       this one they're trying to activate.
	if err := ctx.Set(activeContractKey, record.Address); err != nil {
		return err
	}

	ownerAddr := loom.UnmarshalAddressPB(record.Owner)
	if err := incrementOwnedContracts(ctx, loom.UnmarshalAddressPB(record.Owner), 1); err != nil {
		return errors.Wrapf(err, "increment owned contract count for %v", ownerAddr.String())
	}
	return nil
}

// TODO: only the contract owner or the oracle should be allowed to deactivate a contract
func (k *Karma) SetInactive(ctx contract.Context, contract *types.Address) error {
	// TODO: check for contract == nil
	record, err := GetContractRecord(ctx, loom.UnmarshalAddressPB(contract))
	if err != nil {
		return err
	}
	return DeactivateContract(ctx, record)
}

func DeactivateContract(ctx contract.Context, record *ktypes.KarmaContractRecord) error {
	if record.Owner == nil {
		return errors.New("owner not found")
	}
	ownerAddr := loom.UnmarshalAddressPB(record.Owner)
	activeContractKey, err := ContractActiveKey(ownerAddr, record.ContractId)
	if err != nil {
		return errors.Wrapf(err, "making contract id key from %v", record.ContractId)
	}
	if !ctx.Has(activeContractKey) {
		return errors.Wrapf(err, "contract %v is not active", loom.UnmarshalAddressPB(record.Address).String())
	}
	ctx.Delete(activeContractKey)
	if err := incrementOwnedContracts(ctx, ownerAddr, -1); err != nil {
		return errors.Wrapf(err, "failed to update active contract count for %v", ownerAddr.String())
	}
	return nil
}

func GetContractRecord(ctx contract.StaticContext, contractAddr loom.Address) (*ktypes.KarmaContractRecord, error) {
	var record ktypes.KarmaContractRecord
	if err := ctx.Get(ContractRecordKey(contractAddr), &record); err != nil {
		return nil, errors.Errorf("failed to load record for contract %s", contractAddr.String())
	}
	return &record, nil
}

func IsContractActive(ctx contract.StaticContext, contract loom.Address) (bool, error) {
	var record ktypes.KarmaContractRecord
	if err := ctx.Get(ContractRecordKey(contract), &record); err != nil {
		return false, errors.Errorf("record for contract %v not found", contract.String())
	}
	if record.Owner == nil {
		return false, errors.Errorf("owner not found for contract %v", contract.String())
	}
	activeContractKey, err := ContractActiveKey(loom.UnmarshalAddressPB(record.Owner), record.ContractId)
	if err != nil {
		return false, errors.Wrapf(err, "making contract id key from %v", record.ContractId)
	}
	return ctx.Has(activeContractKey), nil
}

func AddOwnedContract(ctx contract.Context, owner, contract loom.Address) error {
	var next ktypes.KarmaContractId
	if err := ctx.Get(NextContractIdKey, &next); err != nil {
		return errors.Wrap(err, "failed to load next contract ID")
	}

	record := &ktypes.KarmaContractRecord{
		Owner:      owner.MarshalPB(),
		Address:    contract.MarshalPB(),
		ContractId: next.ContractId,
	}

	activeKey, err := ContractActiveKey(owner, next.ContractId)
	if err != nil {
		return errors.Wrap(err, "failed to generate active contract key")
	}

	next.ContractId++

	if err := ctx.Set(NextContractIdKey, &next); err != nil {
		return errors.Wrap(err, "failed to save next contract ID")
	}

	// TODO: Not convinced maintaining the active contracts total improves performance enough to
	//       justify the added complexity.
	if err := incrementOwnedContracts(ctx, owner, 1); err != nil {
		return errors.Wrapf(err, "increment owned contract count for %v", owner.String())
	}
	if err := ctx.Set(ContractRecordKey(contract), record); err != nil {
		return errors.Wrap(err, "failed to save contract record")
	}
	if err := ctx.Set(activeKey, contract.MarshalPB()); err != nil {
		return errors.Wrap(err, "failed to mark contract as active")
	}
	return nil
}

func GetActiveContractRecords(ctx contract.StaticContext, owner loom.Address) ([]*ktypes.KarmaContractRecord, error) {
	var records []*ktypes.KarmaContractRecord
	activeRecords := ctx.Range(util.PrefixKey(ActivePrefix, owner.Bytes()))
	for _, kv := range activeRecords {
		var contractAddr types.Address
		if err := proto.Unmarshal(kv.Value, &contractAddr); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal address value for key %v", kv.Key)
		}
		record, err := GetContractRecord(ctx, loom.UnmarshalAddressPB(&contractAddr))
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func GetActiveUsers(ctx contract.StaticContext) (map[string]ktypes.KarmaState, error) {
	users := make(map[string]ktypes.KarmaState)
	activeUsers := ctx.Range([]byte(UserStateKeyPrefix))
	for _, kv := range activeUsers {
		var userState ktypes.KarmaState
		if err := proto.Unmarshal(kv.Value, &userState); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal user state for key %v", kv.Key)
		}
		if userState.NumOwnedContracts > 0 {
			var ownerPB types.Address
			if err := proto.Unmarshal(kv.Key, &ownerPB); err != nil {
				return nil, errors.Wrapf(err, "unmarshal owner %v", kv.Key)
			}
			owner := loom.UnmarshalAddressPB(&ownerPB)
			// TODO: Get rid of this address->string->address thing.
			users[owner.String()] = userState
		}
	}
	return users, nil
}

func incrementOwnedContracts(ctx contract.Context, owner loom.Address, amount int64) error {
	userState, err := GetUserState(ctx, owner)
	if err != nil {
		return errors.Wrapf(err, "failed to load karma state for user %s", owner.String())
	}

	userState.NumOwnedContracts += amount

	userStateKey, err := UserStateKey(owner.MarshalPB())
	if err != nil {
		return err
	}
	return ctx.Set(userStateKey, userState)
}

func GetUpkeepState(ctx contract.StaticContext) (*ktypes.UpkeepState, error) {
	var upkeepState ktypes.UpkeepState
	if err := ctx.Get(upkeepStateKey, &upkeepState); err != nil {
		if err != contract.ErrNotFound {
			return nil, errors.Wrap(err, "failed to load upkeep state")
		}
	}
	return &upkeepState, nil
}

func Upkeep(ctx contract.Context) error {
	// TODO: merge KarmaUpkeepParams into UpkeepState
	var upkeep ktypes.KarmaUpkeepParams
	if err := ctx.Get(UpkeepKey, &upkeep); err != nil {
		return errors.Wrap(err, "failed to load upkeep params")
	}

	// Ignore upkeep if parameters are not valid
	if upkeep.Cost == 0 || upkeep.Period == 0 {
		return nil
	}

	upkeepState, err := GetUpkeepState(ctx)
	if err != nil {
		return err
	}

	// First time upkeep, first block for new chain
	if upkeepState.LastUpkeepHeight == 0 {
		upkeepState.LastUpkeepHeight = uint64(ctx.Block().Height)
		if err := ctx.Set(upkeepStateKey, upkeepState); err != nil {
			return errors.Wrap(err, "failed to save upkeep state")
		}
		return nil
	}

	if ctx.Block().Height < int64(upkeepState.LastUpkeepHeight)+upkeep.Period {
		return nil
	}

	activeUsers, err := GetActiveUsers(ctx)
	if err != nil {
		return errors.Wrap(err, "getting users with active contracts")
	}

	var karmaSources ktypes.KarmaSources
	if err := ctx.Get(SourcesKey, &karmaSources); err != nil {
		return errors.Wrap(err, "failed to load allowed karma sources")
	}

	deployUpkeep(ctx, upkeep, activeUsers, karmaSources.Sources)

	upkeepState.LastUpkeepHeight = uint64(ctx.Block().Height)
	if err := ctx.Set(upkeepStateKey, upkeepState); err != nil {
		return errors.Wrap(err, "failed to save upkeep state")
	}
	return nil
}

func deployUpkeep(ctx contract.Context, params ktypes.KarmaUpkeepParams, activeUsers map[string]ktypes.KarmaState, karmaSources []*ktypes.KarmaSourceReward) {
	sourceMap := make(map[string]int)
	for i, source := range karmaSources {
		sourceMap[source.Name] = i
	}

	for userStr, userState := range activeUsers {
		user, err := loom.ParseAddress(userStr)
		if err != nil {
			log.Error("cannot parse user %v during karma upkeep. %v", userStr, err)
			continue
		}

		upkeepCost := loom.NewBigUIntFromInt(userState.NumOwnedContracts * params.Cost)
		paramCost := loom.NewBigUIntFromInt(params.Cost)
		userKarma := common.BigZero()
		for _, userSource := range userState.SourceStates {
			if karmaSources[sourceMap[userSource.Name]].Target == ktypes.KarmaSourceTarget_DEPLOY {
				userKarma.Add(userKarma, &userSource.Count.Value)
			}
		}

		if userKarma.Cmp(upkeepCost) >= 0 {
			payKarma(upkeepCost, &userState, karmaSources, sourceMap)
			userState.DeployKarmaTotal.Value.Sub(&userState.DeployKarmaTotal.Value, upkeepCost)
		} else {
			canAfford := common.BigZero()
			_, leftOver := canAfford.DivMod(userKarma.Int, paramCost.Int, paramCost.Int)
			numberToInactivate := userState.NumOwnedContracts - int64(canAfford.Int64())

			if err := setInactiveContractIdOrdered(ctx, user, uint64(numberToInactivate)); err != nil {
				log.Error("inactivating %v contracts owned by user %v during karma upkeep. %v", numberToInactivate, userStr, err)
				continue
			}

			payKarma(canAfford.Mul(canAfford, loom.NewBigUIntFromInt(params.Cost)), &userState, karmaSources, sourceMap)
			if leftOver == nil || leftOver.Cmp(big.NewInt(0)) == 0 {
				userState.DeployKarmaTotal = loom.BigZeroPB()
			} else {
				userState.DeployKarmaTotal.Value.Int = leftOver
			}
		}
		userStateKey, localErr := UserStateKey(user.MarshalPB())
		if localErr != nil {
			log.Error("cannot make db key for user %v's karma state, error %v", userStr, localErr)
			continue
		}

		ctx.Set(userStateKey, &userState)
	}
}

func payKarma(upkeepCost *common.BigUInt, userState *ktypes.KarmaState, karmaSources []*ktypes.KarmaSourceReward, sourceMap map[string]int) {
	coinIndex := -1
	for i, userSource := range userState.SourceStates {
		if userSource.Name == CoinDeployToken {
			coinIndex = i
		} else if karmaSources[sourceMap[userSource.Name]].Target == ktypes.KarmaSourceTarget_DEPLOY {
			if userSource.Count.Value.Cmp(upkeepCost) > 0 {
				userSource.Count.Value.Sub(&userSource.Count.Value, upkeepCost)
				upkeepCost = common.BigZero()
				break
			} else {
				upkeepCost.Sub(upkeepCost, &userSource.Count.Value)
				userSource.Count.Value.Int = common.BigZero().Int
			}
		}
	}
	// TODO: Instead of doing this hack to charge the coin source last, could keep sources ordered
	//       so the coin is always charged last.
	if -1 != upkeepCost.Cmp(common.BigZero()) {
		userState.SourceStates[coinIndex].Count.Value.Sub(&userState.SourceStates[coinIndex].Count.Value, upkeepCost)
	}
}

func setInactiveContractIdOrdered(ctx contract.Context, user loom.Address, numberToInactivate uint64) error {
	records, err := GetActiveContractRecords(ctx, user)
	if err != nil {
		return errors.Wrapf(err, "get user %v's active contracts", user)
	}

	if numberToInactivate > uint64(len(records)) {
		numberToInactivate = uint64(len(records))
	}

	// TODO: This sort would be unnecessary if the contract records are iterated in ID order,
	//       the IDs are supposed to be sequential... may need to test.
	sort.Slice(records, func(i, j int) bool {
		if records[i].ContractId == records[j].ContractId {
			return j < i
		}
		return records[i].ContractId < records[j].ContractId
	})
	return setInactive(ctx, records[:numberToInactivate])
}

func setInactive(ctx contract.Context, records []*ktypes.KarmaContractRecord) error {
	for _, record := range records {
		if err := DeactivateContract(ctx, record); err != nil {
			return err
		}
	}
	return nil
}
