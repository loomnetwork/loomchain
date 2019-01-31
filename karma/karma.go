package karma

import (
	"encoding/binary"
	"math/big"
	"sort"

	"github.com/pkg/errors"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/common"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/registry/factory"
)

var (
	lastKarmaUpkeepKey = []byte("last:upkeep:karma")
)

func NewKarmaHandler(regVer factory.RegistryVersion, karmaEnabled bool) loomchain.KarmaHandler {
	if regVer == factory.RegistryV2 && karmaEnabled {
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
	registryFactroy factory.RegistryFactoryFunc
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
		state.Set(lastKarmaUpkeepKey, UintToBytesBigEndian(uint64(state.Block().Height)))
		return nil
	}
	upkeepBytes := state.Get(lastKarmaUpkeepKey)
	lastUpkeep := binary.BigEndian.Uint64(upkeepBytes)

	if state.Block().Height < int64(lastUpkeep)+upkeep.Period {
		return nil
	}

	//contractRecords, err := karma.GetActiveContractRecords(karmaState)
	//if err != nil {
	//	return errors.Wrap(err, "getting active records")
	//}

	activeUsers, err := karma.GetActiveUsers(karmaState)
	if err != nil {
		return errors.Wrap(err, "getting users with active contracts")
	}

	var karmaSources ktypes.KarmaSources
	if err := proto.Unmarshal(karmaState.Get(karma.SourcesKey), &karmaSources); err != nil {
		return errors.Wrapf(err, "unmarshal karma sources %v", karmaState.Get(karma.SourcesKey))
	}

	deployUpkeep(karmaState, upkeep, activeUsers, karmaSources.Sources)

	state.Set(lastKarmaUpkeepKey, UintToBytesBigEndian(uint64(state.Block().Height)))

	return nil
}

func deployUpkeep(karmaState loomchain.State, params ktypes.KarmaUpkeepParams, activeUsers map[string]ktypes.KarmaState, karmaSources []*ktypes.KarmaSourceReward) {
	//activeRecords := make(map[string][]*ktypes.KarmaContractRecord)
	//for _, record := range contractRecords {
	//	index := loom.UnmarshalAddressPB(record.Owner).String()
	//	activeRecords[index] = append(activeRecords[index], record)
	//}

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

		//userStateKey := karma.UserStateKey(loom.MustParseAddress(user).MarshalPB())
		//if !karmaState.Has(userStateKey) {
		//	log.Error("cannot find state for user %s: %v", user)
		//	setInactive(karmaState, contractRecords)
		//	continue
		//}

		//data := karmaState.Get(userStateKey)
		//var userState ktypes.KarmaState
		//if localErr := proto.Unmarshal(data, &userState); localErr != nil {
		//	log.Error("cannot unmarshal state for user %s: %v", user, localErr)
		//	setInactive(karmaState, contractRecords)
		//	continue
		//}

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

			if err := setInactiveContractIdOrdered(karmaState, user, uint64(numberToInactivate)); err != nil {
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
		protoState, localErr := proto.Marshal(&userState)
		if localErr != nil {
			log.Error("cannot marshal user %v's karma state, error %v", userStr, localErr)
			continue
		}
		userStateKey := karma.UserStateKey(user.MarshalPB())
		karmaState.Set(userStateKey, protoState)
	}
}

func payKarma(upkeepCost *common.BigUInt, userState *ktypes.KarmaState, karmaSources []*ktypes.KarmaSourceReward, sourceMap map[string]int) {
	coinIndex := -1
	for i, userSource := range userState.SourceStates {
		if userSource.Name == karma.CoinDeployToken {
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
	if -1 != upkeepCost.Cmp(common.BigZero()) {
		userState.SourceStates[coinIndex].Count.Value.Sub(&userState.SourceStates[coinIndex].Count.Value, upkeepCost)
	}
}

func setInactiveContractIdOrdered(karmaState loomchain.State, user loom.Address, numberToInactivate uint64) error {
	records, err := karma.GetActiveContractRecords(karmaState, user)
	if err != nil {
		return errors.Wrapf(err, "get user %v's active contracts", user)
	}

	if numberToInactivate > uint64(len(records)) {
		numberToInactivate = uint64(len(records))
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].ContractId == records[j].ContractId {
			return j < i
		}
		return records[i].ContractId < records[j].ContractId
	})
	setInactive(karmaState, records[:numberToInactivate])
	return nil
}

func setInactive(karmaState loomchain.State, records []*ktypes.KarmaContractRecord) {
	for _, record := range records {
		if localErr := karma.SetInactive(karmaState, *record); localErr != nil {
			log.Error("cannot set contact %v inactive: %v", loom.UnmarshalAddressPB(record.Address).String(), localErr)
		}
	}
}

func UintToBytesBigEndian(height uint64) []byte {
	heightB := make([]byte, 8)
	binary.BigEndian.PutUint64(heightB, height)
	return heightB
}
