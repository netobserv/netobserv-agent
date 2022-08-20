package flow

import (
	"time"

	"github.com/gavv/monotime"
	"github.com/sirupsen/logrus"
)

// Accounter accumulates flows metrics in memory and eventually evicts them via an evictor channel.
// The accounting process is usually done at kernel-space. This type reimplements it at userspace
// for the edge case where packets are submitted directly via ring-buffer because the kernel-side
// accounting map is full.
type Accounter struct {
	maxEntries    int
	interfaceName string
	evictTimeout  time.Duration
	entries       map[RecordKey]RecordMetrics
}

var alog = logrus.WithField("component", "flow/Accounter")

// NewAccounter creates a new Accounter.
// The cache has no limit and it's assumed that eviction is done by the caller.
func NewAccounter(interfaceName string, maxEntries int, evictTimeout time.Duration) *Accounter {
	return &Accounter{
		interfaceName: interfaceName,
		maxEntries:    maxEntries,
		evictTimeout:  evictTimeout,
		entries:       make(map[RecordKey]RecordMetrics, maxEntries),
	}
}

// Account runs in a new goroutine. It reads all the records from the input channel
// and accumulate their metrics internally. Once the metrics have reached their max size
// or the eviction times out, it evicts all the accumulated flows by the returned channel.
func (c *Accounter) Account(in <-chan *RawRecord, out chan<- []*Record) {
	evictTick := time.NewTicker(c.evictTimeout)
	defer evictTick.Stop()
	for {
		select {
		case <-evictTick.C:
			evictingEntries := c.entries
			c.entries = make(map[RecordKey]RecordMetrics, c.maxEntries)
			go evict(c.interfaceName, evictingEntries, out)
		case record, ok := <-in:
			if !ok {
				alog.Debug("input channel closed. Evicting entries")
				// if the records channel is closed, we evict the entries in the
				// same goroutine to wait for all the entries to be sent before
				// closing the channel
				evict(c.interfaceName, c.entries, out)
				alog.Debug("exiting account routine")
				return
			}
			if stored, ok := c.entries[record.RecordKey]; ok {
				stored.Accumulate(&record.RecordMetrics)
			} else {
				if len(c.entries) >= c.maxEntries {
					evictingEntries := c.entries
					c.entries = make(map[RecordKey]RecordMetrics, c.maxEntries)
					go evict(c.interfaceName, evictingEntries, out)
				}
				c.entries[record.RecordKey] = record.RecordMetrics
			}
		}

	}
}

func evict(interfaceName string, entries map[RecordKey]RecordMetrics, evictor chan<- []*Record) {
	now := time.Now()
	monotonicNow := uint64(monotime.Now())
	records := make([]*Record, 0, len(entries))
	for key, metrics := range entries {
		records = append(records, NewRecord(key, metrics, now, monotonicNow, interfaceName))
	}
	evictor <- records
}
