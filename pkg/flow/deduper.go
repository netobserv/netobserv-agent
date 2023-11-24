package flow

import (
	"container/list"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/netobserv/netobserv-ebpf-agent/pkg/ebpf"
)

var dlog = logrus.WithField("component", "flow/Deduper")
var timeNow = time.Now

// deduperCache implement a LRU cache whose elements are evicted if they haven't been accessed
// during the expire duration.
// It is not safe for concurrent access.
type deduperCache struct {
	expire time.Duration
	// key: ebpf.BpfFlowId with the interface and MACs erased, to detect duplicates
	// value: listElement pointing to a struct entry
	ifaces map[ebpf.BpfFlowId]*list.Element
	// element: entry structs of the ifaces map ordered by expiry time
	entries *list.List
}

type entry struct {
	key        *ebpf.BpfFlowId
	dnsRecord  *ebpf.BpfDnsRecordT
	ifIndex    uint32
	expiryTime time.Time
	dupList    *[]map[string]uint8
}

// Dedupe receives flows and filters these belonging to duplicate interfaces. It will forward
// the flows from the first interface coming to it, until that flow expires in the cache
// (no activity for it during the expiration time)
// The justMark argument tells that the deduper should not drop the duplicate flows but
// set their Duplicate field.
func Dedupe(expireTime time.Duration, justMark, mergeDup bool) func(in <-chan []*Record, out chan<- []*Record) {
	cache := &deduperCache{
		expire:  expireTime,
		entries: list.New(),
		ifaces:  map[ebpf.BpfFlowId]*list.Element{},
	}
	return func(in <-chan []*Record, out chan<- []*Record) {
		for records := range in {
			cache.removeExpired()
			fwd := make([]*Record, 0, len(records))
			for _, record := range records {
				cache.checkDupe(record, justMark, mergeDup, &fwd)
			}
			if len(fwd) > 0 {
				out <- fwd
			}
		}
	}
}

// checkDupe check current record if its already available nad if not added to fwd records list
func (c *deduperCache) checkDupe(r *Record, justMark, mergeDup bool, fwd *[]*Record) {
	mergeEntry := make(map[string]uint8)
	rk := r.Id
	// zeroes fields from key that should be ignored from the flow comparison
	rk.IfIndex = 0
	rk.SrcMac = [MacLen]uint8{0, 0, 0, 0, 0, 0}
	rk.DstMac = [MacLen]uint8{0, 0, 0, 0, 0, 0}
	rk.Direction = 0
	// If a flow has been accounted previously, whatever its interface was,
	// it updates the expiry time for that flow
	if ele, ok := c.ifaces[rk]; ok {
		fEntry := ele.Value.(*entry)
		fEntry.expiryTime = timeNow().Add(c.expire)
		c.entries.MoveToFront(ele)
		// The input flow is duplicate if its interface is different to the interface
		// of the non-duplicate flow that was first registered in the cache
		// except if the new flow has DNS enrichment in this case will enrich the flow in the cache
		// with DNS info and mark the current flow as duplicate
		if r.Metrics.DnsRecord.Latency != 0 && fEntry.dnsRecord.Latency == 0 {
			// copy DNS record to the cached entry and mark it as duplicate
			fEntry.dnsRecord.Flags = r.Metrics.DnsRecord.Flags
			fEntry.dnsRecord.Id = r.Metrics.DnsRecord.Id
			fEntry.dnsRecord.Latency = r.Metrics.DnsRecord.Latency
			// fall through to do interface check
		}
		if fEntry.ifIndex != r.Id.IfIndex {
			if justMark {
				r.Duplicate = true
				*fwd = append(*fwd, r)
			}
			if mergeDup {
				mergeEntry[r.Interface] = r.Id.Direction
				*fEntry.dupList = append(*fEntry.dupList, mergeEntry)
			}
			return
		}
		*fwd = append(*fwd, r)
		return
	}
	// The flow has not been accounted previously (or was forgotten after expiration)
	// so we register it for that concrete interface
	e := entry{
		key:        &rk,
		dnsRecord:  &r.Metrics.DnsRecord,
		ifIndex:    r.Id.IfIndex,
		expiryTime: timeNow().Add(c.expire),
	}
	if mergeDup {
		mergeEntry[r.Interface] = r.Id.Direction
		r.DupList = append(r.DupList, mergeEntry)
		e.dupList = &r.DupList
	}
	c.ifaces[rk] = c.entries.PushFront(&e)
	*fwd = append(*fwd, r)
}

func (c *deduperCache) removeExpired() {
	now := timeNow()
	ele := c.entries.Back()
	evicted := 0
	for ele != nil && now.After(ele.Value.(*entry).expiryTime) {
		evicted++
		c.entries.Remove(ele)
		delete(c.ifaces, *ele.Value.(*entry).key)
		ele = c.entries.Back()
	}
	if evicted > 0 {
		dlog.WithFields(logrus.Fields{
			"current":    c.entries.Len(),
			"evicted":    evicted,
			"expiryTime": c.expire,
		}).Debug("entries evicted from the deduper cache")
	}
}
