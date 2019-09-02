package fnConsensus

import dbm "github.com/tendermint/tendermint/libs/db"

const reactorStateKey = "fnConsensusReactor:state"

func loadReactorState(db dbm.DB) (*ReactorState, error) {
	rectorStateBytes := db.Get([]byte(reactorStateKey))
	if rectorStateBytes == nil {
		return NewReactorState(), nil
	}

	persistedRectorState := &ReactorState{}
	if err := persistedRectorState.Unmarshal(rectorStateBytes); err != nil {
		return nil, err
	}
	persistedRectorState.Messages = make(map[string]Message)
	return persistedRectorState, nil
}

func saveReactorState(db dbm.DB, reactorState *ReactorState, sync bool) error {
	marshalledBytes, err := reactorState.Marshal()
	if err != nil {
		return err
	}

	if sync {
		db.SetSync([]byte(reactorStateKey), marshalledBytes)
	} else {
		db.Set([]byte(reactorStateKey), marshalledBytes)
	}

	return nil
}
