package flow

import (
	"container/list"
	"time"

	"github.com/sirupsen/logrus"
)

var dlog = logrus.WithField("component", "flow/Deduper")
var timeNow = time.Now

// deduperCache implement a LRU cache whose elements are evicted if they haven't been accessed
// during the expire duration.
// It is not safe for concurrent access.
type deduperCache struct {
	expire time.Duration
	// key: RecordKey with the interface and MACs erased, to detect duplicates
	// value: listElement pointing to a struct entry
	ifaces map[RecordKey]*list.Element
	// element: entry structs of the ifaces map ordered by expiry time
	entries *list.List
}

type entry struct {
	key        *RecordKey
	ifIndex    uint32
	expiryTime time.Time
}

// Dedupe receives flows and filters these belonging to duplicate interfaces. It will forward
// the flows from the first interface coming to it, until that flow expires in the cache
// (no activity for it during the expiration time)
// The justMark argument tells that the deduper should not drop the duplicate flows but
// set their Duplicate field.
func Dedupe(expireTime time.Duration, justMark bool) func(in <-chan []*Record, out chan<- []*Record) {
	cache := &deduperCache{
		expire:  expireTime,
		entries: list.New(),
		ifaces:  map[RecordKey]*list.Element{},
	}
	return func(in <-chan []*Record, out chan<- []*Record) {
		for records := range in {
			cache.removeExpired()
			fwd := make([]*Record, 0, len(records))
			for _, record := range records {
				if cache.isDupe(&record.RecordKey) {
					if justMark {
						record.Duplicate = true
					} else {
						continue
					}
				}
				fwd = append(fwd, record)
			}
			if len(fwd) > 0 {
				out <- fwd
			}
		}
	}
}

// isDupe returns whether the passed record has been already checked for duplicate for
// another interface
func (c *deduperCache) isDupe(key *RecordKey) bool {
	rk := *key
	// zeroes fields from key that should be ignored from the flow comparison
	rk.IFIndex = 0
	rk.DataLink = DataLink{}
	// If a flow has been accounted previously, whatever its interface was,
	// it updates the expiry time for that flow
	if ele, ok := c.ifaces[rk]; ok {
		fEntry := ele.Value.(*entry)
		fEntry.expiryTime = timeNow().Add(c.expire)
		c.entries.MoveToFront(ele)
		// The input flow is duplicate if its interface is different to the interface
		// of the non-duplicate flow that was first registered in the cache
		return fEntry.ifIndex != key.IFIndex
	}
	// The flow has not been accounted previously (or was forgotten after expiration)
	// so we register it for that concrete interface
	e := entry{
		key:        &rk,
		ifIndex:    key.IFIndex,
		expiryTime: timeNow().Add(c.expire),
	}
	c.ifaces[rk] = c.entries.PushFront(&e)
	return false
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
