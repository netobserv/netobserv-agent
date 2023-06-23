package ebpf

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/btf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/ifaces"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// $BPF_CLANG and $BPF_CFLAGS are set by the Makefile.
//go:generate bpf2go -cc $BPF_CLANG -cflags $BPF_CFLAGS -type flow_metrics_t -type flow_id_t -type flow_record_t -type tcp_drops_t -type dns_record_t Bpf ../../bpf/flows.c -- -I../../bpf/headers

const (
	qdiscType = "clsact"
	// constants defined in flows.c as "volatile const"
	constSampling      = "sampling"
	constTraceMessages = "trace_messages"
	aggregatedFlowsMap = "aggregated_flows"
)

var log = logrus.WithField("component", "ebpf.FlowFetcher")

// FlowFetcher reads and forwards the Flows from the Traffic Control hooks in the eBPF kernel space.
// It provides access both to flows that are aggregated in the kernel space (via PerfCPU hashmap)
// and to flows that are forwarded by the kernel via ringbuffer because could not be aggregated
// in the map
type FlowFetcher struct {
	objects              *BpfObjects
	qdiscs               map[ifaces.Interface]*netlink.GenericQdisc
	egressFilters        map[ifaces.Interface]*netlink.BpfFilter
	ingressFilters       map[ifaces.Interface]*netlink.BpfFilter
	ringbufReader        *ringbuf.Reader
	cacheMaxSize         int
	enableIngress        bool
	enableEgress         bool
	tcpDropsTracePoint   link.Link
	dnsTrackerTracePoint link.Link
}

func NewFlowFetcher(
	traceMessages bool,
	sampling, cacheMaxSize int,
	ingress, egress, tcpDrops, dnsTracker bool,
) (*FlowFetcher, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		log.WithError(err).
			Warn("can't remove mem lock. The agent could not be able to start eBPF programs")
	}

	objects := BpfObjects{}
	spec, err := LoadBpf()
	if err != nil {
		return nil, fmt.Errorf("loading BPF data: %w", err)
	}

	// Resize aggregated flows map according to user-provided configuration
	spec.Maps[aggregatedFlowsMap].MaxEntries = uint32(cacheMaxSize)

	traceMsgs := 0
	if traceMessages {
		traceMsgs = 1
	}
	if err := spec.RewriteConstants(map[string]interface{}{
		constSampling:      uint32(sampling),
		constTraceMessages: uint8(traceMsgs),
	}); err != nil {
		return nil, fmt.Errorf("rewriting BPF constants definition: %w", err)
	}

	if err := spec.LoadAndAssign(&objects, nil); err != nil {
		var ve *ebpf.VerifierError
		if errors.As(err, &ve) {
			// Using %+v will print the whole verifier error, not just the last
			// few lines.
			log.Infof("Verifier error: %+v", ve)
		}
		return nil, fmt.Errorf("loading and assigning BPF objects: %w", err)
	}
	/*
	 * since we load the program only when the we start we need to release
	 * memory used by cached kernel BTF see https://github.com/cilium/ebpf/issues/1063
	 * for more details.
	 */
	btf.FlushKernelSpec()

	var tcpDropsLink link.Link
	if tcpDrops {
		tcpDropsLink, err = link.Tracepoint("skb", "kfree_skb", objects.KfreeSkb, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to attach the BPF program to kfree_skb tracepoint: %w", err)
		}
	}

	var dnsTrackerLink link.Link
	if dnsTracker {
		dnsTrackerLink, err = link.Tracepoint("net", "net_dev_queue", objects.TraceNetPackets, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to attach the BPF program to trace_net_packets: %w", err)
		}
	}

	// read events from igress+egress ringbuffer
	flows, err := ringbuf.NewReader(objects.DirectFlows)
	if err != nil {
		return nil, fmt.Errorf("accessing to ringbuffer: %w", err)
	}
	return &FlowFetcher{
		objects:              &objects,
		ringbufReader:        flows,
		egressFilters:        map[ifaces.Interface]*netlink.BpfFilter{},
		ingressFilters:       map[ifaces.Interface]*netlink.BpfFilter{},
		qdiscs:               map[ifaces.Interface]*netlink.GenericQdisc{},
		cacheMaxSize:         cacheMaxSize,
		enableIngress:        ingress,
		enableEgress:         egress,
		tcpDropsTracePoint:   tcpDropsLink,
		dnsTrackerTracePoint: dnsTrackerLink,
	}, nil
}

// Register and links the eBPF fetcher into the system. The program should invoke Unregister
// before exiting.
func (m *FlowFetcher) Register(iface ifaces.Interface) error {
	ilog := log.WithField("iface", iface)
	// Load pre-compiled programs and maps into the kernel, and rewrites the configuration
	ipvlan, err := netlink.LinkByIndex(iface.Index)
	if err != nil {
		return fmt.Errorf("failed to lookup ipvlan device %d (%s): %w", iface.Index, iface.Name, err)
	}
	qdiscAttrs := netlink.QdiscAttrs{
		LinkIndex: ipvlan.Attrs().Index,
		Handle:    netlink.MakeHandle(0xffff, 0),
		Parent:    netlink.HANDLE_CLSACT,
	}
	qdisc := &netlink.GenericQdisc{
		QdiscAttrs: qdiscAttrs,
		QdiscType:  qdiscType,
	}
	if err := netlink.QdiscDel(qdisc); err == nil {
		ilog.Warn("qdisc clsact already existed. Deleted it")
	}
	if err := netlink.QdiscAdd(qdisc); err != nil {
		if errors.Is(err, fs.ErrExist) {
			ilog.WithError(err).Warn("qdisc clsact already exists. Ignoring")
		} else {
			return fmt.Errorf("failed to create clsact qdisc on %d (%s): %w", iface.Index, iface.Name, err)
		}
	}
	m.qdiscs[iface] = qdisc

	if err := m.registerEgress(iface, ipvlan); err != nil {
		return err
	}

	return m.registerIngress(iface, ipvlan)
}

func (m *FlowFetcher) registerEgress(iface ifaces.Interface, ipvlan netlink.Link) error {
	ilog := log.WithField("iface", iface)
	if !m.enableEgress {
		ilog.Debug("ignoring egress traffic, according to user configuration")
		return nil
	}
	// Fetch events on egress
	egressAttrs := netlink.FilterAttrs{
		LinkIndex: ipvlan.Attrs().Index,
		Parent:    netlink.HANDLE_MIN_EGRESS,
		Handle:    netlink.MakeHandle(0, 1),
		Protocol:  3,
		Priority:  1,
	}
	egressFilter := &netlink.BpfFilter{
		FilterAttrs:  egressAttrs,
		Fd:           m.objects.EgressFlowParse.FD(),
		Name:         "tc/egress_flow_parse",
		DirectAction: true,
	}
	if err := netlink.FilterDel(egressFilter); err == nil {
		ilog.Warn("egress filter already existed. Deleted it")
	}
	if err := netlink.FilterAdd(egressFilter); err != nil {
		if errors.Is(err, fs.ErrExist) {
			ilog.WithError(err).Warn("egress filter already exists. Ignoring")
		} else {
			return fmt.Errorf("failed to create egress filter: %w", err)
		}
	}
	m.egressFilters[iface] = egressFilter
	return nil
}

func (m *FlowFetcher) registerIngress(iface ifaces.Interface, ipvlan netlink.Link) error {
	ilog := log.WithField("iface", iface)
	if !m.enableIngress {
		ilog.Debug("ignoring ingress traffic, according to user configuration")
		return nil
	}
	// Fetch events on ingress
	ingressAttrs := netlink.FilterAttrs{
		LinkIndex: ipvlan.Attrs().Index,
		Parent:    netlink.HANDLE_MIN_INGRESS,
		Handle:    netlink.MakeHandle(0, 1),
		Protocol:  unix.ETH_P_ALL,
		Priority:  1,
	}
	ingressFilter := &netlink.BpfFilter{
		FilterAttrs:  ingressAttrs,
		Fd:           m.objects.IngressFlowParse.FD(),
		Name:         "tc/ingress_flow_parse",
		DirectAction: true,
	}
	if err := netlink.FilterDel(ingressFilter); err == nil {
		ilog.Warn("ingress filter already existed. Deleted it")
	}
	if err := netlink.FilterAdd(ingressFilter); err != nil {
		if errors.Is(err, fs.ErrExist) {
			ilog.WithError(err).Warn("ingress filter already exists. Ignoring")
		} else {
			return fmt.Errorf("failed to create ingress filter: %w", err)
		}
	}
	m.ingressFilters[iface] = ingressFilter
	return nil
}

// Close the eBPF fetcher from the system.
// We don't need an "Close(iface)" method because the filters and qdiscs
// are automatically removed when the interface is down
func (m *FlowFetcher) Close() error {
	log.Debug("unregistering eBPF objects")

	var errs []error

	if m.tcpDropsTracePoint != nil {
		m.tcpDropsTracePoint.Close()
	}

	if m.dnsTrackerTracePoint != nil {
		m.dnsTrackerTracePoint.Close()
	}
	// m.ringbufReader.Read is a blocking operation, so we need to close the ring buffer
	// from another goroutine to avoid the system not being able to exit if there
	// isn't traffic in a given interface
	if m.ringbufReader != nil {
		if err := m.ringbufReader.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if m.objects != nil {
		if err := m.objects.EgressFlowParse.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := m.objects.IngressFlowParse.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := m.objects.AggregatedFlows.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := m.objects.DirectFlows.Close(); err != nil {
			errs = append(errs, err)
		}
		m.objects = nil
	}
	for iface, ef := range m.egressFilters {
		log := log.WithField("interface", iface)
		log.Debug("deleting egress filter")
		if err := doIgnoreNoDev(netlink.FilterDel, netlink.Filter(ef), log); err != nil {
			errs = append(errs, fmt.Errorf("deleting egress filter: %w", err))
		}
	}
	m.egressFilters = map[ifaces.Interface]*netlink.BpfFilter{}
	for iface, igf := range m.ingressFilters {
		log := log.WithField("interface", iface)
		log.Debug("deleting ingress filter")
		if err := doIgnoreNoDev(netlink.FilterDel, netlink.Filter(igf), log); err != nil {
			errs = append(errs, fmt.Errorf("deleting ingress filter: %w", err))
		}
	}
	m.ingressFilters = map[ifaces.Interface]*netlink.BpfFilter{}
	for iface, qd := range m.qdiscs {
		log := log.WithField("interface", iface)
		log.Debug("deleting Qdisc")
		if err := doIgnoreNoDev(netlink.QdiscDel, netlink.Qdisc(qd), log); err != nil {
			errs = append(errs, fmt.Errorf("deleting qdisc: %w", err))
		}
	}
	m.qdiscs = map[ifaces.Interface]*netlink.GenericQdisc{}
	if len(errs) == 0 {
		return nil
	}

	var errStrings []string
	for _, err := range errs {
		errStrings = append(errStrings, err.Error())
	}
	return errors.New(`errors: "` + strings.Join(errStrings, `", "`) + `"`)
}

// doIgnoreNoDev runs the provided syscall over the provided device and ignores the error
// if the cause is a non-existing device (just logs the error as debug).
// If the agent is deployed as part of the Network Observability pipeline, normally
// undeploying the FlowCollector could cause the agent to try to remove resources
// from Pods that have been removed immediately before (e.g. flowlogs-pipeline or the
// console plugin), so we avoid logging some errors that would unnecessarily raise the
// user's attention.
// This function uses generics because the set of provided functions accept different argument
// types.
func doIgnoreNoDev[T any](sysCall func(T) error, dev T, log *logrus.Entry) error {
	if err := sysCall(dev); err != nil {
		if errors.Is(err, unix.ENODEV) {
			log.WithError(err).Error("can't delete. Ignore this error if other pods or interfaces " +
				" are also being deleted at this moment. For example, if you are undeploying " +
				" a FlowCollector or Deployment where this agent is part of")
		} else {
			return err
		}
	}
	return nil
}

func (m *FlowFetcher) ReadRingBuf() (ringbuf.Record, error) {
	return m.ringbufReader.Read()
}

// LookupAndDeleteMap reads all the entries from the eBPF map and removes them from it.
// It returns a map where the key
// For synchronization purposes, we get/delete a whole snapshot of the flows map.
// This way we avoid missing packets that could be updated on the
// ebpf side while we process/aggregate them here
// Changing this method invocation by BatchLookupAndDelete could improve performance
// TODO: detect whether BatchLookupAndDelete is supported (Kernel>=5.6) and use it selectively
// Supported Lookup/Delete operations by kernel: https://github.com/iovisor/bcc/blob/master/docs/kernel-versions.md
// Race conditions here causes that some flows are lost in high-load scenarios
func (m *FlowFetcher) LookupAndDeleteMap() map[BpfFlowId]*BpfFlowMetrics {
	flowMap := m.objects.AggregatedFlows

	iterator := flowMap.Iterate()
	var flow = make(map[BpfFlowId]*BpfFlowMetrics, m.cacheMaxSize)
	var id BpfFlowId
	var metric BpfFlowMetrics

	// Changing Iterate+Delete by LookupAndDelete would prevent some possible race conditions
	// TODO: detect whether LookupAndDelete is supported (Kernel>=4.20) and use it selectively
	for iterator.Next(&id, &metric) {
		if err := flowMap.Delete(id); err != nil {
			log.WithError(err).WithField("flowId", id).
				Warnf("couldn't delete flow entry")
		}
		metricPtr := new(BpfFlowMetrics)
		*metricPtr = metric
		flow[id] = metricPtr
	}
	return flow
}
