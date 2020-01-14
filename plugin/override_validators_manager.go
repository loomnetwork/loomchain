package plugin

import (
	"encoding/hex"

	"github.com/pkg/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

// OverrideValidatorsManager implements loomchain.ValidatorsManager interface.
// This manager overrides the current validator set at a certain height (specified via loom.yml).
type OverrideValidatorsManager struct {
	height     int64
	validators []OverrideValidator
}

type OverrideValidator struct {
	// Hex-encoded public key identifying the validator whose power should be overriden.
	// Would've made more sense to encode the public key as base64 but YAML handling of base64 is
	// hilariously WTF so the keys get mangled on their way to/from loom.yml, so best to stick to
	// plain old hex-encoding.
	PubKey string
	// The power specified here will override whatever power the validator has at the height this
	// override is applied. If the override power is zero Tendermint will remove the validator from
	// the active validator set.
	Power int64
}

func NewOverrideValidatorsManager(height int64, validators []OverrideValidator) (*OverrideValidatorsManager, error) {
	return &OverrideValidatorsManager{
		height:     height,
		validators: validators,
	}, nil
}

func (m *OverrideValidatorsManager) BeginBlock(_ abci.RequestBeginBlock, _ int64) error {
	return nil
}

func (m *OverrideValidatorsManager) EndBlock(req abci.RequestEndBlock) ([]abci.ValidatorUpdate, error) {
	if req.Height != m.height {
		return nil, nil
	}

	var validators []abci.ValidatorUpdate

	for _, validator := range m.validators {
		pubKey, err := hex.DecodeString(validator.PubKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode validator pubkey")
		}
		validators = append(validators, abci.ValidatorUpdate{
			PubKey: abci.PubKey{
				Data: pubKey,
				Type: tmtypes.ABCIPubKeyTypeEd25519,
			},
			Power: validator.Power,
		})
	}

	return validators, nil
}
