package store

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/loomnetwork/loomchain/log"
)

var (
	testno = int(0)
)

type benchFunc func(b *testing.B)

func BenchmarkIavlStore(b *testing.B) {
	log.Setup("debug", "file://-")
	log.Root.With("module", "dual-iavlstore")

	require.NoError(b, os.RemoveAll("testdata"))
	_, err := os.Stat("testdata")
	require.True(b, os.IsNotExist(err))

	diskDbType = "goleveldb"
	numBlocks = 10000
	blockSize = 100
	generateBlocks(b)
	saveFrequency = 500
	versionFrequency = 100
	maxVersions = 2

	benchmarkIavlStore(b, "normal", benchmarkNormal)
	benchmarkIavlStore(b, "maxVersionFrequencySaveFrequency", benchmarkMaxVersionFrequencySaveFrequency)

	files, err := ioutil.ReadDir("./testdata")
	require.NoError(b, err)
	for _, f := range files {
		fmt.Println("size of "+f.Name()+" : ", f.Size())
	}

}

func benchmarkIavlStore(b *testing.B, name string, fn benchFunc) {
	b.Run(name, func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fn(b)
		}
	})
}

func benchmarkNormal(b *testing.B) {
	testno++
	diskDb := getDiskDb(b, "normal"+strconv.Itoa(testno))
	store, err := NewDelayIavlStore(diskDb, 0, 0, 0, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
}

func benchmarkMaxVersionFrequencySaveFrequency(b *testing.B) {
	testno++
	diskDb := getDiskDb(b, "maxVersionFrequencySaveFrequency"+strconv.Itoa(testno))
	store, err := NewDelayIavlStore(diskDb, int64(maxVersions), 0, uint64(saveFrequency), uint64(versionFrequency))
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
}
