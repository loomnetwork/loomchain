package db

import (
	"strconv"

	"github.com/loomnetwork/go-loom"
	"github.com/prometheus/client_golang/prometheus"
)

var _ prometheus.Collector = &statsCollector{}

// A statsCollector is a prometheus.Collector for GoLevelDB database
type statsCollector struct {
	db       *GoLevelDB
	name     string
	database string
	dbname   string
	dbpath   string
	log      *loom.Logger
	//leveldbnumfiles     *prometheus.Desc
	//	leveldbstats        *prometheus.Desc
	//	leveldbsstables     *prometheus.Desc
	//	leveldbblockpool    *prometheus.Desc
	leveldbcachedblock  *prometheus.Desc
	leveldbopenedtables *prometheus.Desc

	leveldbalivesnaps *prometheus.Desc
	leveldbaliveiters *prometheus.Desc
}

// newStatsCollector creates a new statsCollector with the specified name
func newStatsCollector(name string, logger *loom.Logger, db *GoLevelDB) *statsCollector {
	const (
		dbSubsystem = "db"
	)

	var (
		labels    = []string{"database"}
		namespace = namespace
	)

	return &statsCollector{
		db:   db,
		name: name,
		log:  logger,

		leveldbcachedblock: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbcachedblock"),
			"size of cached block.",
			labels,
			prometheus.Labels{"db": name},
		),

		leveldbopenedtables: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbopenedtables"),
			"number of opened tables.",
			labels,
			prometheus.Labels{"db": name},
		),

		leveldbalivesnaps: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbalivesnaps"),
			"number of alive snapshots.",
			labels,
			prometheus.Labels{"db": name},
		),

		leveldbaliveiters: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbaliveiters"),
			"number of alive iterators.",
			labels,
			prometheus.Labels{"db": name},
		),
	}
}

var _ prometheus.Collector = &statsCollector{}

// Describe implements the prometheus.Collector interface.
func (c *statsCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		c.leveldbcachedblock,
		c.leveldbopenedtables,
		c.leveldbalivesnaps,
		c.leveldbaliveiters,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect implements the prometheus.Collector interface.
func (c *statsCollector) Collect(ch chan<- prometheus.Metric) {

	s := c.db.Stats()

	data1, err := strconv.ParseFloat(s["leveldb.cachedblock"], 64)

	if err != nil {
		c.log.Error("Parse Error", "err", err)
	}

	ch <- prometheus.MustNewConstMetric(
		c.leveldbcachedblock,
		prometheus.GaugeValue,
		float64(data1),
		c.name,
	)

	data2, err := strconv.ParseFloat(s["leveldb.openedtables"], 64)

	if err != nil {
		c.log.Error("Parse Error", "err", err)
	}

	ch <- prometheus.MustNewConstMetric(
		c.leveldbopenedtables,
		prometheus.GaugeValue,
		float64(data2),
		c.name,
	)

	data3, err := strconv.ParseFloat(s["leveldb.alivesnaps"], 64)

	if err != nil {
		c.log.Error("Parse Error", "err", err)
	}

	ch <- prometheus.MustNewConstMetric(
		c.leveldbalivesnaps,
		prometheus.GaugeValue,
		float64(data3),
		c.name,
	)

	data4, err := strconv.ParseFloat(s["leveldb.aliveiters"], 64)

	if err != nil {
		c.log.Error("Parse Error", "err", err)
	}

	ch <- prometheus.MustNewConstMetric(
		c.leveldbaliveiters,
		prometheus.GaugeValue,
		float64(data4),
		c.name,
	)

}
