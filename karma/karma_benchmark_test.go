//nolint
package karma

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/registry"
	appstate "github.com/loomnetwork/loomchain/state"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/db"
)

const (
	maxlogDbSize         = 1
	maxLogUsers          = 5
	maxLogContracts      = 4
	pcentDeactivateTicks = 1
)

type benchmarkFunc func(state appstate.State) error

func BenchmarkUpkeep(b *testing.B) {
	//kh2 := NewKarmaHandler(factory.RegistryV2, true, true)
	//benchmarkKarmaFunc(b, "Upkeep, registry version 2", kh2.Upkeep)
}

func benchmarkKarmaFunc(b *testing.B, name string, fn benchmarkFunc) {
	karmaInit := ktypes.KarmaInitRequest{
		Users:   []*ktypes.KarmaAddressSource{{User: user1, Sources: emptySourceStates}},
		Sources: awardSoures,
		Upkeep: &ktypes.KarmaUpkeepParams{
			Cost:   1,
			Period: 1,
		},
		Oracle: user1,
	}
	for logDbSize := 0; logDbSize < maxlogDbSize; logDbSize++ {
		require.True(b, pcentDeactivateTicks > 0)
		for pctTick := pcentDeactivateTicks; pctTick >= 0; pctTick-- {
			pctHaveKarma := int(float64(pctTick) * float64(100/pcentDeactivateTicks))
			for logUsers := float64(0); logUsers < maxLogUsers; logUsers++ {
				for logContracts := 0; logContracts < maxLogContracts; logContracts++ {
					dbName := "dbs/mockDB" + "-s" + strconv.Itoa(logDbSize) + "-c" + strconv.Itoa(logContracts) + "-u" + strconv.Itoa(int(logUsers*100)) + "-t" + strconv.Itoa(pctTick)
					state, reg, _ := karma.MockStateWithKarmaAndCoinB(b, &karmaInit, nil, dbName)
					karmaAddr, err := reg.Resolve("karma")
					require.NoError(b, err)
					karmaState := state.WithPrefix(loom.DataPrefix(karmaAddr))
					addMockUsersWithContracts(b, karmaState, reg, logUsers, pctHaveKarma, logContracts)
					state.Set(lastKarmaUpkeepKey, UintToBytesBigEndian(uint64(0)))
					state = addUsers(state, logDbSize)

					title := name + fmt.Sprintf("dbsize %v users  %v, contracts per user %v, percentage users have karma %v",
						int(math.Pow(10, float64(logDbSize))),
						int(math.Pow(10, float64(logUsers))),
						int(math.Pow(10, float64(logContracts))),
						pctHaveKarma,
					)

					b.Run(title, func(b *testing.B) {
						for i := 0; i < b.N; i++ {
							_ = fn(state)
						}
					})
				}
			}
		}
	}
}

func addMockUsersWithContracts(b *testing.B, karmaState appstate.State, reg registry.Registry, logUsers float64, pctUsersHaveKarma int, logContracts int) {
	users := uint64(math.Pow(10, float64(logUsers)))
	usersWith := uint64(float64(users) * float64(pctUsersHaveKarma) / 100)
	numContracts := uint64(math.Pow(10, float64(logContracts)))

	userHaveState := ktypes.KarmaState{
		SourceStates: []*ktypes.KarmaSource{{
			Name:  karma.CoinDeployToken,
			Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(99999999)},
		}},
		DeployKarmaTotal: &types.BigUInt{Value: *loom.NewBigUIntFromInt(99999999)},
	}
	protoHaveKarmaState, err := proto.Marshal(&userHaveState)
	require.NoError(b, err)

	userHaveNotState := ktypes.KarmaState{
		SourceStates: []*ktypes.KarmaSource{{
			Name:  karma.CoinDeployToken,
			Count: loom.BigZeroPB(),
		}},
		DeployKarmaTotal: loom.BigZeroPB(),
	}
	protoHaveNotKarmaState, err := proto.Marshal(&userHaveNotState)
	require.NoError(b, err)

	for i := uint64(0); i < users; i++ {
		userAddr := userAddr(i)
		key, err := karma.UserStateKey(userAddr.MarshalPB())
		require.NoError(b, err)
		if i < usersWith {
			karmaState.Set(key, protoHaveKarmaState)
		} else {
			karmaState.Set(key, protoHaveNotKarmaState)
		}

		for c := uint64(0); c < numContracts; c++ {
			MockDeployEvmContract(b, karmaState, userAddr, c, reg)
		}
	}
}

func userAddr(user uint64) loom.Address {
	tail := strconv.FormatUint(user, 10) + "END"
	tail += strings.Repeat("0", 20-len(tail))
	return loom.MustParseAddress("chain:0x" + hex.EncodeToString([]byte(tail)))
}

func MockDeployEvmContract(b *testing.B, karmaState appstate.State, owner loom.Address, nonce uint64, reg registry.Registry) loom.Address {
	contractAddr := plugin.CreateAddress(owner, nonce)
	err := reg.Register("", contractAddr, owner)
	require.NoError(b, err)
	//require.NoError(b, karma.AddOwnedContract(karmaState, owner, contractAddr))

	return contractAddr
}

func TestUpkeepBenchmark(t *testing.T) {
	/*
		t.Skip("benchmark test")
		kh2 := NewKarmaHandler(factory.RegistryV2, true, true)
		testUpkeepFunc(t, "Upkeep, registry version 2", kh2.Upkeep)
	*/
}

func testUpkeepFunc(t *testing.T, name string, fn benchmarkFunc) {
	karmaInit := ktypes.KarmaInitRequest{
		Users:   []*ktypes.KarmaAddressSource{{User: user1, Sources: emptySourceStates}},
		Sources: awardSoures,
		Upkeep: &ktypes.KarmaUpkeepParams{
			Cost:   1,
			Period: 1,
		},
		Oracle: user1,
	}
	fmt.Println(name)
	require.True(t, pcentDeactivateTicks > 0)
	for pctTick := pcentDeactivateTicks; pctTick >= 0; pctTick-- {
		pctHaveKarma := int(float64(pctTick) * float64(100/pcentDeactivateTicks))
		for logUsers := float64(0); logUsers < maxLogUsers; logUsers++ {
			for logContracts := 0; logContracts < maxLogContracts; logContracts++ {
				for logDbSize := 0; logDbSize < maxlogDbSize; logDbSize++ {
					dbName := "dbs/mockDB" + "-s" + strconv.Itoa(logDbSize) + "-c" + strconv.Itoa(logContracts) + "-u" + strconv.Itoa(int(logUsers*100)) + "-t" + strconv.Itoa(pctTick)
					appDb, err := db.NewGoLevelDB(dbName, ".")
					require.NoError(t, err)
					state, reg, _, err := karma.MockStateWithKarmaAndCoin(&karmaInit, nil, appDb)
					require.NoError(t, err)
					karmaAddr, err := reg.Resolve("karma")
					require.NoError(t, err)
					karmaState := state.WithPrefix(loom.DataPrefix(karmaAddr))
					addMockUsersWithContractsT(t, karmaState, reg, logUsers, pctHaveKarma, logContracts)
					state.Set(lastKarmaUpkeepKey, UintToBytesBigEndian(uint64(0)))
					state = addUsers(state, logDbSize)

					start := time.Now()
					_ = fn(state)
					now := time.Now()
					elapsed := now.Sub(start)
					fmt.Printf("deb size %v time taken users  %v, contracts per user %v, percentage users have karma %v is %v\n",
						int(math.Pow(10, float64(logDbSize))),
						int(math.Pow(10, float64(logUsers))),
						int(math.Pow(10, float64(logContracts))),
						pctHaveKarma,
						elapsed,
					)
				}
			}
		}
	}
}

func MockDeployEvmContractT(t *testing.T, karmaState appstate.State, owner loom.Address, nonce uint64, reg registry.Registry) loom.Address {
	contractAddr := plugin.CreateAddress(owner, nonce)
	err := reg.Register("", contractAddr, owner)
	require.NoError(t, err)
	//require.NoError(t, karma.AddOwnedContract(karmaState, owner, contractAddr))

	return contractAddr
}

func addMockUsersWithContractsT(t *testing.T, karmaState appstate.State, reg registry.Registry, logUsers float64, pctUsersHaveKarma int, logContracts int) {
	users := uint64(math.Pow(10, float64(logUsers)))
	usersWith := uint64(float64(users) * float64(pctUsersHaveKarma) / 100)
	numContracts := uint64(math.Pow(10, float64(logContracts)))

	userHaveState := ktypes.KarmaState{
		SourceStates: []*ktypes.KarmaSource{{
			Name:  karma.CoinDeployToken,
			Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(99999999)},
		}},
		DeployKarmaTotal: &types.BigUInt{Value: *loom.NewBigUIntFromInt(99999999)},
	}
	protoHaveKarmaState, err := proto.Marshal(&userHaveState)
	require.NoError(t, err)

	userHaveNotState := ktypes.KarmaState{
		SourceStates: []*ktypes.KarmaSource{{
			Name:  karma.CoinDeployToken,
			Count: loom.BigZeroPB(),
		}},
		DeployKarmaTotal: loom.BigZeroPB(),
	}
	protoHaveNotKarmaState, err := proto.Marshal(&userHaveNotState)
	require.NoError(t, err)

	for i := uint64(0); i < users; i++ {
		userAddr := userAddr(i)
		key, err := karma.UserStateKey(userAddr.MarshalPB())
		require.NoError(t, err)
		if i < usersWith {
			karmaState.Set(key, protoHaveKarmaState)
		} else {
			karmaState.Set(key, protoHaveNotKarmaState)
		}

		for c := uint64(0); c < numContracts; c++ {
			MockDeployEvmContractT(t, karmaState, userAddr, c, reg)
		}
	}
}

func addUsers(state appstate.State, logSize int) appstate.State {
	entries := uint64(math.Pow(10, float64(logSize)))
	for i := uint64(0); i < entries; i++ {
		strI := strconv.FormatUint(i, 10)
		state.Set([]byte("user"+strI), []byte(strI))
	}
	return state
}
