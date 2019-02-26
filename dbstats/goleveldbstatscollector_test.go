package dbstats

import (
	"encoding/binary"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/db"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGoLevelDBMetricsCollector(t *testing.T) {

	numItems := int64(100)
	internal := map[int64]int64{}
	for i := 0; i < int(numItems); i++ {
		internal[int64(i)] = int64(0)
	}
	db, err := db.NewGoLevelDB("app2", "")
	if err != nil {
		t.Fatal(err.Error())
		return
	}


	for i := 0; i < 1000; i++ {
		// Write something

		idx := (int64(cmn.RandInt()) % numItems)
		internal[idx]++
		val := internal[idx]
		idxBytes := int642Bytes(int64(idx))
		valBytes := int642Bytes(int64(val))
		db.Set(
			idxBytes,
			valBytes,
		)
	}

	for i := 0; i < 1000; i++ {
		// Read something
		idx := (int64(cmn.RandInt()) % numItems)
		val := internal[idx]
		idxBytes := int642Bytes(int64(idx))
		_ = int642Bytes(int64(val))
		_ = db.Get(idxBytes)

	}

	fmt.Println("ok, starting Collection")


}

// testCollector performs a single metrics collection pass against the input
// prometheus.Collector, and returns a string containing metrics output.
// Go level DB metrics are available at end point and get ingested in prometheus
func testCollector(collector prometheus.Collector) {
	if err := prometheus.Register(collector); err != nil {
		fmt.Println(err)
	}
	defer prometheus.Unregister(collector)

	promServer := httptest.NewServer(prometheus.Handler())
	defer promServer.Close()

	resp, err := http.Get(promServer.URL)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Print(string(buf))
}




func int642Bytes(
	i int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func bytes2Int64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}
