package db

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var _ prometheus.Collector = &statsCollector{}

// A statsCollector is a prometheus.Collector for GoLevelDB database
type statsCollector struct {
	name                string
	database            string
	dbname              string
	dbpath              string
	leveldbnumfiles     *prometheus.Desc
	leveldbstats        *prometheus.Desc
	leveldbsstables     *prometheus.Desc
	leveldbblockpool    *prometheus.Desc
	leveldbcachedblock  *prometheus.Desc
	leveldbopenedtables *prometheus.Desc

	leveldbalivesnaps *prometheus.Desc
	leveldbaliveiters *prometheus.Desc
}

// newStatsCollector creates a new statsCollector with the specified name
func newStatsCollector(name, dbname, dbpath string) *statsCollector {
	const (
		dbSubsystem = "db"
	)

	var (
		labels    = []string{"database"}
		namespace = namespace + name
	)

	return &statsCollector{
		name: name,
		leveldbnumfiles: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbnumfilesatlevel"),
			"the number of files at level",
			labels,
			nil,
		),

		leveldbstats: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbstats"),
			"stats",
			labels,
			nil,
		),

		leveldbsstables: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbsstables"),
			"sstables list for each level.",
			labels,
			nil,
		),

		leveldbblockpool: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbblockpool"),
			"block pool stats.",
			labels,
			nil,
		),

		leveldbcachedblock: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbcachedblock"),
			"size of cached block.",
			labels,
			nil,
		),

		leveldbopenedtables: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbopenedtables"),
			"number of opened tables.",
			labels,
			nil,
		),

		leveldbalivesnaps: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbalivesnaps"),
			"number of alive snapshots.",
			labels,
			nil,
		),

		leveldbaliveiters: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, dbSubsystem, "leveldbaliveiters"),
			"number of alive iterators.",
			labels,
			nil,
		),
	}
}

var _ prometheus.Collector = &statsCollector{}

// Describe implements the prometheus.Collector interface.
func (c *statsCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		c.leveldbnumfiles,
		c.leveldbstats,
		c.leveldbsstables,
		c.leveldbblockpool,
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

	db, err := dbm.NewGoLevelDB(c.dbname, "")
	if err != nil {
		fmt.Println(err)
	}
	s := db.Stats()

	data1, _ := strconv.ParseFloat(s["leveldbnumfilesatlevel"], 64)

	ch <- prometheus.MustNewConstMetric(
		c.leveldbnumfiles,
		prometheus.GaugeValue,
		float64(data1),
		c.name,
	)

	data2, _ := strconv.ParseFloat(s["leveldbstats"], 64)
	ch <- prometheus.MustNewConstMetric(
		c.leveldbstats,
		prometheus.GaugeValue,
		float64(data2),
		c.name,
	)

	data3, _ := strconv.ParseFloat(s["leveldbsstables"], 64)
	ch <- prometheus.MustNewConstMetric(
		c.leveldbsstables,
		prometheus.GaugeValue,
		float64(data3),
		c.name,
	)

	data4, _ := strconv.ParseFloat(s["leveldbblockpool"], 64)
	ch <- prometheus.MustNewConstMetric(
		c.leveldbblockpool,
		prometheus.GaugeValue,
		float64(data4),
		c.name,
	)

	data5, _ := strconv.ParseFloat(s["leveldbcachedblock"], 64)
	ch <- prometheus.MustNewConstMetric(
		c.leveldbcachedblock,
		prometheus.GaugeValue,
		float64(data5),
		c.name,
	)

	data6, _ := strconv.ParseFloat(s["leveldbopenedtables"], 64)

	ch <- prometheus.MustNewConstMetric(
		c.leveldbopenedtables,
		prometheus.GaugeValue,
		float64(data6),
		c.name,
	)

	data7, _ := strconv.ParseFloat(s["leveldbalivesnaps"], 64)

	ch <- prometheus.MustNewConstMetric(
		c.leveldbalivesnaps,
		prometheus.GaugeValue,
		float64(data7),
		c.name,
	)

	data8, _ := strconv.ParseFloat(s["leveldbaliveiters"], 64)

	ch <- prometheus.MustNewConstMetric(
		c.leveldbaliveiters,
		prometheus.GaugeValue,
		float64(data8),
		c.name,
	)

	db.Close()

}
