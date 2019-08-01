package karma

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	maxLogDbSize  = 3
	maxLogSources = 3
	maxLogUsers   = 5
)

var (
	dummyKarma int64
)

type testFunc func(state loomchain.State)

func TestKarma(t *testing.T) {
	t.Skip("use benchmark")
	testKarmaFunc(t, "calculateKarma", calculateKarma)
	fmt.Println()
	testKarmaFunc(t, "readKarma", readKarma)
	fmt.Println()
	testKarmaFunc(t, "updateKarma", updateKarma)
}

func testKarmaFunc(_ *testing.T, name string, fn testFunc) {
	for logDbSize := 0; logDbSize < maxLogDbSize; logDbSize++ {
		state := mockState(logDbSize)
		for logSources := 0; logSources < maxLogSources; logSources++ {
			var sources karma.KarmaSources
			state, sources = mockSources(state, logSources)
			for logUsers := 0; logUsers < maxLogUsers; logUsers++ {
				stateNew := mockUsers(state, sources, logUsers)
				start := time.Now()
				fn(stateNew)
				now := time.Now()
				elapsed := now.Sub(start)
				fmt.Printf(name+": Time taken for stateSize %v, sources %v, users %v is %v\n",
					int(math.Pow(10, float64(logDbSize))),
					int(math.Pow(10, float64(logSources))),
					int(math.Pow(10, float64(logUsers))),
					elapsed)
			}
		}
	}
}

func BenchmarkKarma(b *testing.B) {
	benchmarkKarmaFunc(b, "calculateKarma", calculateKarma)
	benchmarkKarmaFunc(b, "readKarma", readKarma)
	benchmarkKarmaFunc(b, "updateKarma", updateKarma)
}

func benchmarkKarmaFunc(b *testing.B, name string, fn testFunc) {
	for logDbSize := 0; logDbSize < maxLogDbSize; logDbSize++ {
		state := mockState(logDbSize)
		for logSources := 0; logSources < maxLogSources; logSources++ {
			var sources karma.KarmaSources
			state, sources = mockSources(state, logSources)
			for logUsers := 0; logUsers < maxLogUsers; logUsers++ {
				stateLoop := mockUsers(state, sources, logUsers)
				b.Run(name+fmt.Sprintf(" stateSize %v, sources %v, users %v",
					int(math.Pow(10, float64(logDbSize))),
					int(math.Pow(10, float64(logSources))),
					int(math.Pow(10, float64(logUsers))),
				),
					func(b *testing.B) {
						for i := 0; i < b.N; i++ {
							fn(stateLoop)
						}
					},
				)
			}
		}
	}
}

func calculateKarma(state loomchain.State) {
	const user = 0

	var karmaSources karma.KarmaSources
	protoSources := state.Get(SourcesKey)
	if err := proto.Unmarshal(protoSources, &karmaSources); err != nil {
		panic("unmarshal sources")
	}

	var karmaStates karma.KarmaState
	protoUserState := state.Get(userKey(user))
	if err := proto.Unmarshal(protoUserState, &karmaStates); err != nil {
		panic("unmarshal state")
	}
	var karmaValue = common.BigZero()
	for _, c := range karmaSources.Sources {
		for _, s := range karmaStates.SourceStates {
			if c.Name == s.Name && c.Target == karma.KarmaSourceTarget_DEPLOY {
				reward := loom.NewBigUIntFromInt(c.Reward)
				karmaValue.Add(karmaValue, reward.Mul(reward, &s.Count.Value))
			}
		}
	}
	dummyKarma = karmaValue.Int64()
}

func readKarma(state loomchain.State) {
	var err error
	const user = 0
	protoAmount := state.Get(userKarmaKey(user))
	dummyKarma, err = strconv.ParseInt(string(protoAmount), 10, 64)
	if err != nil {
		panic("pasring karma int64")
	}
}

func updateKarma(state loomchain.State) {
	userRange := state.Range([]byte("user."))
	for _, userKV := range userRange {
		var karmaStates karma.KarmaState
		if err := proto.Unmarshal(userKV.Value, &karmaStates); err != nil {
			panic("unmarshal state")
		}

		for index, userSource := range karmaStates.SourceStates {
			if userSource.Name == "0deploy" {
				//newCount := karmaStates.SourceStates[index].Count.Value
				karmaStates.SourceStates[index].Count.Value.Sub(
					&karmaStates.SourceStates[index].Count.Value,
					loom.NewBigUIntFromInt(1),
				)
				break
			}
		}
		protoUserState, err := proto.Marshal(&karmaStates)
		if err != nil {
			panic("cannot marshal user state")
		}
		state.Set(userKV.Key, protoUserState)
	}
}

func mockUsers(state loomchain.State, sources karma.KarmaSources, logUsers int) loomchain.State {
	users := uint64(math.Pow(10, float64(logUsers)))
	totalKarma := []byte(strconv.FormatInt(10, 10))
	var userState karma.KarmaState
	for _, source := range sources.Sources {
		userState.SourceStates = append(userState.SourceStates, &karma.KarmaSource{
			Name:  source.Name,
			Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(5)},
		})
	}
	protoUserState, err := proto.Marshal(&userState)
	if err != nil {
		panic("cannot marshal user state")
	}

	for i := uint64(0); i < users; i++ {
		state.Set(userKey(i), protoUserState)
		state.Set(userKarmaKey(i), totalKarma)
	}
	return state
}

func userKey(user uint64) []byte {
	return []byte("user." + strconv.FormatUint(user, 10))
}

func userKarmaKey(user uint64) []byte {
	return append([]byte("total-karma.user."), userKey(user)...)
}

func mockState(logSize int) loomchain.State {
	header := abci.Header{}
	state := loomchain.NewStoreState(context.Background(), store.NewMemStore(), header, nil, nil, nil)
	entries := uint64(math.Pow(10, float64(logSize)))
	for i := uint64(0); i < entries; i++ {
		strI := strconv.FormatUint(i, 10)
		state.Set([]byte("user"+strI), []byte(strI))
	}
	return state
}

func mockSources(state loomchain.State, logSize int) (loomchain.State, karma.KarmaSources) {
	numStates := uint64(math.Pow(10, float64(logSize)))
	var sources karma.KarmaSources
	for i := uint64(0); i < numStates; i++ {
		sources.Sources = append(sources.Sources, &karma.KarmaSourceReward{
			Name:   strconv.FormatUint(i, 10) + "call",
			Reward: 1,
			Target: karma.KarmaSourceTarget_CALL,
		})
		sources.Sources = append(sources.Sources, &karma.KarmaSourceReward{
			Name:   strconv.FormatUint(i, 10) + "deploy",
			Reward: 1,
			Target: karma.KarmaSourceTarget_DEPLOY,
		})
	}

	protoSource, err := proto.Marshal(&sources)
	if err != nil {
		panic("cannot marshal user state")
	}

	state.Set(SourcesKey, protoSource)

	return state, sources
}
