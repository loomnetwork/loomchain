package common

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Credit: https://stackoverflow.com/questions/5884154/read-text-file-into-string-array-and-write

// ReadLines reads a whole file into memory
// and returns a slice of its lines.
func ReadLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// WriteLines writes the lines to the given file.
func WriteLines(lines []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}

// Copy the src file to dst. Any existing file will be overwritten and will not
// copy file attributes.
// credit: https://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file-in-golang
func Copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

//___________________________________________________________________________________________-
// Directory Processing

// OperateOnDir - loop through files in the path and perform the Operation
func OperateOnDir(path string, op Operation) {
	filepath.Walk(path, visitFunc(op))
}

type Operation func(path string) error // nolint

func visitFunc(op Operation) filepath.WalkFunc {
	return func(path string, f os.FileInfo, err error) error {
		return op(path)
	}
}
