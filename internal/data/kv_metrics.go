package data

import (
	"github.com/cockroachdb/pebble"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

func RegisterMetrics(pebbleDB *pebble.DB) {
	blockCacheHits := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pebble_block_cache_hits_total",
		Help: "Total block cache hits.",
	})
	prometheus.MustRegister(blockCacheHits)
	compactCount := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pebble_compactions_total",
			Help: "Total number of compactions.",
		}, []string{"type"},
	)
	prometheus.MustRegister(compactCount)
	flushCount := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_flush_count",
		Help: "Total number of flushes.",
	})
	prometheus.MustRegister(flushCount)
	memtableSize := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_memtable_size_bytes",
		Help: "Size of memtables in bytes.",
	})
	prometheus.MustRegister(memtableSize)
	walBytesWritten := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_wal_bytes_written_total",
		Help: "Total number of bytes written to WAL.",
	})
	prometheus.MustRegister(walBytesWritten)
	snapshotsCount := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_snapshots_count",
		Help: "Number of open snapshots.",
	})
	prometheus.MustRegister(snapshotsCount)
	tableObsoleteCount := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_table_obsolete_count",
		Help: "Number of obsolete tables.",
	})
	prometheus.MustRegister(tableObsoleteCount)
	go func() {
		tick := time.NewTicker(3 * time.Second)
		defer tick.Stop()
		for range tick.C {
			metrics := pebbleDB.Metrics()
			blockCacheHits.Add(float64(metrics.BlockCache.Hits))
			compactCount.WithLabelValues("total").Set(float64(metrics.Compact.Count))
			compactCount.WithLabelValues("default").Set(float64(metrics.Compact.DefaultCount))
			compactCount.WithLabelValues("move").Set(float64(metrics.Compact.MoveCount))
			compactCount.WithLabelValues("read").Set(float64(metrics.Compact.ReadCount))
			flushCount.Set(float64(metrics.Flush.Count))
			memtableSize.Set(float64(metrics.MemTable.Size))
			walBytesWritten.Set(float64(metrics.WAL.BytesWritten))
			snapshotsCount.Set(float64(metrics.Snapshots.Count))
			tableObsoleteCount.Set(float64(metrics.Table.ObsoleteCount))
		}
	}()
}
