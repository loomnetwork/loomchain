package registry

import (
	"errors"
	"regexp"

	proto "github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
	common "github.com/loomnetwork/loomchain/registry"
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

	contractVersionKeyPrefix = []byte("reg_cvk")
)

func contractVersionKey(contractName, contractVersion string) []byte {
	return util.PrefixKey(contractVersionKeyPrefix, []byte(contractName+":"+contractVersion))
}

func contractAddrKey(contractName string) []byte {
	return util.PrefixKey(contractAddrKeyPrefix, []byte(contractName))
}

func contractRecordKey(contractAddr loom.Address) []byte {
	return util.PrefixKey(contractRecordKeyPrefix, contractAddr.Bytes())
}

// StateRegistry stores contract meta data for named & unnamed contracts, and allows lookup by
// contract name or contract address.
type StateRegistry struct {
	State loomchain.State
}

var _ common.Registry = &StateRegistry{}

// Register stores the given contract meta data, the contract name may be empty.
func (r *StateRegistry) Register(contractName string, contractVersion string, contractAddr, owner loom.Address) error {
	if contractName != "" {
		err := validateName(contractName)
		if err != nil {
			return err
		}

		// Cant register sentinel version
		if contractVersion == common.SentinelVersion {
			return common.ErrInvalidContractVersion
		}

		if contractVersion != "" {
			data := r.State.Get(contractVersionKey(contractName, contractVersion))
			if len(data) != 0 {
				return common.ErrAlreadyRegistered
			}

			// Since atleast one version exists, record/overwrite sentinel version key
			r.State.Set(contractVersionKey(contractName, common.SentinelVersion), []byte{1})

			r.State.Set(contractVersionKey(contractName, contractVersion), []byte{1})
		}

		data := r.State.Get(contractAddrKey(contractName))

		// if contract version was not passed, and version is empty, return error for
		// backward compatibility. Otherwise, return null since, it means other Registry
		// refs are already in place.
		if len(data) != 0 {
			if contractVersion != "" {
				return nil
			} else {
				return common.ErrAlreadyRegistered
			}
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
		Name:           contractName,
		Owner:          owner.MarshalPB(),
		Address:        contractAddr.MarshalPB(),
		InitialVersion: contractVersion,
	})
	if err != nil {
		return err
	}
	r.State.Set(contractRecordKey(contractAddr), recBytes)
	return nil
}

func (r *StateRegistry) Resolve(contractName string, contractVersion string) (loom.Address, error) {

	if contractVersion != "" {
		data := r.State.Get(contractVersionKey(contractName, contractVersion))
		if len(data) == 0 {
			return loom.Address{}, common.ErrNotFound
		}
	}

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
