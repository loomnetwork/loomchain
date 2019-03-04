//provides a Prometheus metrics collector for Golevel database backend in tendermint
package db

import (
	"sync"

	loom "github.com/loomnetwork/go-loom"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// namespace is the top-level namespace metric names.
	namespace = "goleveldb"
)

// New creates a new prometheus.Collector that can be registered with
// Prometheus to scrape metrics from a Golevel database handle.
//
// Name should specify a unique name for the collector (preferably name of database backend), and will be added
// as a label to all produced Prometheus metrics.
func New(name string, logger *loom.Logger, db *GoLevelDB) prometheus.Collector {
	return &collector{
		stats: newStatsCollector(name, logger, db),
	}
}

// Enforce that collector is a prometheus.Collector.
var _ prometheus.Collector = &collector{}

// A collector is a prometheus.Collector for Golevel database metrics.
type collector struct {
	mu    sync.Mutex
	stats *statsCollector
}

// Describe implements the prometheus.Collector interface.
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	
         c.stats.Describe(ch)
}

// Collect implements the prometheus.Collector interface.
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	
	 c.stats.Collect(ch)
}
