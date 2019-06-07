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
	//"github.com/syndtr/goleveldb/leveldb"

	"github.com/loomnetwork/loomchain/log"
)

var (
	testno = int(0)
)

type benchFunc func(b *testing.B)

func BenchmarkIavlStore(b *testing.B) {
	log.Setup("debug", "file://-")
	log.Root.With("module", "diff-iavlstore")

	testno = 0
	require.NoError(b, os.RemoveAll("testdata"))
	_, err := os.Stat("testdata")
	require.True(b, os.IsNotExist(err))

	diskDbType = "goleveldb"
	numBlocks = 1000
	blockSize = 1000
	saveFrequency = 50
	versionFrequency = 20
	maxVersions = 2
	paddingLength := 20
	padding := strings.Repeat("A", paddingLength)

	generateBlocks(b)
	paddoutBlocks(padding)
	fmt.Println("num blocks", numBlocks, "block size", blockSize, "save frequency", saveFrequency,
		"version frequecny", versionFrequency, "max versions", maxVersions, "padding", paddingLength)

	benchmarkIavlStore(b, "normal", benchmarkNormal)
	benchmarkIavlStore(b, "normal-dif", benchmarkNormalDif)
	benchmarkIavlStore(b, "maxVersions-dif", benchmarkMaxVersionsDif)
	benchmarkIavlStore(b, "maxVersions", benchmarkMaxVersions)
	benchmarkIavlStore(b, "VarableCache-dif", benchmarkVarableCacheDif)
	benchmarkIavlStore(b, "maxVersionFreqSaveFreq-dif", benchmarkMaxVersionFrequencySaveFrequencyDif)
	benchmarkIavlStore(b, "maxVerSaveFrequency-diff", benchmarkVersionFrequencyDif)
	benchmarkIavlStore(b, "SaveFrequency-diff", benchmarkSaveFrequencyDif)

	files, err := ioutil.ReadDir("./testdata")
	require.NoError(b, err)
	for _, f := range files {
		require.True(b, f.IsDir())
		fmt.Println("size of "+f.Name()+" : ", DirSize(b, f))
	}
	/*
		for _, f := range files {
			db, err := leveldb.OpenFile(f.Name(), nil)
			require.NoError(b, err)
			stats := leveldb.DBStats{}
			require.NoError(b,  db.Stats(&stats))
			fmt.Println(f.Name(), "stats", stats)
			//require.NoError(b, db.CompactRange(util.Range{nil, nil}))
		}*/
}

func benchmarkIavlStore(b *testing.B, name string, fn benchFunc) {
	b.Run(name, func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fn(b)
		}
	})
}

func benchmarkNormalDif(b *testing.B) {
	testno++
	diskDb := getDiskDb(b, "normal-diff\t"+strconv.Itoa(testno/1000000))
	store, err := NewDiffIavlStore(diskDb, 0, 0, 0, 0, 1000, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store, diskDb)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
}

func benchmarkNormal(b *testing.B) {
	testno++
	diskDb := getDiskDb(b, "Normal\t"+strconv.Itoa(testno/1000000))
	store, err := NewIAVLStore(diskDb, 0, 0, 0, 0, 1000, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store, diskDb)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
}

func benchmarkMaxVersions(b *testing.B) {
	testno++
	diskDb := getDiskDb(b, "maxVers\t"+strconv.Itoa(testno/1000000))
	store, err := NewIAVLStore(diskDb, int64(2), 0, 0, 0, 1000, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store, diskDb)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
}

func benchmarkMaxVersionsDif(b *testing.B) {
	testno++
	diskDb := getDiskDb(b, "maxVers-diff\t"+strconv.Itoa(testno/1000000))
	store, err := NewDiffIavlStore(diskDb, int64(2), 0, 0, 0, 1000, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store, diskDb)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
}

func benchmarkVarableCacheDif(b *testing.B) {
	testno++
	diskDb := getDiskDb(b, "varCache-diff\t"+strconv.Itoa(testno/1000000))
	store, err := NewDiffIavlStore(diskDb, int64(2), 0, 0, 0, 100, 2000)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store, diskDb)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
}

func benchmarkMaxVersionFrequencySaveFrequencyDif(b *testing.B) {
	testno++
	diskDb := getDiskDb(b, "maxVFSF-diff\t"+strconv.Itoa(testno/1000000))
	store, err := NewDiffIavlStore(diskDb, int64(2), 0, uint64(saveFrequency), uint64(versionFrequency), 1000, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store, diskDb)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
}

func benchmarkVersionFrequencyDif(b *testing.B) {
	testno++
	diskDb := getDiskDb(b, "verFreq-diff\t"+strconv.Itoa(testno/1000000))
	store, err := NewDiffIavlStore(diskDb, int64(maxVersions), 0, 0, uint64(versionFrequency), 1000, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store, diskDb)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
}

func benchmarkSaveFrequencyDif(b *testing.B) {
	testno++
	diskDb := getDiskDb(b, "saveFreq-diff\t"+strconv.Itoa(testno/1000000))
	store, err := NewDiffIavlStore(diskDb, int64(maxVersions), 0, uint64(saveFrequency), 0, 1000, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store, diskDb)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
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
