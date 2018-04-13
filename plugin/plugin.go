package plugin

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

type Meta struct {
	Name    string
	Version *version.Version
}

func (m *Meta) Compare(other *Meta) int {
	ret := strings.Compare(m.Name, other.Name)
	if ret == 0 {
		ret = -1 * m.Version.Compare(other.Version)
	}

	return ret
}

func ParseMeta(s string) (*Meta, error) {
	parts := strings.SplitN(string(s), ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid plugin format")
	}

	ver, err := version.NewVersion(parts[1])
	if err != nil {
		return nil, err
	}

	return &Meta{
		Name:    parts[0],
		Version: ver,
	}, nil
}

type Entry struct {
	Path     string
	Meta     Meta
	Contract Contract
}

type Entries []*Entry

// Len returns length of version collection
func (s Entries) Len() int {
	return len(s)
}

// Swap swaps two versions inside the collection by its indices
func (s Entries) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less checks if version at index i is less than version at index j
func (s Entries) Less(i, j int) bool {
	return s[i].Meta.Compare(&s[j].Meta) < 0
}

type Manager struct {
	Dir string
}

func NewManager(dir string) *Manager {
	return &Manager{
		Dir: dir,
	}
}

func (m *Manager) List() ([]*Entry, error) {
	files, err := ioutil.ReadDir(m.Dir)
	if err != nil {
		return nil, err
	}

	var entries []*Entry
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

		entries = append(entries, &Entry{
			Path:     fullPath,
			Meta:     contract.Meta(),
			Contract: contract,
		})
	}

	sort.Sort(Entries(entries))
	return entries, nil
}

func (m *Manager) Find(name string) (*Entry, error) {
	meta, err := ParseMeta(name)
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

func (m *Manager) LoadContract(name string) (Contract, error) {
	entry, err := m.Find(name)
	if err != nil {
		return nil, err
	}
	return entry.Contract, nil
}

func loadPlugin(path string) (Contract, error) {
	plug, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}

	sym, err := plug.Lookup("Contract")
	if err != nil {
		return nil, errInvalidPluginInterface
	}

	contract, ok := sym.(Contract)
	if !ok {
		return nil, errInvalidPluginInterface
	}

	return contract, nil
}
