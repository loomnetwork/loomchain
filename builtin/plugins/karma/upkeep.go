package karma

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/loomnetwork/go-loom/types"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
)

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
	addr := loom.UnmarshalAddressPB(contract)
	var record ktypes.ContractRecord
	if err := ctx.Get(ContractInactiveRecordKey(addr), &record); err != nil {
		return errors.Wrapf(err, "getting record for %s", addr.String())
	}
	ctx.Delete(ContractInactiveRecordKey(addr))
	if err := ctx.Set(ContractActiveRecordKey(addr), &record); err != nil {
		return errors.Wrapf(err, "setting record %v for %s", record, addr.String())
	}
	return nil
}

func (k *Karma) SetInactive(ctx contract.Context, contract *types.Address) error {
	addr := loom.UnmarshalAddressPB(contract)
	var record ktypes.ContractRecord
	if err := ctx.Get(ContractActiveRecordKey(addr), &record); err != nil {
		return errors.Wrapf(err, "getting record for %s", addr.String())
	}
	ctx.Delete(ContractActiveRecordKey(addr))
	if err := ctx.Set(ContractInactiveRecordKey(addr), &record); err != nil {
		return errors.Wrapf(err, "setting record %v for %s", record, addr.String())
	}
	return nil
}

func (k *Karma) IsActive(ctx contract.StaticContext, contract *types.Address) (bool, error) {
	return ctx.Has(ContractActiveRecordKey(loom.UnmarshalAddressPB(contract))), nil
}

func AddOwnedContract(state loomchain.State, owner loom.Address, contract loom.Address, block int64, nonce uint64) error {
	record, err := proto.Marshal(&ktypes.ContractRecord{
		Owner:         	owner.MarshalPB(),
		Address:       	contract.MarshalPB(),
		CreationBlock: 	block,
		Nonce: 			int64(nonce),
	})
	if err != nil {
		return errors.Wrapf(err, "marshal record %v", record)
	}
	state.Set(ContractActiveRecordKey(contract), record)
	return nil
}

func SetInactive(state loomchain.State, contract loom.Address) error {
	record := state.Get(ContractActiveRecordKey(contract))
	if len(record) == 0 {
		return errors.Errorf("contract not found %v", contract.String())
	}
	state.Delete(ContractActiveRecordKey(contract))
	state.Set(ContractInactiveRecordKey(contract), record)

	return nil
}


func GetActiveContractRecords(state loomchain.State) ([]*ktypes.ContractRecord, error) {
	var records []*ktypes.ContractRecord
	activeRecords := state.Range(ActivePrefix)
	for _, kv := range activeRecords {
		var record ktypes.ContractRecord
		if err := proto.Unmarshal(kv.Value, &record); err != nil {
			return nil, errors.Wrapf(err, "unmarshal record %v", kv.Value)
		}
		records = append(records, &record)
	}
	return records, nil
}