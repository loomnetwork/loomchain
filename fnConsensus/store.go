package fnConsensus

import dbm "github.com/tendermint/tendermint/libs/db"

const ReactorStateKey = "fnConsensusReactor:state"

func LoadReactorState(db dbm.DB) (*ReactorState, error) {
	rectorStateBytes := db.Get([]byte(ReactorStateKey))
	if rectorStateBytes == nil {
		return NewReactorState(), nil
	}

	persistedRectorState := &ReactorState{}
	if err := persistedRectorState.Unmarshal(rectorStateBytes); err != nil {
		return nil, err
	}
	return persistedRectorState, nil
}

func SaveReactorState(db dbm.DB, reactorState *ReactorState, sync bool) error {
	marshalledBytes, err := reactorState.Marshal()
	if err != nil {
		return err
	}

	if sync {
		db.SetSync([]byte(ReactorStateKey), marshalledBytes)
	} else {
		db.Set([]byte(ReactorStateKey), marshalledBytes)
	}

	return nil
}
