package flow

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/netobserv/netobserv-ebpf-agent/pkg/ebpf"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/metrics"

	"github.com/gavv/monotime"
	"github.com/netobserv/gopipes/pkg/node"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var mtlog = logrus.WithField("component", "flow.MapTracer")

// MapTracer accesses a mapped source of flows (the eBPF PerCPU HashMap), deserializes it into
// a flow Record structure, and performs the accumulation of each perCPU-record into a single flow
type MapTracer struct {
	mapFetcher               mapFetcher
	evictionTimeout          time.Duration
	staleEntriesEvictTimeout time.Duration
	// manages the access to the eviction routines, avoiding two evictions happening at the same time
	evictionCond               *sync.Cond
	lastEvictionNs             uint64
	hmapEvictionCounter        prometheus.Counter
	numberOfEvictedFlows       prometheus.Counter
	timeSpentinLookupAndDelete prometheus.Histogram
	errors                     *metrics.ErrorCounter
}

type mapFetcher interface {
	LookupAndDeleteMap(counter prometheus.Counter) map[ebpf.BpfFlowId][]ebpf.BpfFlowMetrics
	DeleteMapsStaleEntries(timeOut time.Duration)
}

func NewMapTracer(fetcher mapFetcher, evictionTimeout, staleEntriesEvictTimeout time.Duration, m *metrics.Metrics) *MapTracer {
	return &MapTracer{
		mapFetcher:                 fetcher,
		evictionTimeout:            evictionTimeout,
		lastEvictionNs:             uint64(monotime.Now()),
		evictionCond:               sync.NewCond(&sync.Mutex{}),
		staleEntriesEvictTimeout:   staleEntriesEvictTimeout,
		hmapEvictionCounter:        m.CreateHashMapCounter(),
		numberOfEvictedFlows:       m.CreateNumberOfEvictedFlows(),
		timeSpentinLookupAndDelete: m.CreateTimeSpendInLookupAndDelete(),
		errors:                     m.GetErrorsCounter(),
	}
}

// Flush forces reading (and removing) all the flows from the source eBPF map
// and sending the entries to the next stage in the pipeline
func (m *MapTracer) Flush() {
	m.evictionCond.Broadcast()
}

func (m *MapTracer) TraceLoop(ctx context.Context, forceGC bool) node.StartFunc[[]*Record] {
	return func(out chan<- []*Record) {
		evictionTicker := time.NewTicker(m.evictionTimeout)
		go m.evictionSynchronization(ctx, forceGC, out)
		for {
			select {
			case <-ctx.Done():
				evictionTicker.Stop()
				mtlog.Debug("exiting trace loop due to context cancellation")
				return
			case <-evictionTicker.C:
				mtlog.Debug("triggering flow eviction on timer")
				m.Flush()
			}
		}
	}
}

// evictionSynchronization loop just waits for the evictionCond to happen
// and triggers the actual eviction. It makes sure that only one eviction
// is being triggered at the same time
func (m *MapTracer) evictionSynchronization(ctx context.Context, forceGC bool, out chan<- []*Record) {
	// flow eviction loop. It just keeps waiting for eviction until someone triggers the
	// evictionCond.Broadcast signal
	for {
		// make sure we only evict once at a time, even if there are multiple eviction signals
		m.evictionCond.L.Lock()
		m.evictionCond.Wait()
		select {
		case <-ctx.Done():
			mtlog.Debug("context canceled. Stopping goroutine before evicting flows")
			return
		default:
			mtlog.Debug("evictionSynchronization signal received")
			m.evictFlows(ctx, forceGC, out)
		}
		m.evictionCond.L.Unlock()

	}
}

func (m *MapTracer) evictFlows(ctx context.Context, forceGC bool, forwardFlows chan<- []*Record) {
	// it's important that this monotonic timer reports same or approximate values as kernel-side bpf_ktime_get_ns()
	monotonicTimeNow := monotime.Now()
	currentTime := time.Now()

	var forwardingFlows []*Record
	laterFlowNs := uint64(0)
	flows := m.mapFetcher.LookupAndDeleteMap(m.errors.WithValues("CannotDeleteFlows", ""))
	elapsed := time.Since(currentTime)
	for flowKey, flowMetrics := range flows {
		aggregatedMetrics := m.aggregate(flowMetrics)
		// we ignore metrics that haven't been aggregated (e.g. all the mapped values are ignored)
		if aggregatedMetrics.EndMonoTimeTs == 0 {
			continue
		}
		// If it iterated an entry that do not have updated flows
		if aggregatedMetrics.EndMonoTimeTs > laterFlowNs {
			laterFlowNs = aggregatedMetrics.EndMonoTimeTs
		}
		forwardingFlows = append(forwardingFlows, NewRecord(
			flowKey,
			aggregatedMetrics,
			currentTime,
			uint64(monotonicTimeNow),
		))
	}
	m.mapFetcher.DeleteMapsStaleEntries(m.staleEntriesEvictTimeout)
	m.lastEvictionNs = laterFlowNs
	select {
	case <-ctx.Done():
		mtlog.Debug("skipping flow eviction as agent is being stopped")
	default:
		forwardFlows <- forwardingFlows
	}

	if forceGC {
		runtime.GC()
	}
	m.hmapEvictionCounter.Inc()
	m.numberOfEvictedFlows.Add(float64(len(forwardingFlows)))
	m.timeSpentinLookupAndDelete.Observe(elapsed.Seconds())
	mtlog.Debugf("%d flows evicted", len(forwardingFlows))
}

func (m *MapTracer) aggregate(metrics []ebpf.BpfFlowMetrics) *ebpf.BpfFlowMetrics {
	if len(metrics) == 0 {
		mtlog.Warn("invoked aggregate with no values")
		return &ebpf.BpfFlowMetrics{}
	}
	aggr := &ebpf.BpfFlowMetrics{}
	for _, mt := range metrics {
		// eBPF hashmap values are not zeroed when the entry is removed. That causes that we
		// might receive entries from previous collect-eviction timeslots.
		// We need to check the flow time and discard old flows.
		if mt.StartMonoTimeTs <= m.lastEvictionNs || mt.EndMonoTimeTs <= m.lastEvictionNs {
			continue
		}
		Accumulate(aggr, &mt)
	}
	return aggr
}
