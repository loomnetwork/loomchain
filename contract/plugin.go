package contract

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"plugin"
	"sort"
	"strings"

	"github.com/hashicorp/go-version"
)

var (
	errInvalidPluginInterface = errors.New("invalid plugin interface")
)

type PluginMeta struct {
	Name    string
	Version *version.Version
}

func (m *PluginMeta) Compare(other *PluginMeta) int {
	ret := strings.Compare(m.Name, other.Name)
	if ret == 0 {
		ret = -1 * m.Version.Compare(other.Version)
	}

	return ret
}

func ParsePluginMeta(s string) (*PluginMeta, error) {
	parts := strings.SplitN(string(s), ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid plugin format")
	}

	ver, err := version.NewVersion(parts[1])
	if err != nil {
		return nil, err
	}

	return &PluginMeta{
		Name:    parts[0],
		Version: ver,
	}, nil
}

type PluginEntry struct {
	Path     string
	Meta     PluginMeta
	Contract PluginContract
}

type PluginEntries []*PluginEntry

// Len returns length of version collection
func (s PluginEntries) Len() int {
	return len(s)
}

// Swap swaps two versions inside the collection by its indices
func (s PluginEntries) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less checks if version at index i is less than version at index j
func (s PluginEntries) Less(i, j int) bool {
	return s[i].Meta.Compare(&s[j].Meta) < 0
}

type PluginManager struct {
	Dir string
}

func NewPluginManager(dir string) *PluginManager {
	return &PluginManager{
		Dir: dir,
	}
}

func (m *PluginManager) List() ([]*PluginEntry, error) {
	files, err := ioutil.ReadDir(m.Dir)
	if err != nil {
		return nil, err
	}

	var entries []*PluginEntry
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fullPath := path.Join(m.Dir, file.Name())
		contract, err := loadPlugin(fullPath)
		if err == errInvalidPluginInterface {
			fmt.Printf("encountered invalid plugin at %s\n", fullPath)
		}
		if err != nil {
			continue
		}

		entries = append(entries, &PluginEntry{
			Path:     fullPath,
			Meta:     contract.Meta(),
			Contract: contract,
		})
	}

	sort.Sort(PluginEntries(entries))
	return entries, nil
}

func (m *PluginManager) Find(name string) (*PluginEntry, error) {
	meta, err := ParsePluginMeta(name)
	if err != nil {
		return nil, err
	}

	allEntries, err := m.List()
	if err != nil {
		return nil, err
	}

	for _, entry := range allEntries {
		if entry.Meta.Compare(meta) == 0 {
			return entry, nil
		}
	}

	return nil, errors.New("contract not found")
}

func (m *PluginManager) LoadContract(name string) (PluginContract, error) {
	entry, err := m.Find(name)
	if err != nil {
		return nil, err
	}
	return entry.Contract, nil
}

func loadPlugin(path string) (PluginContract, error) {
	plug, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}

	contractsPlug, err := plug.Lookup("Contract")
	if err != nil {
		return nil, errInvalidPluginInterface
	}

	contract, ok := contractsPlug.(PluginContract)
	if !ok {
		return nil, errInvalidPluginInterface
	}

	return contract, nil
}
