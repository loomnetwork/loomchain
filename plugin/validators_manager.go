package plugin

import (
	"github.com/loomnetwork/go-loom"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv2"
	"github.com/loomnetwork/loomchain/privval/auth"
	abci "github.com/tendermint/tendermint/abci/types"
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

func (m *ValidatorsManager) Slash(validatorAddr loom.Address) error {
	return dposv2.Slash(m.ctx, validatorAddr)
}

func (m *ValidatorsManager) Reward(validatorAddr loom.Address) error {
	return dposv2.Reward(m.ctx, validatorAddr)
}

func (m *ValidatorsManager) Elect() error {
	return dposv2.Elect(m.ctx)
}

func (m *ValidatorsManager) ValidatorList() (*dposv2.ListValidatorsResponse, error) {
	return dposv2.ValidatorList(m.ctx)
}

func (m *ValidatorsManager) BeginBlock(req abci.RequestBeginBlock, chainID string) error {
	// Check if the function has been called with NoopValidatorsManager
	if m == nil {
		return nil
	}

	for _, signingValidator := range req.Validators {
		localValidatorAddr := loom.LocalAddressFromPublicKey(signingValidator.Validator.PubKey.Data)
		validatorAddr := loom.Address{
			ChainID: chainID,
			Local:   localValidatorAddr,
		}
		err := m.Reward(validatorAddr)
		if err != nil {
			return err
		}
	}

	for _, evidence := range req.ByzantineValidators {
		localValidatorAddr := loom.LocalAddressFromPublicKey(evidence.Validator.PubKey.Data)
		// TODO check that evidence is valid (once tendermint is upgraded)
		validatorAddr := loom.Address{
			ChainID: chainID,
			Local:   localValidatorAddr,
		}
		err := m.Slash(validatorAddr)
		if err != nil {
			return err
		}

	}
	return nil
}

func (m *ValidatorsManager) EndBlock(req abci.RequestEndBlock) ([]abci.Validator, error) {
	// Check if the function has been called with NoopValidatorsManager
	if m == nil {
		return nil, nil
	}

	var validators []abci.Validator
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

	// clearing current validators by passing in list of zero-power update to tendermint
	for _, validator := range oldValidatorList.Validators {
		validators = append(validators, abci.Validator{
			PubKey: abci.PubKey{
				Data: validator.PubKey,
				Type: auth.ABCIPubKeyType,
			},
			Power: 0,
		})
	}

	for _, validator := range validatorList.Validators {
		validators = append(validators, abci.Validator{
			PubKey: abci.PubKey{
				Data: validator.PubKey,
				Type: auth.ABCIPubKeyType,
			},
			Power: validator.Power,
		})
	}

	return validators, nil
}
