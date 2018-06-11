package registry

import (
	"errors"
	"regexp"

	proto "github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
)

const (
	minNameLen = 1
	maxNameLen = 255
)

var (
	ErrAlreadyRegistered = errors.New("name is already registered")
	ErrNotFound          = errors.New("name is not registered")

	validNameRE = regexp.MustCompile("^[a-zA-Z0-9\\.\\-]+$")
)

func recordKey(name string) []byte {
	return util.PrefixKey([]byte("registry"), []byte(name))
}

type Registry interface {
	Register(name string, addr, owner loom.Address) error
	Resolve(name string) (loom.Address, error)
}

type StateRegistry struct {
	State loomchain.State
}

func (r *StateRegistry) Register(name string, addr, owner loom.Address) error {
	err := validateName(name)
	if err != nil {
		return err
	}

	_, err = r.Resolve(name)
	if err == nil {
		return ErrAlreadyRegistered
	}
	if err != ErrNotFound {
		return err
	}

	data, err := proto.Marshal(&Record{
		Name:    name,
		Owner:   owner.MarshalPB(),
		Address: addr.MarshalPB(),
	})
	if err != nil {
		return err
	}
	r.State.Set(recordKey(name), data)
	return nil
}

func (r *StateRegistry) Resolve(name string) (loom.Address, error) {
	data := r.State.Get(recordKey(name))
	if len(data) == 0 {
		return loom.Address{}, ErrNotFound
	}
	var record Record
	err := proto.Unmarshal(data, &record)
	if err != nil {
		return loom.Address{}, err
	}
	return loom.UnmarshalAddressPB(record.Address), nil
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
