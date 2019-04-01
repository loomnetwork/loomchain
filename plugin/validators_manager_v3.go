package plugin

import (
	"fmt"

	"github.com/loomnetwork/go-loom"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	types "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv3"
	abci "github.com/tendermint/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

// ValidatorsManager implements loomchain.ValidatorsManager interface
type ValidatorsManagerV3 struct {
	ctx contract.Context
}

func NewValidatorsManagerV3(pvm *PluginVM) (*ValidatorsManagerV3, error) {
	caller := loom.RootAddress(pvm.State.Block().ChainID)
	contractAddr, err := pvm.Registry.Resolve("dposV3")
	if err != nil {
		return nil, err
	}
	readOnly := false
	ctx := contract.WrapPluginContext(pvm.createContractContext(caller, contractAddr, readOnly))
	return &ValidatorsManagerV3{
		ctx: ctx,
	}, nil
}

func (m *ValidatorsManagerV3) SlashInactivity(validatorAddr []byte) error {
	return dposv3.SlashInactivity(m.ctx, validatorAddr)
}

func (m *ValidatorsManagerV3) SlashDoubleSign(validatorAddr []byte) error {
	return dposv3.SlashDoubleSign(m.ctx, validatorAddr)
}

func (m *ValidatorsManagerV3) Elect() error {
	return dposv3.Elect(m.ctx)
}

func (m *ValidatorsManagerV3) ValidatorList() ([]*types.Validator, error) {
	return dposv3.ValidatorList(m.ctx)
}

func (m *ValidatorsManagerV3) BeginBlock(req abci.RequestBeginBlock, currentHeight int64) error {
	// Check if the function has been called with NoopValidatorsManager
	if m == nil {
		return nil
	}

	// A VoteInfo struct is created for every active validator. If
	// SignedLastBlock is not true for any of the validators, slash them for
	// inactivity. TODO limit slashes to once per election cycle
	for _, voteInfo := range req.LastCommitInfo.GetVotes() {
		if !voteInfo.SignedLastBlock {
			m.ctx.Logger().Info("DPOS BeginBlock", "DowntimeEvidence", fmt.Sprintf("%v+", voteInfo), "validatorAddress", voteInfo.Validator.Address)
			// err := m.SlashInactivity(voteInfo.Validator.Address)
			// if err != nil {
			// 	return err
			// }
		}
	}

	for _, evidence := range req.ByzantineValidators {
		// DuplicateVoteEvidence is the only type of evidence currently
		// implemented in tendermint but we don't get access to this via the
		// ABCI. Instead, we're just given a validator address and block height.
		// The conflicting vote data is kept within the consensus engine itself.
		m.ctx.Logger().Info("DPOS BeginBlock", "ByzantineEvidence", fmt.Sprintf("%v+", evidence))

		// TODO what prevents someone from resubmitting evidence?
		// evidence.ValidateBasic() seems to already be called by Tendermint,
		// I think it takes care of catching duplicates as well...
		if evidence.Height > (currentHeight - 100) {
			m.ctx.Logger().Info("DPOS BeginBlock Byzantine Slashing", "FreshEvidenceHeight", evidence.Height, "CurrentHeight", currentHeight)
			//err := m.SlashDoubleSign(evidence.Validator.Address)
			//if err != nil {
			//	return err
			//}
		}
	}

	return nil
}

func (m *ValidatorsManagerV3) EndBlock(req abci.RequestEndBlock) ([]abci.ValidatorUpdate, error) {
	// Check if the function has been called with NoopValidatorsManager
	if m == nil {
		return nil, nil
	}

	oldValidatorList, err := m.ValidatorList()
	if err != nil {
		return nil, err
	}

	m.ctx.Logger().Debug("DPOSv3 EndBlock", "OldValidatorsList", fmt.Sprintf("%v+", oldValidatorList))

	err = m.Elect()
	if err != nil {
		return nil, err
	}

	validatorList, err := m.ValidatorList()
	if err != nil {
		return nil, err
	}

	m.ctx.Logger().Debug("DPOSv3 EndBlock", "NewValidatorsList", fmt.Sprint("%v+", validatorList))

	var validators []abci.ValidatorUpdate

	// Clearing current validators by passing in list of zero-power update to
	// tendermint.
	removedValidators := dposv3.MissingValidators(oldValidatorList, validatorList)
	for _, validator := range removedValidators {
		validators = append(validators, abci.ValidatorUpdate{
			PubKey: abci.PubKey{
				Data: validator.PubKey,
				Type: tmtypes.ABCIPubKeyTypeEd25519,
			},
			Power: 0,
		})
	}

	// After the list of zero-power updates are processed by tendermint, the
	// rest of the validators updates will set the tendermint validator set to
	// be exactly the contents of the dpos validators list
	for _, validator := range validatorList {
		validators = append(validators, abci.ValidatorUpdate{
			PubKey: abci.PubKey{
				Data: validator.PubKey,
				Type: tmtypes.ABCIPubKeyTypeEd25519,
			},
			Power: validator.Power,
		})
	}

	return validators, nil
}
