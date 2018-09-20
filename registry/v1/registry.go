package registry

import (
	"errors"
	"regexp"

	proto "github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
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
)

func recordKey(name string) []byte {
	return util.PrefixKey([]byte("registry"), []byte(name))
}

func versionRecordKey(name, version string) []byte {
	return util.PrefixKey([]byte("registry"), []byte(name+":"+version))
}

// StateRegistry stores contract meta data for named contracts only, and allows lookup by contract name.
type StateRegistry struct {
	State loomchain.State
}

var _ common.Registry = &StateRegistry{}

func (r *StateRegistry) Register(name string, version string, addr, owner loom.Address) error {
	// In previous builds this function was only called when the name wasn't empty, so to maintain
	// backward compatibility do nothing if the name is empty.
	if name == "" {
		return nil
	}

	err := validateName(name)
	if err != nil {
		return err
	}

	if version != "" {
		data := r.State.Get(versionRecordKey(name, version))
		if len(data) != 0 {
			return common.ErrAlreadyRegistered
		}

		retBytes, err := proto.Marshal(&common.VersionRecord{
			ContractAddrKey: name,
		})
		if err != nil {
			return err
		}

		r.State.Set(versionRecordKey(name, version), retBytes)
	}

	_, err = r.Resolve(name, version)
	if err == nil && version == "" {
		return common.ErrAlreadyRegistered
	} else if err == common.ErrNotFound {
		// No need to have a check whether version is empty or not.
		// If version is empty, then, we need to add this entry.
		// If version is not empty, we already added link between version<->name above, so
		// Error couldnt be due to that.
		data, err := proto.Marshal(&common.Record{
			Name:           name,
			Owner:          owner.MarshalPB(),
			Address:        addr.MarshalPB(),
			InitialVersion: version,
		})
		if err != nil {
			return err
		}
		r.State.Set(recordKey(name), data)
	} else if err != common.ErrNotFound {
		return err
	}

	return nil
}

func (r *StateRegistry) Resolve(name, version string) (loom.Address, error) {
	var key []byte

	if version == "" {
		key = recordKey(name)
	} else {
		var versionRecord common.VersionRecord

		data := r.State.Get(versionRecordKey(name, version))
		if len(data) == 0 {
			return loom.Address{}, common.ErrNotFound
		}

		err := proto.Unmarshal(data, &versionRecord)
		if err != nil {
			return loom.Address{}, err
		}

		key = recordKey(versionRecord.ContractAddrKey)
	}

	data := r.State.Get(key)
	if len(data) == 0 {
		return loom.Address{}, common.ErrNotFound
	}
	var record common.Record
	err := proto.Unmarshal(data, &record)
	if err != nil {
		return loom.Address{}, err
	}
	return loom.UnmarshalAddressPB(record.Address), nil
}

func (r *StateRegistry) GetRecord(contractAddr loom.Address) (*common.Record, error) {
	return nil, common.ErrNotImplemented
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
