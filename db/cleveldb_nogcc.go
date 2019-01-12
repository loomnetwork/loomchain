// +build !gcc

package db

import (
	"fmt"
)

func LoadCLevelDB(name, dir string) (DBWrapperWithBatch, error) {
	return nil, fmt.Errorf("DBBackend: %s is not available in build without gcc build tag", CLevelDBBackend)
}
