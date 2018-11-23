package karma

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	maxLogDbSize = 5
	maxLogSources = 3
	maxLogUsers = 5
)

var (
	dummyKarma int64
)

type testFunc func (b *testing.B, state loomchain.State)

func BenchmarkKarma(b *testing.B) {
	benchmarkKarmaFunc(b, "calculateKarma", calculateKarma)
}

func benchmarkKarmaFunc(b *testing.B, name string, fn testFunc) {

	for logDbSize := 1; logDbSize < maxLogDbSize; logDbSize++ {
		state := mockState(logDbSize)
		for logSources := 1; logSources < maxLogSources; logSources++ {
			var sources karma.KarmaSources
			state, sources = mockSources(b, state, logSources)
			for logUsers := 1; logUsers < maxLogUsers; logUsers++ {
				state := mockUsers(b, state, sources, logUsers)
				b.Run(name + fmt.Sprintf(" stateSize %v, sources %v, users %v",
						int(math.Pow(10, float64(logDbSize))),
						int(math.Pow(10, float64(logSources))),
						int(math.Pow(10, float64(logUsers))),
					),
					func(b *testing.B) {
						fn(b, state)
					},
				)
			}
		}
	}
}
func calculateKarma(b *testing.B, state loomchain.State) {
	const user = 1

	var karmaSources karma.KarmaSources
	protoSources := state.Get(SourcesKey)
	require.NoError(b, proto.Unmarshal(protoSources, &karmaSources))

	var karmaStates karma.KarmaState
	protoUserState := state.Get(userKey(user))
	require.NoError(b, proto.Unmarshal(protoUserState, &karmaStates))

	var karmaValue = int64(0)
	for _, c := range karmaSources.Sources {
		for _, s := range karmaStates.SourceStates {
			if c.Name == s.Name && c.Target == karma.SourceTarget_DEPLOY {
				karmaValue += c.Reward * s.Count
			}
		}
	}
	dummyKarma = karmaValue
}

func mockUsers(b *testing.B, state loomchain.State, sources karma.KarmaSources, logUsers int) loomchain.State {
	users := int(math.Pow(10, float64(logUsers)))
	var userState karma.KarmaState
	for _, source := range sources.Sources {
		userState.SourceStates = append(userState.SourceStates, &karma.KarmaSource{
			Name: source.Name,
			Count: 5,
		})
	}
	protoUserState, err := proto.Marshal(&userState)
	require.NoError(b, err)

	for i:=0 ; i<users; i++ {
		state.Set(userKey(i), protoUserState)
	}
	return state
}

func userKey(user int) []byte {
	return []byte("user" + strconv.FormatInt(int64(user), 10))
}

func mockState(logSize int) loomchain.State {
	header := abci.Header{}
	state := loomchain.NewStoreState(context.Background(), store.NewMemStore(), header, nil)
	entries := int(math.Pow(10, float64(logSize)))
	for i:=0; i< entries; i++{
		strI := strconv.FormatInt(int64(i), 10)
		state.Set([]byte("user" + strI), []byte(strI))
	}
	return state
}

func mockSources(b *testing.B, state loomchain.State, logSize int) (loomchain.State, karma.KarmaSources) {
	numStates := int(math.Pow(10, float64(logSize)))
	var sources karma.KarmaSources
	for i :=0; i< numStates; i++ {
		sources.Sources = append(sources.Sources, &karma.KarmaSourceReward{
			Name: strconv.FormatInt(int64(i), 10) + "call",
			Reward: 1,
			Target: karma.SourceTarget_CALL,
		})
		sources.Sources = append(sources.Sources, &karma.KarmaSourceReward{
			Name: strconv.FormatInt(int64(i), 10) + "deploy",
			Reward: 1,
			Target: karma.SourceTarget_DEPLOY,
		})
	}

	protoSource, err := proto.Marshal(&sources)
	require.NoError(b, err)
	state.Set(SourcesKey, protoSource)

	return state, sources
}