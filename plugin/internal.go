package plugin

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"plugin"
	"sort"
	"strings"
	"sync"

	lp "github.com/loomnetwork/go-loom/plugin"
)

var (
	errInvalidPluginInterface = errors.New("invalid plugin interface")
)

func ParseMeta(s string) (*lp.Meta, error) {
	parts := strings.SplitN(string(s), ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid plugin format")
	}

	return &lp.Meta{
		Name:    parts[0],
		Version: parts[1],
	}, nil
}

type Entry struct {
	Path     string
	Meta     lp.Meta
	Contract lp.Contract
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
	return compareMeta(&s[i].Meta, &s[j].Meta) < 0
}

func compareMeta(a *lp.Meta, b *lp.Meta) int {
	ret := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	if ret == 0 {
		ret = -1 * strings.Compare(a.Version, b.Version)
	}

	return ret
}

type Manager struct {
	Dir       string
	contracts map[string]lp.Contract
	mu        sync.Mutex
}

func NewManager(dir string) *Manager {
	return &Manager{
		Dir:       dir,
		contracts: map[string]lp.Contract{},
	}
}

func (m *Manager) List() ([]*Entry, error) {
	files, err := ioutil.ReadDir(m.Dir)
	if err != nil {
		return nil, err
	}

	entries := make([]*Entry, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fullPath := path.Join(m.Dir, file.Name())
		contract, err := m.loadPluginFull(fullPath)
		if err == errInvalidPluginInterface {
			fmt.Printf("encountered invalid plugin at %s\n", fullPath)
		}
		if err != nil {
			fmt.Printf("error while loading plugin at %s, %v\n", fullPath, err)
			continue
		}

		meta, err := contract.Meta()
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}

		entries = append(entries, &Entry{
			Path:     fullPath,
			Meta:     meta,
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
		if compareMeta(meta, &entry.Meta) == 0 {
			return entry, nil
		}
	}

	return nil, ErrPluginNotFound
}

func (m *Manager) UnloadContracts() {}

func (m *Manager) LoadContract(name string, blockHeight int64) (lp.Contract, error) {
	return m.loadPlugin(name)
}

func (m *Manager) loadPlugin(name string) (lp.Contract, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	contract, loaded := m.contracts[name]
	if loaded {
		return contract, nil
	}

	meta, err := ParseMeta(name)
	if err != nil {
		return nil, err
	}

	fullPath := path.Join(m.Dir, meta.Name+".so."+meta.Version)
	contract, err = m.loadPluginFull(fullPath)
	if err != nil {
		return nil, err
	}

	m.contracts[name] = contract
	return contract, nil
}

func (m *Manager) loadPluginFull(path string) (lp.Contract, error) {
	plug, err := plugin.Open(path)
	if err != nil {
		return nil, ErrPluginNotFound
	}

	sym, err := plug.Lookup("Contract")
	if err != nil {
		return nil, errInvalidPluginInterface
	}

	contract, ok := sym.(*lp.Contract)
	if !ok {
		return nil, errInvalidPluginInterface
	}

	return *contract, nil
}
