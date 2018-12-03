package karma

import (
	"encoding/binary"
	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain/log"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/loomnetwork/loomchain/registry/factory"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
)

var (
	lastKarmaUpkeepKey = []byte("upkeep:karma")
)

func NewKarmaHandler(regVer registry.RegistryVersion, karmaEnabled bool) loomchain.KarmaHandler {
	if  regVer == registry.RegistryV2 && karmaEnabled {
		createRegistry, err := factory.NewRegistryFactory(registry.RegistryV2)
		if err != nil {
			panic("registry.RegistryV2 does not return registry factory " + err.Error())
		}
		return karmaRegistryV2Handler{
			createRegistry,
		}
	}
	return emptyHandler{}
}

type emptyHandler struct {
}

func (kh emptyHandler) Upkeep(state loomchain.State) error {
	return nil
}

type karmaRegistryV2Handler struct {
	registryFactroy  factory.RegistryFactoryFunc
}

func (kh karmaRegistryV2Handler) Upkeep(state loomchain.State) error {
	reg := kh.registryFactroy(state)
	karmaContractAddress, err := reg.Resolve("karma")
	karmaState := loomchain.StateWithPrefix(loom.DataPrefix(karmaContractAddress), state)

	var upkeep ktypes.KarmaUpkeepParmas
	if err = proto.Unmarshal(karmaState.Get(karma.UpkeepKey), &upkeep); err != nil {
		return errors.Wrap(err, "unmarshal upkeep")
	}

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

	contractRecords, err := reg.GetRecords(true)
	if err != nil {
		return errors.Wrap(err, "getting active records")
	}

	deployUpkeep(reg, karmaState, upkeep, contractRecords)
	return nil
}

func deployUpkeep(reg registry.Registry, state loomchain.State, params ktypes.KarmaUpkeepParmas, contractRecords []*registry.Record)  {
	for _, record := range contractRecords {
		userStateKey := karma.GetUserStateKey(record.Owner)
		if !state.Has(userStateKey) {
			log.Error("cannot find state for user %s: %v", record.Owner.String())
			if localErr := reg.SetInactive(loom.UnmarshalAddressPB(record.Address)); localErr != nil {
				log.Error("cannot set contact %v inactive: %v", loom.UnmarshalAddressPB(record.Address).String(), localErr)
			}
			continue
		}

		data := state.Get(userStateKey)
		var userState ktypes.KarmaState
		if localErr := proto.Unmarshal(data, &userState); localErr != nil {
			log.Error("cannot unmarshal state for user %s: %v", record.Owner.String(), localErr)
			if localErr := reg.SetInactive(loom.UnmarshalAddressPB(record.Address)); localErr != nil {
				log.Error("cannot set contact %v inactive: %v", loom.UnmarshalAddressPB(record.Address).String(), localErr)
			}
			continue
		}

		var index int
		var userSource *ktypes.KarmaSource
		for i, source := range userState.SourceStates {
			if source.Name == params.Source {
				index = i
				userSource = source
				break
			}
		}

		if  userSource == nil || params.Cost > userSource.Count {
			if localErr := reg.SetInactive(loom.UnmarshalAddressPB(record.Address)); localErr != nil {
				log.Error("cannot set contact %v inactive: %v", loom.UnmarshalAddressPB(record.Address).String(), localErr)
			}
		} else {
			userSource.Count -= params.Cost
			userState.SourceStates[index] = userSource
			protoState, localErr := proto.Marshal(&userState)
			if localErr != nil {
				log.Error("cannot marshal user %v  inactive: %v", loom.UnmarshalAddressPB(record.Address).String(), localErr)
				continue
			}
			state.Set(userStateKey, protoState)
		}
	}
}