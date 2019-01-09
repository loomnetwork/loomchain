// +build !gcc

package db

import (
	"fmt"
)

func LoadCLevelDB(name, dir string) (DBWrapper, error) {
	return nil, fmt.Errorf("DBBackend: %s is not available in build without gcc build tag", CLevelDBBackend)
}
