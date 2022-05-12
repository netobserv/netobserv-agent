package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/netobserv/gopipes/pkg/node"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/ebpf"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/exporter"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/flow"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/ifaces"
	"github.com/sirupsen/logrus"
)

var alog = logrus.WithField("component", "agent.Flows")

// Flows reporting agent
type Flows struct {
	// trMutex provides synchronized access to the tracers map
	trMutex sync.Mutex
	// tracers stores a flowTracer implementation for each interface in the system, with a
	// cancel function that allows stopping it when its interface is deleted
	tracers    map[ifaces.Name]cancellableTracer
	accounter  *flow.Accounter
	exporter   flowExporter
	interfaces ifaces.Informer
	filter     interfaceFilter
	// tracerFactory specifies how to instantiate flowTracer implementations
	tracerFactory func(name string, sampling uint32) flowTracer
	cfg           *Config
}

// flowTracer abstracts the interface of ebpf.FlowTracer to allow dependency injection in tests
type flowTracer interface {
	Trace(ctx context.Context, forwardFlows chan<- *flow.Record)
	Register() error
	Unregister() error
}

type cancellableTracer struct {
	tracer flowTracer
	cancel context.CancelFunc
}

// flowExporter abstract the ExportFlows' method of exporter.GRPCProto to allow dependency injection
// in tests
type flowExporter func(in <-chan []*flow.Record)

// FlowsAgent instantiates a new agent, given a configuration.
func FlowsAgent(cfg *Config) (*Flows, error) {
	alog.Info("initializing Flows agent")

	// configure allow/deny interfaces filter
	filter, err := initInterfaceFilter(cfg.Interfaces, cfg.ExcludeInterfaces)
	if err != nil {
		return nil, fmt.Errorf("configuring interface filters: %w", err)
	}

	// configure informer for new interfaces
	var informer ifaces.Informer
	switch cfg.ListenInterfaces {
	case ListenPoll:
		alog.WithField("period", cfg.ListenPollPeriod).
			Debug("listening for new interfaces: use polling")
		informer = ifaces.NewPoller(cfg.ListenPollPeriod, cfg.BuffersLength)
	case ListenWatch:
		alog.Debug("listening for new interfaces: use watching")
		informer = ifaces.NewWatcher(cfg.BuffersLength)
	default:
		alog.WithField("providedValue", cfg.ListenInterfaces).
			Warn("wrong interface listen method. Using file watcher as default")
		informer = ifaces.NewWatcher(cfg.BuffersLength)
	}

	// configure GRPC+Protobuf exporter
	target := fmt.Sprintf("%s:%d", cfg.TargetHost, cfg.TargetPort)
	grpcExporter, err := exporter.StartGRPCProto(target)
	if err != nil {
		return nil, err
	}

	return &Flows{
		tracers:    map[ifaces.Name]cancellableTracer{},
		accounter:  flow.NewAccounter(cfg.CacheMaxFlows, cfg.BuffersLength, cfg.CacheActiveTimeout),
		exporter:   grpcExporter.ExportFlows,
		interfaces: informer,
		filter:     filter,
		tracerFactory: func(name string, sampling uint32) flowTracer {
			return ebpf.NewFlowTracer(name, sampling)
		},
		cfg: cfg,
	}, nil
}

// Run a Flows agent. The function will keep running in the same thread
// until the passed context is canceled
func (f *Flows) Run(ctx context.Context) error {
	alog.Info("starting Flows agent")

	systemSetup()

	tracedRecords, err := f.interfacesManager(ctx)
	if err != nil {
		return err
	}
	graph := f.processRecords(tracedRecords)

	alog.Info("Flows agent successfully started")
	<-ctx.Done()
	alog.Info("stopping Flows agent")

	alog.Debug("waiting for all nodes to finish their pending work")
	<-graph.Done()

	alog.Info("Flows agent stopped")
	return nil
}

// interfacesManager uses an informer to check new/deleted network interfaces. For each running
// interface, it registers a flow tracer that will forward new flows to the returned channel
func (f *Flows) interfacesManager(ctx context.Context) (<-chan *flow.Record, error) {
	slog := alog.WithField("function", "interfacesManager")

	slog.Debug("subscribing for network interface events")
	ifaceEvents, err := f.interfaces.Subscribe(ctx)
	if err != nil {
		return nil, fmt.Errorf("instantiating interfaces' informer: %w", err)
	}

	tracedRecords := make(chan *flow.Record, f.cfg.BuffersLength)
	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Debug("detaching all the flow tracers before closing the records' channel")
				f.detachAllTracers()
				slog.Debug("closing channel and exiting internal goroutine")
				close(tracedRecords)
				return
			case event := <-ifaceEvents:
				slog.WithField("event", event).Debug("received event")
				switch event.Type {
				case ifaces.EventAdded:
					f.onInterfaceAdded(ctx, event.Interface, tracedRecords)
				case ifaces.EventDeleted:
					f.onInterfaceDeleted(event.Interface)
				default:
					slog.WithField("event", event).Warn("unknown event type")
				}
			}
		}
	}()

	return tracedRecords, nil
}

// processRecords creates the tracers --> accounter --> forwarder Flow processing graph
func (f *Flows) processRecords(tracedRecords <-chan *flow.Record) *node.Terminal {
	// The start node receives Records from the eBPF flow tracers. Currently it is just an external
	// channel forwarder, as the Pipes library does not yet accept
	// adding/removing nodes dynamically: https://github.com/mariomac/pipes/issues/5
	alog.Debug("registering tracers' input")
	tracersCollector := node.AsInit(func(out chan<- *flow.Record) {
		for i := range tracedRecords {
			out <- i
		}
	})
	alog.Debug("registering accounter")
	accounter := node.AsMiddle(f.accounter.Account)
	alog.Debug("registering exporter")
	export := node.AsTerminal(f.exporter)
	alog.Debug("connecting graph")
	tracersCollector.SendsTo(accounter)
	accounter.SendsTo(export)
	alog.Debug("starting graph")
	tracersCollector.Start()
	return export
}

func (f *Flows) onInterfaceAdded(ctx context.Context, name ifaces.Name, flowsCh chan *flow.Record) {
	// ignore interfaces that do not match the user configuration acceptance/exclusion lists
	if !f.filter.Allowed(name) {
		alog.WithField("name", name).
			Debug("interface does not match the allow/exclusion filters. Ignoring")
		return
	}
	f.trMutex.Lock()
	defer f.trMutex.Unlock()
	if _, ok := f.tracers[name]; !ok {
		alog.WithField("name", name).Info("interface detected. Registering flow tracer")
		tracer := f.tracerFactory(string(name), f.cfg.Sampling)
		if err := tracer.Register(); err != nil {
			alog.WithField("interface", name).WithError(err).
				Warn("can't register flow tracer. Ignoring")
			return
		}
		tctx, cancel := context.WithCancel(ctx)
		go tracer.Trace(tctx, flowsCh)
		f.tracers[name] = cancellableTracer{
			tracer: tracer,
			cancel: cancel,
		}
	}
}

func (f *Flows) onInterfaceDeleted(name ifaces.Name) {
	f.trMutex.Lock()
	defer f.trMutex.Unlock()
	if ft, ok := f.tracers[name]; ok {
		alog.WithField("name", name).Info("interface deleted. Removing flow tracer")
		ft.cancel()
		delete(f.tracers, name)
		// qdiscs, ingress and egress filters are automatically deleted so we don't need to
		// specifically detach the tracer
	}
}

func (f *Flows) detachAllTracers() {
	f.trMutex.Lock()
	defer f.trMutex.Unlock()
	for name, ft := range f.tracers {
		ft.cancel()
		flog := alog.WithField("name", name)
		flog.Info("unregistering flow tracer")
		if err := ft.tracer.Unregister(); err != nil {
			flog.WithError(err).Warn("can't unregister flow tracer")
		}
	}
	f.tracers = map[ifaces.Name]cancellableTracer{}
}
