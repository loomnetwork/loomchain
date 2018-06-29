package lib

import (
	"bytes"
	"io/ioutil"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

type TestCase struct {
	Dir        string
	RunCmd     string
	Condition  string
	Expected   []string
	Iterations int
	Delay      int64 // in millisecond
	All        bool
}

type Tests struct {
	TestCases []TestCase
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
