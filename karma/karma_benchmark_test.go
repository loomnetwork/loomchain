package karma

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/loomnetwork/loomchain/registry/factory"
	"github.com/stretchr/testify/require"
)

const (
	maxLogUsers          = 4
	maxLogContracts      = 5
	pcentDeactivateTicks = 3
)

type benchmarkFunc func(state loomchain.State) error

func BenchmarkKarma(b *testing.B) {
	kh2 := NewKarmaHandler(factory.RegistryV2, true)
	benchmarkKarmaFunc(b, "Upkeep, registry version 2", kh2.Upkeep)
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

	for pctTick := 0; pctTick < pcentDeactivateTicks; pctTick++ {
		pctHaveKarma := int(float64(pctTick) * float64(100/pcentDeactivateTicks))
		for logUsers := 0; logUsers < maxLogUsers; logUsers++ {
			for logContracts := 0; logContracts < maxLogContracts; logContracts++ {
				dbName := "dbs/mockDB" + "-c" + strconv.Itoa(logContracts) + "-u" + strconv.Itoa(logUsers) + "-t" + strconv.Itoa(pctTick)
				state, reg, _ := karma.MockStateWithKarmaAndCoinB(b, &karmaInit, nil, dbName)
				karmaAddr, err := reg.Resolve("karma")
				require.NoError(b, err)
				karmaState := loomchain.StateWithPrefix(loom.DataPrefix(karmaAddr), state)

				addMockUsersWithContracts(b, karmaState, reg, logUsers, pctHaveKarma, logContracts)
				state.Set(lastKarmaUpkeepKey, UintToBytesBigEndian(uint64(0)))

				b.Run(name+fmt.Sprintf(" users  %v, contracts per user %v, percentage users have karma %v",
					int(math.Pow(10, float64(logUsers))),
					int(math.Pow(10, float64(logContracts))),
					pctHaveKarma,
				),
					func(b *testing.B) {
						for i := 0; i < b.N; i++ {
							require.NoError(b, fn(state))
						}
					},
				)

			}
		}
	}
}

func addMockUsersWithContracts(b *testing.B, karmaState loomchain.State, reg registry.Registry, logUsers int, pctUsersHaveKarma int, logContracts int) {
	users := uint64(math.Pow(10, float64(logUsers)))
	usersWith := uint64(float64(users) * float64(pctUsersHaveKarma) / 100)
	numContracts := uint64(math.Pow(10, float64(logContracts)))

	userHaveState := ktypes.KarmaState{
		SourceStates: []*ktypes.KarmaSource{{
			Name:  karma.CoinDeployToken,
			Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(99999999)},
		}},
	}
	protoHaveKarmaState, err := proto.Marshal(&userHaveState)
	require.NoError(b, err)

	userHaveNotState := ktypes.KarmaState{
		SourceStates: []*ktypes.KarmaSource{{
			Name:  karma.CoinDeployToken,
			Count: loom.BigZeroPB(),
		}},
	}
	protoHaveNotKarmaState, err := proto.Marshal(&userHaveNotState)
	require.NoError(b, err)

	for i := uint64(0); i < users; i++ {
		userAddr := userAddr(i)
		if i < usersWith {
			karmaState.Set(karma.UserStateKey(userAddr.MarshalPB()), protoHaveKarmaState)
		} else {
			karmaState.Set(karma.UserStateKey(userAddr.MarshalPB()), protoHaveNotKarmaState)
		}

		for c := uint64(0); c < numContracts; c++ {
			MockDeployEvmContract(b, karmaState, userAddr, c, reg)
		}
	}
}

func userAddr(user uint64) loom.Address {
	tail := strconv.FormatUint(user, 10)
	tail += strings.Repeat("0", 20-len(tail))
	return loom.MustParseAddress("chain:0x" + hex.EncodeToString([]byte(tail)))
}

func MockDeployEvmContract(b *testing.B, karmaState loomchain.State, owner loom.Address, nonce uint64, reg registry.Registry) loom.Address {
	contractAddr := plugin.CreateAddress(owner, nonce)
	err := reg.Register("", contractAddr, owner)
	require.NoError(b, err)
	require.NoError(b, karma.AddOwnedContract(karmaState, owner, contractAddr))

	return contractAddr
}
