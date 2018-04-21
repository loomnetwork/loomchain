package vm

import (
	"bufio"
	"io"
	"strings"
)

type SolOutput struct {
	FileName     string
	ContractName string
	Description  string
	Text         string
}

func MarshalSolOutput(r io.Reader) (*SolOutput, error) {
	reader := bufio.NewReader(r)

	// Skip empty lines
	var line string
	var err error
	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		if strings.HasPrefix(line, "=") {
			break
		}
	}

	output := &SolOutput{}
	// TODO: fill in other fields

	_, err = reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	text, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	output.Text = text[:len(text)-1]
	return output, nil

}
