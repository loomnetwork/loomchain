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
	ErrNotFound          = errors.New("name is ")

	validNameRE = regexp.MustCompile("^[a-z0-9\\.\\-]+$")
)

func recordKey(name string) []byte {
	return util.PrefixKey([]byte("registry"), []byte(name))
}

func Register(
	state loomchain.State,
	name string,
	addr loom.Address,
	owner loom.Address,
) error {
	err := validateName(name)
	if err != nil {
		return err
	}

	_, err = Resolve(state, name)
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
	state.Set(recordKey(name), data)
	return nil
}

func Unregister(name string) error {
	return errors.New("Unregister: not implemented")
}

func Resolve(state loomchain.ReadOnlyState, name string) (loom.Address, error) {
	data := state.Get(recordKey(name))
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

func ReverseResolve(addr loom.Address) (string, error) {
	return "", errors.New("ReverseResolve: not implemented")
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
