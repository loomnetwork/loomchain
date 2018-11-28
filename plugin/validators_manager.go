package plugin

import (
	"github.com/loomnetwork/go-loom"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv2"
	abci "github.com/tendermint/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/types"

	"fmt"
)

// ValidatorsManager implements loomchain.ValidatorsManager interface
type ValidatorsManager struct {
	ctx contract.Context
}

func NewValidatorsManager(pvm *PluginVM) (*ValidatorsManager, error) {
	caller := loom.RootAddress(pvm.State.Block().ChainID)
	contractAddr, err := pvm.Registry.Resolve("dposV2")
	if err != nil {
		return nil, err
	}
	readOnly := false
	ctx := contract.WrapPluginContext(pvm.createContractContext(caller, contractAddr, readOnly))
	return &ValidatorsManager{
		ctx: ctx,
	}, nil
}

func NewNoopValidatorsManager() *ValidatorsManager {
	var manager *ValidatorsManager
	return manager
}

func (m *ValidatorsManager) SlashInactivity(validatorAddr []byte) error {
	return dposv2.SlashInactivity(m.ctx, validatorAddr)
}

func (m *ValidatorsManager) SlashDoubleSign(validatorAddr []byte) error {
	return dposv2.SlashDoubleSign(m.ctx, validatorAddr)
}

func (m *ValidatorsManager) Elect() error {
	return dposv2.Elect(m.ctx)
}

func (m *ValidatorsManager) ValidatorList() (*dposv2.ListValidatorsResponse, error) {
	return dposv2.ValidatorList(m.ctx)
}

func (m *ValidatorsManager) BeginBlock(req abci.RequestBeginBlock, currentHeight int64) error {
	// Check if the function has been called with NoopValidatorsManager
	if m == nil {
		return nil
	}

	// A VoteInfo struct is created for every active validator. If
	// SignedLastBlock is not true for any of the validators, slash them for
	// inactivity. TODO limit slashes to once per election cycle
	for _, voteInfo := range req.LastCommitInfo.GetVotes() {
		if !voteInfo.SignedLastBlock {
			err := m.SlashInactivity(voteInfo.Validator.Address)
			if err != nil {
				return err
			}
		}
	}

	for _, evidence := range req.ByzantineValidators {
		// DuplicateVoteEvidence is the only type of evidence currently
		// implemented in tendermint but we don't get access to this via the
		// ABCI. Instead, we're just given a validator address and block height.
		// The conflicting vote data is kept within the consensus engine itself.
		fmt.Println("evidence", evidence.Validator.Address)

		// TODO what prevents someone from resubmitting evidence?
		// evidence.ValidateBasic() seems to already be called by Tendermint,
		// I think it takes care of catching duplicates as well...
		if evidence.Height > (currentHeight - 100) {
			err := m.SlashDoubleSign(evidence.Validator.Address)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *ValidatorsManager) EndBlock(req abci.RequestEndBlock) ([]abci.ValidatorUpdate, error) {
	// Check if the function has been called with NoopValidatorsManager
	if m == nil {
		return nil, nil
	}

	oldValidatorList, err := m.ValidatorList()
	if err != nil {
		return nil, err
	}

	err = m.Elect()
	if err != nil {
		return nil, err
	}

	validatorList, err := m.ValidatorList()
	if err != nil {
		return nil, err
	}

	var validators []abci.ValidatorUpdate
	// Clearing current validators by passing in list of zero-power update to
	// tendermint.
	for _, validator := range oldValidatorList.Validators {
		validators = append(validators, abci.ValidatorUpdate{
			PubKey: abci.PubKey{
				Data: validator.PubKey,
				Type: tmtypes.ABCIPubKeyTypeEd25519,
			},
			Power: 0,
		})
	}

	// After the list of zero-power updates are procecessed by tendermint, the
	// rest of the validators updates will set the tendermint validator set to
	// be exactly the contents of the dpos validators list
	for _, validator := range validatorList.Validators {
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
