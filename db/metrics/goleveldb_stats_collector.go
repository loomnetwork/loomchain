package metrics

import (
	"github.com/loomnetwork/go-loom"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/syndtr/goleveldb/leveldb"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var _ prometheus.Collector = &GoLevelDBStatsCollector{}

// GoLevelDBStatsCollector is a prometheus.Collector for GoLevelDB database
type GoLevelDBStatsCollector struct {
	db                  *dbm.GoLevelDB
	name                string
	log                 *loom.Logger
	leveldbcachedblock  *prometheus.Desc
	leveldbopenedtables *prometheus.Desc
	leveldbalivesnaps   *prometheus.Desc
	leveldbaliveiters   *prometheus.Desc
	leveldbwriteio      *prometheus.Desc
	leveldbreadio       *prometheus.Desc
}

// NewStatsCollector creates a new Prometheus collector for GoLevelDB stats.
func NewStatsCollector(name string, logger *loom.Logger, db *dbm.GoLevelDB) *GoLevelDBStatsCollector {
	const (
		dbSubsystem = "db"
		namespace   = "goleveldb"
	)

	labels := []string{"database"}

	return &GoLevelDBStatsCollector{
		db:   db,
		name: name,
		log:  logger,

		leveldbcachedblock: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbcachedblock"),
			"size of cached block",
			labels,
			prometheus.Labels{"db": name},
		),

		leveldbopenedtables: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbopenedtables"),
			"number of opened tables",
			labels,
			prometheus.Labels{"db": name},
		),

		leveldbalivesnaps: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbalivesnaps"),
			"number of live snapshots",
			labels,
			prometheus.Labels{"db": name},
		),

		leveldbaliveiters: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbaliveiters"),
			"number of live iterators",
			labels,
			prometheus.Labels{"db": name},
		),

		leveldbwriteio: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbwriteio"),
			"disk io write stats",
			labels,
			prometheus.Labels{"db": name},
		),

		leveldbreadio: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbreadio"),
			"disk io read stats",
			labels,
			prometheus.Labels{"db": name},
		),
	}
}

// Describe implements the prometheus.Collector interface.
func (c *GoLevelDBStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		c.leveldbcachedblock,
		c.leveldbopenedtables,
		c.leveldbalivesnaps,
		c.leveldbaliveiters,
		c.leveldbreadio,
		c.leveldbwriteio,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect implements the prometheus.Collector interface.
func (c *GoLevelDBStatsCollector) Collect(ch chan<- prometheus.Metric) {
	var stats leveldb.DBStats
	err := c.db.DB().Stats(&stats)
	if err != nil {
		c.log.Error("Fetching Stats Error", "err", err)
	} else {
		ch <- prometheus.MustNewConstMetric(
			c.leveldbcachedblock,
			prometheus.GaugeValue,
			float64(stats.BlockCacheSize),
			c.name,
		)
		ch <- prometheus.MustNewConstMetric(
			c.leveldbopenedtables,
			prometheus.GaugeValue,
			float64(stats.OpenedTablesCount),
			c.name,
		)
		ch <- prometheus.MustNewConstMetric(
			c.leveldbalivesnaps,
			prometheus.GaugeValue,
			float64(stats.AliveSnapshots),
			c.name,
		)
		ch <- prometheus.MustNewConstMetric(
			c.leveldbaliveiters,
			prometheus.GaugeValue,
			float64(stats.AliveIterators),
			c.name,
		)
		ch <- prometheus.MustNewConstMetric(
			c.leveldbreadio,
			prometheus.GaugeValue,
			float64(stats.IORead),
			c.name,
		)
		ch <- prometheus.MustNewConstMetric(
			c.leveldbwriteio,
			prometheus.GaugeValue,
			float64(stats.IOWrite),
			c.name,
		)
	}
}
