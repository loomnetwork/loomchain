package lib

import (
	"bytes"
	"io/ioutil"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

type TestCase struct {
	RunCmd    string
	TestCmd   string
	Condition string
	Expected  string
}

type TestCases []TestCase

func WriteTestCases(tc TestCases, filename string) error {
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(tc); err != nil {
		return errors.Wrapf(err, "encoding runner TestCases error")
	}

	if err := ioutil.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return err
	}
	return nil
}

func ReadTestCases(filename string) (TestCases, error) {
	var tc TestCases
	if _, err := toml.DecodeFile(filename, &tc); err != nil {
		return tc, err
	}
	return tc, nil
}
