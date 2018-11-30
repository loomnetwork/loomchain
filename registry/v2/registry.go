package registry

import (
	"github.com/pkg/errors"
	"regexp"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
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
	activePrefix = []byte("active")
	inactivePrefix = []byte("inactive")
)

func contractActiveAddrKey(contractName string) []byte {
	return util.PrefixKey(activePrefix, contractAddrKeyPrefix, []byte(contractName))
}

func contractActiveRecordKey(contractAddr loom.Address) []byte {
	return util.PrefixKey(activePrefix, contractRecordKeyPrefix, contractAddr.Bytes())
}

func contractInactiveAddrKey(contractName string) []byte {
	return util.PrefixKey(inactivePrefix, contractAddrKeyPrefix, []byte(contractName))
}

func contractInactiveRecordKey(contractAddr loom.Address) []byte {
	return util.PrefixKey(inactivePrefix, contractRecordKeyPrefix, contractAddr.Bytes())
}

// StateRegistry stores contract meta data for named & unnamed contracts, and allows lookup by
// contract name or contract address.
type StateRegistry struct {
	State loomchain.State
}

var _ common.Registry = &StateRegistry{}

// Register stores the given contract meta data, the contract name may be empty.
func (r *StateRegistry) Register(contractName string, contractAddr, owner loom.Address) error {
	if contractName != "" {
		err := validateName(contractName)
		if err != nil {
			return err
		}

		data := r.State.Get(contractActiveAddrKey(contractName))
		if len(data) != 0 {
			return common.ErrAlreadyRegistered
		}

		addrBytes, err := proto.Marshal(contractAddr.MarshalPB())
		if err != nil {
			return err
		}
		r.State.Set(contractActiveAddrKey(contractName), addrBytes)
	}

	data := r.State.Get(contractActiveRecordKey(contractAddr))
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
	r.State.Set(contractActiveRecordKey(contractAddr), recBytes)
	return nil
}

func (r *StateRegistry) Resolve(contractName string) (loom.Address, error) {
	data := r.State.Get(contractActiveAddrKey(contractName))
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
	data := r.State.Get(contractActiveRecordKey(contractAddr))
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

func (r *StateRegistry) GetRecords(active bool) ([]*common.Record, error) {
	var prefix []byte
	if active {
		prefix = util.PrefixKey(activePrefix, contractRecordKeyPrefix)
	} else {
		prefix = util.PrefixKey(inactivePrefix, contractRecordKeyPrefix)
	}
	data := r.State.Range(prefix)
	var records []*common.Record
	for _, kv := range data {
		var record common.Record
		if err := proto.Unmarshal(kv.Value, &record); err != nil {
			return nil, errors.Wrapf(err, "unmarshal record %v", kv.Value)
		}
		records = append(records, &record)
	}
	return records, nil
}

func (r *StateRegistry) IsActive(addr loom.Address) bool {
	return r.State.Has(contractActiveRecordKey(addr))
}

func (r *StateRegistry) SetActive(addr loom.Address) error {
	if (r.State.Has(contractActiveRecordKey(addr))) {
		return nil
	}
	if (!r.State.Has(contractInactiveRecordKey(addr))) {
		return errors.Wrapf(common.ErrNotFound, "looking for address %v", addr)
	}

	data := r.State.Get(contractInactiveRecordKey(addr))
	r.State.Delete(contractInactiveRecordKey(addr))
	if len(data) == 0 {
		return errors.Wrapf(common.ErrNotFound, "looking for address %v", addr)
	}

	r.State.Set(contractActiveRecordKey(addr), data)
	return nil
}

func (r *StateRegistry) SetInactive(addr loom.Address) error {
	if (r.State.Has(contractInactiveRecordKey(addr))) {
		return nil
	}
	if (!r.State.Has(contractActiveRecordKey(addr))) {
		return errors.Wrapf(common.ErrNotFound, "looking for address %v", addr)
	}

	data := r.State.Get(contractActiveRecordKey(addr))
	r.State.Delete(contractActiveRecordKey(addr))
	if len(data) == 0 {
		return errors.Wrapf(common.ErrNotFound, "looking for address %v", addr)
	}

	r.State.Set(contractInactiveRecordKey(addr), data)
	return nil
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
