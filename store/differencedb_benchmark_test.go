package store

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

	testno = 0
	require.NoError(b, os.RemoveAll("testdata"))
	_, err := os.Stat("testdata")
	require.True(b, os.IsNotExist(err))

	diskDbType = "goleveldb"
	numBlocks = 20000
	blockSize = 1000
	saveFrequency = 1000
	versionFrequency = 0
	maxVersions = 2
	paddingLength := 1000
	padding := strings.Repeat("A", paddingLength)

	generateBlocks(b)
	paddoutBlocks(padding)
	fmt.Println("num blocks", numBlocks, "block size", blockSize, "save frequency", saveFrequency,
		"version frequecny", versionFrequency, "max versions", maxVersions, "padding", paddingLength)

	benchmarkIavlStore(b, "normal", benchmarkNormal)
	benchmarkIavlStore(b, "maxVersionFrequencySaveFrequency", benchmarkMaxVersionFrequencySaveFrequency)

	files, err := ioutil.ReadDir("./testdata")
	require.NoError(b, err)
	for _, f := range files {
		require.True(b, f.IsDir())
		fmt.Println("size of "+f.Name()+" : ", DirSize(b, f))
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

func paddoutBlocks(padding string) {
	for _, block := range blocks {
		for i := range block.Instructions {
			block.Instructions[i].PadValue(padding)
		}
	}
}

func DirSize(b *testing.B, fi os.FileInfo) int64 {
	var size int64
	cwd, err := os.Getwd()
	require.NoError(b, err)

	path := filepath.Join(cwd, "testdata", fi.Name())
	err = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	require.NoError(b, err)
	return size
}
