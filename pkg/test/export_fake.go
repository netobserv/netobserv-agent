package test

import (
	"testing"
	"time"

	"github.com/netobserv/netobserv-ebpf-agent/pkg/ebpf"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/flow"
)

type ExporterFake struct {
	messages chan []*flow.Record
}

func NewExporterFake() *ExporterFake {
	return &ExporterFake{
		messages: make(chan []*flow.Record, 100),
	}
}

type PerfExporterFake struct {
	messages chan []*ebpf.BpfSockEventT
}

func NewPerfExporterFake() *PerfExporterFake {
	return &PerfExporterFake{
		messages: make(chan []*ebpf.BpfSockEventT, 100),
	}
}

func (ef *ExporterFake) Export(in <-chan []*flow.Record) {
	for i := range in {
		if len(i) > 0 {
			ef.messages <- i
		}
	}
}

func (pef *PerfExporterFake) Export(in <-chan []*ebpf.BpfSockEventT) {
	for i := range in {
		if len(i) > 0 {
			pef.messages <- i
		}
	}
}

func (ef *ExporterFake) Get(t *testing.T, timeout time.Duration) []*flow.Record {
	t.Helper()
	select {
	case <-time.After(timeout):
		t.Fatalf("timeout %s while waiting for a message to be exported", timeout)
		return nil
	case m := <-ef.messages:
		return m
	}
}
