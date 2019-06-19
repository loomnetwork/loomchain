package store

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/tendermint/iavl"
	"github.com/tendermint/tendermint/libs/db"

	// "github.com/syndtr/goleveldb/leveldb"

	"github.com/loomnetwork/loomchain/log"
)

const (
	minCache = 200
	maxCache = 2000
	checkDb  = false
)

type benchResult struct {
	name            string
	timeS           time.Duration
	diskSize        int64
	compactDiskSize int64
}

var (
	testno  = int(0)
	results map[string]benchResult
)

type benchFunc func(b require.TestingT, name string)

func TestBenchmark(t *testing.T) {
	//t.Skip()
	log.Setup("debug", "file://-")
	log.Root.With("module", "diff-iavlstore")
	testno = 0
	require.NoError(t, os.RemoveAll("testdata"))
	_, err := os.Stat("testdata")
	os.IsNotExist(err)

	results = make(map[string]benchResult)

	diskDbType = "goleveldb"
	//diskDbType = "membd"
	numBlocks = 5000
	blockSize = 100
	maxVersions = 2
	fmt.Println("numBlocks", numBlocks, "blockSize", blockSize)
	blocks = nil
	blocks = iavl.GenerateBlocks2(numBlocks, blockSize)
	tree = iavl.NewMutableTree(db.NewMemDB(), 0)
	for _, program := range blocks {
		if err := program.Execute(tree); err != nil {
			return
		}
	}

	t.Run("normal", benchNormal)
	t.Run("normalDif", benchNormalDif)
	t.Run("varCacheK", benchVariableCacheKeep)
	t.Run("varCacheDifK", benchVariableCacheDifKeep)
	t.Run("saveFreqDifK", benchSaveFrequencyDifKeep)
	t.Run("saveFeepK", benchSaveFrequencyKeep)
	t.Run("maxVer", benchMaxVersions)
	t.Run("maxVerDif", benchMaxVersionsDif)
	t.Run("varCache", benchVarableCache)
	t.Run("varCacheDif", benchVarableCacheDif)
	t.Run("saveFeq", benchSaveFrequency)
	t.Run("saveFeqDif", benchSaveFrequencyDif)
	t.Run("verFeq", benchVersionFrequency)
	t.Run("verFeqDif", benchVersionFrequencyDif)
	t.Run("maxVerFreqSaveFeq", benchMaxVersionFrequencySaveFrequency)
	t.Run("maxVerFeqSaveFeqDif", benchMaxVersionFrequencySaveFrequencyDif)
	t.Run("varCacheVerFeq", benchVarableCacheVersFeq)
	t.Run("varCacheVerFeqDif", benchVarableCacheVersFeqDif)

	files, err := ioutil.ReadDir("./testdata")
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		fName := f.Name()
		if len(fName) <= 3 {
			continue
		}
		fName = fName[:len(fName)-3]
		size := DirSize(t, f)

		ldb, err := leveldb.OpenFile("./testdata/"+f.Name(), nil)
		require.NoError(t, err)
		require.NoError(t, ldb.CompactRange(util.Range{nil, nil}))
		require.NoError(t, ldb.Close())

		sizeCompact := DirSize(t, f)
		if _, ok := results[fName]; ok {
			r := results[fName]
			r.diskSize = size
			r.compactDiskSize = sizeCompact
			results[fName] = r
		}

	}
	fmt.Println()
	if _, ok := results["normal"]; ok {
		nTime := results["normal"].timeS
		nSize := results["normal"].diskSize
		nCompactSize := results["normal"].compactDiskSize

		fmt.Println("name\ttime\tdisk size\tcompact\tratio time\tratio size\tratio compact")
		for _, r := range results {
			fmt.Printf(
				"%s\t%v\t%v\t%v\t%v\t%v\t%v\n",
				r.name,
				r.timeS.Seconds(),
				r.diskSize,
				r.compactDiskSize,
				uint64(r.timeS.Seconds()*100/nTime.Seconds()),
				uint64(r.diskSize*100/nSize),
				uint64(r.compactDiskSize*100/nCompactSize),
			)
		}
	} else {
		fmt.Println("name\ttime\tdisk size\tcompact")
		for _, r := range results {
			fmt.Printf(
				"%s\t%v\t%v\t%v\n",
				r.name,
				r.timeS.Seconds(),
				r.diskSize,
				r.compactDiskSize,
			)
		}
	}

}

func benchNormal(t *testing.T) {
	timeIavlStore(t, "normal", benchmarkNormal)
}
func benchNormalDif(t *testing.T) {
	timeIavlStore(t, "normal-dif", benchmarkNormalDif)
}
func benchVariableCacheKeep(t *testing.T) {
	timeIavlStore(t, "varCache-keep", benchmarkVariableCacheKeep)
}
func benchVariableCacheDifKeep(t *testing.T) {
	timeIavlStore(t, "varCache-dif-keep", benchmarVariableCacheDifKeep)
}
func benchSaveFrequencyDifKeep(t *testing.T) {
	timeIavlStore(t, "saveFreq-dif-keep", benchmarkSaveFrequencyDifKeep)
}
func benchSaveFrequencyKeep(t *testing.T) {
	timeIavlStore(t, "saveFreq-keep", benchmarkSaveFrequencyKeep)
}
func benchMaxVersions(t *testing.T) {
	timeIavlStore(t, "maxVersions", benchmarkMaxVersions)
}
func benchMaxVersionsDif(t *testing.T) {
	timeIavlStore(t, "maxVersions-dif", benchmarkMaxVersionsDif)
}
func benchVarableCache(t *testing.T) {
	timeIavlStore(t, "maxVerVarableCache", benchmarkVarableCache)
}
func benchVarableCacheDif(t *testing.T) {
	timeIavlStore(t, "maxVerVarableCache-dif", benchmarkVarableCacheDif)
}
func benchSaveFrequency(t *testing.T) {
	timeIavlStore(t, "SaveFrequency", benchmarkSaveFrequency)
}
func benchSaveFrequencyDif(t *testing.T) {
	timeIavlStore(t, "SaveFrequency-diff", benchmarkSaveFrequencyDif)
}
func benchVersionFrequency(t *testing.T) {
	timeIavlStore(t, "maxVerFreq", benchmarkVersionFrequency)
}
func benchVersionFrequencyDif(t *testing.T) {
	timeIavlStore(t, "maxVerFreq-diff", benchmarkVersionFrequencyDif)
}
func benchMaxVersionFrequencySaveFrequency(t *testing.T) {
	timeIavlStore(t, "maxVerFreqSaveFreq", benchmarkMaxVersionFrequencySaveFrequency)
}
func benchMaxVersionFrequencySaveFrequencyDif(t *testing.T) {
	timeIavlStore(t, "maxVerFreqSaveFreq-dif", benchmarkMaxVersionFrequencySaveFrequencyDif)
}
func benchVarableCacheVersFeq(t *testing.T) {
	timeIavlStore(t, "maxVerFreqVarableCache", benchmarkVarableCacheVersFreqDif)
}
func benchVarableCacheVersFeqDif(t *testing.T) {
	timeIavlStore(t, "maxVerFreqVarableCache-dif", benchmarkVarableCacheVersFreq)
}

func timeIavlStore(b require.TestingT, name string, fn benchFunc) {
	startTime := time.Now()

	fn(b, name)

	now := time.Now()
	elapsed := now.Sub(startTime)
	results[name] = benchResult{
		name:  name,
		timeS: elapsed,
	}
	if checkDb {
		diskDb, err := db.NewGoLevelDB(name, "./testdata")
		diskTree := iavl.NewMutableTree(diskDb, 0)
		_, err = diskTree.Load()
		require.NoError(b, err)
		diskTree.Iterate(func(key []byte, value []byte) bool {
			_, treeValue := tree.Get(key)
			require.Zero(b, bytes.Compare(value, treeValue))
			return false
		})
		tree.Iterate(func(key []byte, value []byte) bool {
			_, diskValue := diskTree.Get(key)
			require.Zero(b, bytes.Compare(value, diskValue))
			return false
		})
		diskDb.Close()
	}
}

func benchmarkNormalDif(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewDiffIavlStore(diskDb, 0, 0, 0, 0, 1000, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func benchmarkNormal(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewIAVLStore(diskDb, 0, 0, 0, 0, minCache, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func benchmarVariableCacheDifKeep(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewDiffIavlStore(diskDb, 0, 0, 0, 0, minCache, maxCache)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func benchmarkVariableCacheKeep(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewIAVLStore(diskDb, 0, 0, 0, 0, minCache, maxCache)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func benchmarkSaveFrequencyDifKeep(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewDiffIavlStore(diskDb, 0, 0, uint64(saveFrequency), 0, minCache, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func benchmarkSaveFrequencyKeep(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewIAVLStore(diskDb, 0, 0, uint64(saveFrequency), 0, minCache, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func benchmarkMaxVersions(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewIAVLStore(diskDb, int64(maxVersions), 0, 0, 0, minCache, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func benchmarkMaxVersionsDif(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewDiffIavlStore(diskDb, int64(maxVersions), 0, 0, 0, minCache, 0)
	require.NoError(b, err)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func benchmarkVarableCacheDif(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewDiffIavlStore(diskDb, int64(maxVersions), 0, 0, 0, minCache, maxCache)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func benchmarkVarableCache(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewIAVLStore(diskDb, int64(maxVersions), 0, 0, 0, minCache, maxCache)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func benchmarkMaxVersionFrequencySaveFrequencyDif(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewDiffIavlStore(diskDb, int64(maxVersions), 0, uint64(saveFrequency), uint64(versionFrequency), minCache, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func benchmarkMaxVersionFrequencySaveFrequency(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewIAVLStore(diskDb, int64(maxVersions), 0, uint64(saveFrequency), uint64(versionFrequency), minCache, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func benchmarkVarableCacheVersFreq(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewIAVLStore(diskDb, int64(maxVersions), 0, 0, uint64(versionFrequency), minCache, maxCache)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func benchmarkVarableCacheVersFreqDif(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewIAVLStore(diskDb, int64(maxVersions), 0, 0, uint64(versionFrequency), minCache, maxCache)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func benchmarkVersionFrequency(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewIAVLStore(diskDb, int64(maxVersions), 0, 0, uint64(versionFrequency), minCache, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func benchmarkVersionFrequencyDif(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewDiffIavlStore(diskDb, int64(maxVersions), 0, 0, uint64(versionFrequency), minCache, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func benchmarkSaveFrequencyDif(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewDiffIavlStore(diskDb, int64(maxVersions), 0, uint64(saveFrequency), 0, minCache, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func benchmarkSaveFrequency(b require.TestingT, name string) {
	testno++
	diskDb := getDiskDb(b, name)
	store, err := NewIAVLStore(diskDb, int64(maxVersions), 0, uint64(saveFrequency), 0, minCache, 0)
	require.NoError(b, err)
	executeBlocks(b, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(b, err)
	diskDb.Close()
}

func DirSize(b require.TestingT, fi os.FileInfo) int64 {
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
