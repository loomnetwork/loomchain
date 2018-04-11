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

type PluginEntry struct {
	Path    string
	Name    string
	Version *version.Version
	Contract
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
	ret := strings.Compare(s[i].Name, s[j].Name)
	if ret == 0 {
		ret = -1 * s[i].Version.Compare(s[j].Version)
	}

	return ret < 0
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

		ver, err := version.NewVersion(contract.Version())
		if err != nil {
			fmt.Printf("invalid plugin version: %s\n", err.Error())
			continue
		}

		entries = append(entries, &PluginEntry{
			Path:     fullPath,
			Name:     contract.Name(),
			Version:  ver,
			Contract: contract,
		})
	}

	sort.Sort(PluginEntries(entries))
	return entries, nil
}

func (m *PluginManager) Find(name, verStr string) (*PluginEntry, error) {
	allEntries, err := m.List()
	if err != nil {
		return nil, err
	}

	var ver *version.Version
	if verStr != "" {
		ver, err = version.NewVersion(verStr)
		if err != nil {
			return nil, err
		}
	}

	for _, entry := range allEntries {
		if entry.Name == name &&
			(ver == nil || entry.Version.Compare(ver) == 0) {
			return entry, nil
		}
	}

	return nil, errors.New("contract not found")
}

func loadPlugin(path string) (Contract, error) {
	plug, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}

	contractsPlug, err := plug.Lookup("Contract")
	if err != nil {
		return nil, errInvalidPluginInterface
	}

	contract, ok := contractsPlug.(Contract)
	if !ok {
		return nil, errInvalidPluginInterface
	}

	return contract, nil
}
