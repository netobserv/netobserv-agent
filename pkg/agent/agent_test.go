//go:build !race

// (This test isn't thread-safe due to reading agent.status)

package agent

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/gavv/monotime"
	test2 "github.com/mariomac/guara/pkg/test"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/ebpf"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/metrics"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/model"
	"github.com/netobserv/netobserv-ebpf-agent/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var agentIP = "192.168.1.13"

const timeout = 2 * time.Second

func TestFlowsAgent_InvalidConfigs(t *testing.T) {
	for _, tc := range []struct {
		d string
		c Config
	}{{
		d: "invalid export type",
		c: Config{Export: "foo"},
	}, {
		d: "GRPC: missing host",
		c: Config{Export: "grpc", TargetPort: 3333},
	}, {
		d: "GRPC: missing port",
		c: Config{Export: "grpc", TargetHost: "flp"},
	}, {
		d: "Kafka: missing brokers",
		c: Config{Export: "kafka"},
	}} {
		t.Run(tc.d, func(t *testing.T) {
			_, err := FlowsAgent(&tc.c)
			assert.Error(t, err)
		})
	}
}

var (
	key1 = ebpf.BpfFlowId{
		SrcPort: 123,
		DstPort: 456,
	}
	key2 = ebpf.BpfFlowId{
		SrcPort: 333,
		DstPort: 532,
	}
)

func TestFlowsAgent_Decoration(t *testing.T) {
	now := uint64(monotime.Now())
	metrics1 := model.BpfFlowContent{
		BpfFlowMetrics: &ebpf.BpfFlowMetrics{Packets: 3, Bytes: 44, StartMonoTimeTs: now + 1000, EndMonoTimeTs: now + 1_000_000_000,
			IfIndexFirstSeen:   1,
			DirectionFirstSeen: 1,
		},
		AdditionalMetrics: &ebpf.BpfAdditionalMetrics{NbObservedIntf: 1,
			ObservedIntf: [model.MaxObservedInterfaces]ebpf.BpfObservedIntfT{{IfIndex: 3, Direction: 0}},
		},
	}
	metrics2 := model.BpfFlowContent{
		BpfFlowMetrics: &ebpf.BpfFlowMetrics{Packets: 7, Bytes: 33, StartMonoTimeTs: now, EndMonoTimeTs: now + 2_000_000_000,
			IfIndexFirstSeen:   4,
			DirectionFirstSeen: 0,
		},
		AdditionalMetrics: &ebpf.BpfAdditionalMetrics{NbObservedIntf: 2,
			ObservedIntf: [model.MaxObservedInterfaces]ebpf.BpfObservedIntfT{{IfIndex: 1, Direction: 1}, {IfIndex: 99, Direction: 1}},
		},
	}
	flows := map[ebpf.BpfFlowId]model.BpfFlowContent{
		key1: metrics1,
		key2: metrics2,
	}

	exported := testAgent(t, flows)
	assert.Len(t, exported, 2)

	// Tests that the decoration stage has been properly executed. It should
	// add the interface name and the agent IP
	for _, f := range exported {
		assert.Equal(t, agentIP, f.AgentIP.String())
		switch f.ID {
		case key1:
			assert.Len(t, f.Interfaces, 2)
			assert.Equal(t, "eth0", f.Interfaces[0].Interface)
			assert.Equal(t, "foo", f.Interfaces[1].Interface)
		case key2:
			assert.Len(t, f.Interfaces, 3)
			assert.Equal(t, "bar", f.Interfaces[0].Interface)
			assert.Equal(t, "eth0", f.Interfaces[1].Interface)
			assert.Equal(t, "unknown", f.Interfaces[2].Interface)
		default:
			assert.Failf(t, "unexpected key", "key: %v", f.ID)
		}
	}
}

func testAgent(t *testing.T, flows map[ebpf.BpfFlowId]model.BpfFlowContent) []*model.Record {
	ebpfTracer := test.NewTracerFake()
	export := test.NewExporterFake()
	agent, err := flowsAgent(
		&Config{
			CacheActiveTimeout: 10 * time.Millisecond,
			CacheMaxFlows:      100,
		},
		metrics.NewMetrics(&metrics.Settings{}),
		test.SliceInformerFake{
			{Name: "eth0", Index: 1},
			{Name: "foo", Index: 3},
			{Name: "bar", Index: 4},
		}, ebpfTracer, export.Export,
		net.ParseIP(agentIP), nil)
	require.NoError(t, err)

	go func() {
		require.NoError(t, agent.Run(context.Background()))
	}()
	test2.Eventually(t, timeout, func(t require.TestingT) {
		require.Equal(t, StatusStarted, agent.status)
	})

	ebpfTracer.AppendLookupResults(flows)
	return export.Get(t, timeout)
}
