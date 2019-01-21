package karma

import (
	"encoding/binary"
	"sort"

	"github.com/pkg/errors"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/common"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/registry/factory"
)

var (
	lastKarmaUpkeepKey = []byte("upkeep:karma")
)

func NewKarmaHandler(regVer factory.RegistryVersion, karmaEnabled bool) loomchain.KarmaHandler {
	if  regVer == factory.RegistryV2 && karmaEnabled {
		createRegistry, err := factory.NewRegistryFactory(factory.RegistryV2)
		if err != nil {
			panic("registry.RegistryV2 does not return registry factory " + err.Error())
		}
		return karmaHandler{
			registryFactroy: createRegistry,
		}
	}
	return emptyHandler{}
}

type emptyHandler struct {
}

func (kh emptyHandler) Upkeep(state loomchain.State) error {
	return nil
}

type karmaHandler struct {
	registryFactroy  factory.RegistryFactoryFunc
}

func (kh karmaHandler) Upkeep(state loomchain.State) error {
	reg := kh.registryFactroy(state)
	karmaContractAddress, err := reg.Resolve("karma")
	karmaState := loomchain.StateWithPrefix(loom.DataPrefix(karmaContractAddress), state)

	var upkeep ktypes.KarmaUpkeepParams
	if err = proto.Unmarshal(karmaState.Get(karma.UpkeepKey), &upkeep); err != nil {
		return errors.Wrap(err, "unmarshal upkeep")
	}

	// Ignore upkeep if parameters are not valid
	if upkeep.Cost == 0 || upkeep.Period == 0 {
		return nil
	}

	// First time upkeep, first block for new chain
	if !state.Has(lastKarmaUpkeepKey) {
		sizeB := make([]byte, 8)
		binary.LittleEndian.PutUint64(sizeB, uint64(state.Block().Height))
		state.Set(lastKarmaUpkeepKey, sizeB)
		return nil
	}
	upkeepBytes := state.Get(lastKarmaUpkeepKey)
	lastUpkeep := binary.LittleEndian.Uint64(upkeepBytes)

	if state.Block().Height < int64(lastUpkeep) + upkeep.Period {
		return nil
	}

	//contractRecords, err := reg.GetRecords(true)
	contractRecords, err := karma.GetActiveContractRecords(karmaState)
	if err != nil {
		return errors.Wrap(err, "getting active records")
	}

	deployUpkeep(karmaState, upkeep, contractRecords)
	return nil
}

func deployUpkeep(karmaState loomchain.State, params ktypes.KarmaUpkeepParams, contractRecords []*ktypes.ContractRecord)  {
	activeRecords := make(map[string][]*ktypes.ContractRecord)
	for _, record := range contractRecords {
		index := loom.UnmarshalAddressPB(record.Owner).String()
		activeRecords[index] = append(activeRecords[index], record)
	}

	karmaSources := karmaState.Get(karma.SourcesKey)
	var sources ktypes.KarmaSources
	if err := proto.Unmarshal(karmaState.Get(karma.SourcesKey), &sources); err != nil {
		return //nil, errors.Wrap(err, "throttle: unmarshal karma sources")
	}

	//sourceMap := new(map[string]int)
	//for i, source := range  karmaSources {
	//	sourceMap[source.n]
	//}

	for user, records := range activeRecords {
		userStateKey := karma.UserStateKey(loom.MustParseAddress(user).MarshalPB())
		if !karmaState.Has(userStateKey) {
			log.Error("cannot find state for user %s: %v", user)
			setInactive(karmaState, contractRecords)
			continue
		}

		data := karmaState.Get(userStateKey)
		var userState ktypes.KarmaState
		if localErr := proto.Unmarshal(data, &userState); localErr != nil {
			log.Error("cannot unmarshal state for user %s: %v", user, localErr)
			setInactive(karmaState, contractRecords)
			continue
		}

		var index int
		var userSource *ktypes.KarmaSource
		coinKarma := common.BigZero()
		awardKarma := common.BigZero()
		for i, source := range userState.SourceStates {
			if source.Name == params.Source {
				index = i
				userSource = source
				break
			}
		}

		if userSource != nil && userSource.Count >= int64(len(records)) * int64(params.Cost) {
			userSource.Count -= params.Cost * int64(len(records))
			userState.DeployKarmaTotal -= params.Cost * int64(len(records))
		} else {
			setInactiveCreationBlockOrder(karmaState, contractRecords, len(records) - int(userSource.Count / params.Cost))
			userSource.Count = userSource.Count % params.Cost
			userState.DeployKarmaTotal = userSource.Count % params.Cost
		}
		userState.SourceStates[index] = userSource
		protoState, localErr := proto.Marshal(&userState)
		if localErr != nil {
			log.Error("cannot marshal user %v  error %v", user, localErr)
			continue
		}
		karmaState.Set(userStateKey, protoState)
	}
}

func setInactiveCreationBlockOrder(karmaState loomchain.State, records []*ktypes.ContractRecord, numberToInactivate int) {
	if numberToInactivate > len(records) {
		numberToInactivate = len(records)
	}
	sort.Slice(records, func(i, j int) bool {
		// records in order of addition to db. Use for records added in the same block.
		if records[i].Nonce == records[j].Nonce {
			return j<i
		}
		return records[i].Nonce < records[j].Nonce
	})
	setInactive(karmaState, records[len(records)-numberToInactivate:])
}

func setInactive(karmaState loomchain.State, records []*ktypes.ContractRecord) {
	for _, record := range records {
		if localErr := karma.SetInactive(karmaState, loom.UnmarshalAddressPB(record.Address)); localErr != nil {
			log.Error("cannot set contact %v inactive: %v", loom.UnmarshalAddressPB(record.Address).String(), localErr)
		}
	}
}
