package karma

import (
	"github.com/golang/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
	"github.com/pkg/errors"
)

const (
	DefaultUpkeepCost   = 1
	DefaultUpkeepPeriod = 3600
)

var (
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

func (k *Karma) SetActive(ctx contract.Context, contract *types.Address) error {
	var record ktypes.KarmaContractRecord
	loomAddr := loom.UnmarshalAddressPB(contract)
	if err := ctx.Get(ContractRecordKey(loomAddr), &record); err != nil {
		return errors.Errorf("record for contract %v not found", loomAddr.String())
	}
	key, err := ContractActiveKey(loom.UnmarshalAddressPB(record.Owner), record.ContractId)
	if err != nil {
		return errors.Wrapf(err, "making contract id key from %v", record.ContractId)
	}
	if ctx.Has(key) {
		return errors.Wrapf(err, "contract %v already active", loomAddr.String())
	}
	if err := ctx.Set(key, contract); err != nil {
		return errors.Wrapf(err, "setting contract %v as active using key %v", contract, key)
	}
	return nil
}

func (k *Karma) SetInactive(ctx contract.Context, contract *types.Address) error {
	var record ktypes.KarmaContractRecord
	loomAddr := loom.UnmarshalAddressPB(contract)
	if err := ctx.Get(ContractRecordKey(loomAddr), &record); err != nil {
		return errors.Errorf("record for contract %v not found", loomAddr.String())
	}
	key, err := ContractActiveKey(loom.UnmarshalAddressPB(record.Owner), record.ContractId)
	if err != nil {
		return errors.Wrapf(err, "making contract id key from %v", record.ContractId)
	}
	if !ctx.Has(key) {
		return errors.Wrapf(err, "contract %v not active", loomAddr.String())
	}
	ctx.Delete(key)
	return nil
}

func (k *Karma) IsActive(ctx contract.StaticContext, contract *types.Address) (bool, error) {
	var record ktypes.KarmaContractRecord
	loomAddr := loom.UnmarshalAddressPB(contract)
	if err := ctx.Get(ContractRecordKey(loomAddr), &record); err != nil {
		return false, errors.Errorf("record for contract %v not found", loomAddr.String())
	}
	key, err := ContractActiveKey(loom.UnmarshalAddressPB(record.Owner), record.ContractId)
	if err != nil {
		return false, errors.Wrapf(err, "making contract id key from %v", record.ContractId)
	}
	return ctx.Has(key), nil
}

func IsActive(karmaState loomchain.State, contract loom.Address) (bool, error) {
	recordByes := karmaState.Get(ContractRecordKey(contract))
	var record ktypes.KarmaContractRecord
	if err := proto.Unmarshal(recordByes, &record); err != nil {
		return false, errors.Wrapf(err, "unmarshal record of contract %v", contract.String())
	}

	key, err := ContractActiveKey(loom.UnmarshalAddressPB(record.Owner), record.ContractId)
	if err != nil {
		return false, errors.Wrapf(err, "making contract id key from %v", record.ContractId)
	}
	return karmaState.Has(key), nil
}

func AddOwnedContract(karmastate loomchain.State, owner loom.Address, contract loom.Address) error {
	nextBytes := karmastate.Get(NextContractIdKey)
	var next ktypes.KarmaContractId
	if err := proto.Unmarshal(nextBytes, &next); err != nil {
		return errors.Wrapf(err, "unmarshal next karma contract id %v", karmastate.Get(UpkeepKey))
	}

	record, err := proto.Marshal(&ktypes.KarmaContractRecord{
		Owner:      owner.MarshalPB(),
		Address:    contract.MarshalPB(),
		ContractId: next.ContractId,
	})
	if err != nil {
		return errors.Wrapf(err, "marshal record %v", record)
	}
	activeKey, err := ContractActiveKey(owner, next.ContractId)
	next.ContractId++
	nextNextBytes, err := proto.Marshal(&next)
	if err != nil {
		return errors.Wrapf(err, "marshal next contract id %v", next)
	}
	karmastate.Set(NextContractIdKey, nextNextBytes)

	contractBytes, err := proto.Marshal(contract.MarshalPB())
	if err != nil {
		return errors.Wrapf(err, "marshal contract address %v", contract)
	}

	if err := incrementOwnedContracts(karmastate, owner, 1); err != nil {
		return errors.Wrapf(err, "increment owned contract count for %v", owner.String())
	}

	karmastate.Set(ContractRecordKey(contract), record)
	karmastate.Set(activeKey, contractBytes)

	return nil
}

func SetInactive(karmastate loomchain.State, record ktypes.KarmaContractRecord) error {
	key, err := ContractActiveKey(loom.UnmarshalAddressPB(record.Owner), record.ContractId)
	if err != nil {
		return errors.Wrapf(err, "making contract id %v into key", record.ContractId)
	}
	if !karmastate.Has(key) {
		return errors.Errorf("contract %v not found", record.Address.String())
	}
	karmastate.Delete(key)
	if err := incrementOwnedContracts(karmastate, loom.UnmarshalAddressPB(record.Owner), -1); err != nil {
		return errors.Wrapf(err, "increment owned contract count for %v", loom.UnmarshalAddressPB(record.Owner).String())
	}
	return nil
}

func GetActiveContractRecords(karmastate loomchain.State, owner loom.Address) ([]*ktypes.KarmaContractRecord, error) {
	var records []*ktypes.KarmaContractRecord
	prefix := util.PrefixKey(ActivePrefix, owner.Bytes())
	activeRecords := karmastate.Range(prefix)
	for _, kv := range activeRecords {
		var contractAddr types.Address
		if err := proto.Unmarshal(kv.Value, &contractAddr); err != nil {
			return nil, errors.Wrapf(err, "unmarshal contract %v", kv.Value)
		}

		var record ktypes.KarmaContractRecord
		if err := proto.Unmarshal(karmastate.Get(ContractRecordKey(loom.UnmarshalAddressPB(&contractAddr))), &record); err != nil {
			return nil, errors.Wrapf(err, "unmarshal contract record %v", contractAddr)
		}
		records = append(records, &record)
	}
	return records, nil
}

func GetActiveUsers(karmastate loomchain.State) ([]*ktypes.KarmaState, error) {
	var userStates []ktypes.KarmaState
	activeUsers := karmastate.Range([]byte(UserStateKeyPrefix))
	for _, kv := range activeUsers {
		var userState ktypes.KarmaState
		if err := proto.Unmarshal(kv.Value, &userState); err != nil {
			return nil, errors.Wrapf(err, "unmarshal user state key %v value %v", kv.Key, kv.Value)
		}
		userStates := append(userStates, userState)
	}
}

func incrementOwnedContracts(karmastate loomchain.State, owner loom.Address, amount int64) error {
	var userstate ktypes.KarmaState
	if err := proto.Unmarshal(karmastate.Get(UserStateKey(owner.MarshalPB())), &userstate); err != nil {
		return errors.Wrapf(err, "unmarshal user %v karma state from %v", owner.String(), karmastate.Get(UserStateKey(owner.MarshalPB())))
	}
	userstate.NumOwnedContracts = userstate.NumOwnedContracts + amount
	protoState, err := proto.Marshal(&userstate)
	if err != nil {
		return errors.Wrapf(err, "unmarshal user %v karma state from %v", owner.String(), karmastate.Get(UserStateKey(owner.MarshalPB())))
	}
	karmastate.Set(UserStateKey(owner.MarshalPB()), protoState)
	return nil
}
