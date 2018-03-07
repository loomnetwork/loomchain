package common

import (
	"errors"
	"os/exec"
	"strings"
)

// Execute - Execute the command, return standard output and error
func Execute(command string) (stdOut string, err error) {
	//split command into command and args
	var outByte []byte
	split := strings.Split(command, " ")
	switch len(split) {
	case 0:
		return "", errors.New("no command provided")
	case 1:
		outByte, err = exec.Command(split[0]).Output()
	default:
		outByte, err = exec.Command(split[0], split[1:]...).Output()
	}
	stdOut = string(outByte)
	stdOut = strings.Trim(stdOut, "\n") //trim any new lines
	return
}
