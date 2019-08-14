package lib

import (
	"bytes"
	"io/ioutil"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

type Datafile struct {
	Filename string `toml:"Filename"`
	Contents string `toml:"Contents"`
}

type TestCase struct {
	Dir          string     `toml:"Dir"`
	RunCmd       string     `toml:"RunCmd"`
	Condition    string     `toml:"Condition"`
	Expected     []string   `toml:"Expected"`
	Excluded     []string   `toml:"Excluded"`
	Iterations   int        `toml:"Iterations"`
	Delay        int64      `toml:"Delay"` // in millisecond
	All          bool       `toml:"All"`
	Node         int        `toml:"Node"`
	Datafiles    []Datafile `toml:"Datafiles"`
	Json         bool       `toml:"Json"`
	ExpectedJson []string   `toml:"ExpectedJson"`
}

type Tests struct {
	TestCases []TestCase `toml:"TestCases"`
}

func WriteTestCases(tc Tests, filename string) error {
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(tc); err != nil {
		return errors.Wrapf(err, "encoding runner TestCases error")
	}

	if err := ioutil.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return err
	}
	return nil
}

func ReadTestCases(filename string) (Tests, error) {
	var tc Tests
	if _, err := toml.DecodeFile(filename, &tc); err != nil {
		return tc, err
	}
	return tc, nil
}
