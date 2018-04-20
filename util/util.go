package util

import (
	"os"
)

func PrefixKey(prefix, key []byte) []byte {
	buf := make([]byte, 0, len(prefix)+len(key)+1)
	buf = append(buf, prefix...)
	buf = append(buf, 0)
	buf = append(buf, key...)
	return buf
}

func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func IgnoreErrNotExists(err error) error {
	if perr, ok := err.(*os.PathError); ok {
		if perr.Err == os.ErrNotExist {
			return nil
		}
	}

	return err
}
