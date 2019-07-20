package fnConsensus

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/tendermint/tendermint/crypto"
)

type OverrideValidatorParsable struct {
	Address     string
	VotingPower int64
}

type OverrideValidator struct {
	Address     crypto.Address
	VotingPower int64
}

type ReactorConfigParsable struct {
	OverrideValidators     []*OverrideValidatorParsable
	FnVoteSigningThreshold SigningThreshold
}

func (r *ReactorConfigParsable) Parse() (*ReactorConfig, error) {
	reactorConfig := &ReactorConfig{}

	if r == nil {
		return nil, fmt.Errorf("fnConsensus reactor's parsable configuration cant be nil")
	}

	if r.FnVoteSigningThreshold != AllSigningThreshold && r.FnVoteSigningThreshold != Maj23SigningThreshold {
		return nil, fmt.Errorf("unknown signing threshold: %s specified", r.FnVoteSigningThreshold)
	}

	reactorConfig.FnVoteSigningThreshold = r.FnVoteSigningThreshold

	reactorConfig.OverrideValidators = make([]*OverrideValidator, len(r.OverrideValidators))

	for i, overrideValidator := range r.OverrideValidators {
		address, err := hex.DecodeString(strings.TrimPrefix(overrideValidator.Address, "0x"))
		if err != nil {
			return nil, fmt.Errorf("unable to parse override validator's address")
		}

		if overrideValidator.VotingPower <= 0 {
			return nil, fmt.Errorf("override validator's voting power need to be greater than zero")
		}

		reactorConfig.OverrideValidators[i] = &OverrideValidator{
			Address:     address,
			VotingPower: overrideValidator.VotingPower,
		}
	}

	return reactorConfig, nil
}

func DefaultReactorConfigParsable() *ReactorConfigParsable {
	return &ReactorConfigParsable{
		FnVoteSigningThreshold: Maj23SigningThreshold,
	}
}

type ReactorConfig struct {
	FnVoteSigningThreshold SigningThreshold
	OverrideValidators     []*OverrideValidator
}
