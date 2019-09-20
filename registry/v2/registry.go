package registry

import (
	"errors"
	"regexp"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"

	common "github.com/loomnetwork/loomchain/registry"
	appstate "github.com/loomnetwork/loomchain/state"
)

const (
	minNameLen = 1
	maxNameLen = 255
)

var (
	validNameRE = regexp.MustCompile("^[a-zA-Z0-9\\.\\-]+$")

	// Store Keys
	contractAddrKeyPrefix   = []byte("reg_caddr")
	contractRecordKeyPrefix = []byte("reg_crec")
)

func contractAddrKey(contractName string) []byte {
	return util.PrefixKey(contractAddrKeyPrefix, []byte(contractName))
}

func contractRecordKey(contractAddr loom.Address) []byte {
	return util.PrefixKey(contractRecordKeyPrefix, contractAddr.Bytes())
}

// StateRegistry stores contract meta data for named & unnamed contracts, and allows lookup by
// contract name or contract address.
type StateRegistry struct {
	State appstate.State
}

var _ common.Registry = &StateRegistry{}

// Register stores the given contract meta data, the contract name may be empty.
func (r *StateRegistry) Register(contractName string, contractAddr, owner loom.Address) error {
	if contractName != "" {
		err := validateName(contractName)
		if err != nil {
			return err
		}

		data := r.State.Get(contractAddrKey(contractName))
		if len(data) != 0 {
			return common.ErrAlreadyRegistered
		}

		addrBytes, err := proto.Marshal(contractAddr.MarshalPB())
		if err != nil {
			return err
		}
		r.State.Set(contractAddrKey(contractName), addrBytes)
	}

	data := r.State.Get(contractRecordKey(contractAddr))
	if len(data) != 0 {
		return common.ErrAlreadyRegistered
	}

	recBytes, err := proto.Marshal(&common.Record{
		Name:    contractName,
		Owner:   owner.MarshalPB(),
		Address: contractAddr.MarshalPB(),
	})
	if err != nil {
		return err
	}
	r.State.Set(contractRecordKey(contractAddr), recBytes)
	return nil
}

func (r *StateRegistry) Resolve(contractName string) (loom.Address, error) {
	data := r.State.Get(contractAddrKey(contractName))
	if len(data) == 0 {
		return loom.Address{}, common.ErrNotFound
	}
	var contractAddr types.Address
	err := proto.Unmarshal(data, &contractAddr)
	if err != nil {
		return loom.Address{}, err
	}
	return loom.UnmarshalAddressPB(&contractAddr), nil
}

func (r *StateRegistry) GetRecord(contractAddr loom.Address) (*common.Record, error) {
	data := r.State.Get(contractRecordKey(contractAddr))
	if len(data) == 0 {
		return nil, common.ErrNotFound
	}
	var record common.Record
	err := proto.Unmarshal(data, &record)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func validateName(name string) error {
	if len(name) < minNameLen {
		return errors.New("name length too short")
	}

	if len(name) > maxNameLen {
		return errors.New("name length too long")
	}

	if !validNameRE.MatchString(name) {
		return errors.New("invalid name format")
	}

	return nil
}
