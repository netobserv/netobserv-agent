package udp

import (
	"fmt"
	"testing"
	"time"

	"github.com/mariomac/guara/pkg/test"
	"github.com/netobserv/netobserv-agent/export/pkg/pbflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const timeout = 5 * time.Second

func TestUDPCommunication(t *testing.T) {
	port, err := test.FreeUDPPort()
	require.NoError(t, err)

	server := NewServer(port)
	serverOut, err := server.Listen(0)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, server.Stop())
	}()

	client := NewClient(fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, client.Start())
	defer func() {
		assert.NoError(t, client.Close())
	}()
	require.NoError(t, client.Send(&pbflow.Record{EthProtocol: 123, Bytes: 456, Network: &pbflow.Network{
		SrcAddr: &pbflow.IP{
			IpFamily: &pbflow.IP_Ipv4{Ipv4: 0x11223344},
		},
		DstAddr: &pbflow.IP{
			IpFamily: &pbflow.IP_Ipv4{Ipv4: 0x55667788},
		},
	}}))
	require.NoError(t, client.Send(&pbflow.Record{EthProtocol: 789, Bytes: 101, Network: &pbflow.Network{
		SrcAddr: &pbflow.IP{
			IpFamily: &pbflow.IP_Ipv4{Ipv4: 0x44332211},
		},
		DstAddr: &pbflow.IP{
			IpFamily: &pbflow.IP_Ipv4{Ipv4: 0x88776655},
		},
	}}))

	var r *pbflow.Record
	select {
	case r = <-serverOut:
	case <-time.After(timeout):
		require.Fail(t, "timeout waiting for flows")
	}
	assert.EqualValues(t, 123, r.EthProtocol)
	assert.EqualValues(t, 456, r.Bytes)
	assert.EqualValues(t, 0x11223344, r.GetNetwork().GetSrcAddr().GetIpv4())
	assert.EqualValues(t, 0x55667788, r.GetNetwork().GetDstAddr().GetIpv4())
	select {
	case r = <-serverOut:
	case <-time.After(timeout):
		require.Fail(t, "timeout waiting for flows")
	}
	assert.EqualValues(t, 789, r.EthProtocol)
	assert.EqualValues(t, 101, r.Bytes)
	assert.EqualValues(t, 0x44332211, r.GetNetwork().GetSrcAddr().GetIpv4())
	assert.EqualValues(t, 0x88776655, r.GetNetwork().GetDstAddr().GetIpv4())

	select {
	case r = <-serverOut:
		assert.Failf(t, "shouldn't have received any flow", "Got: %#v", r)
	default:
		//ok!
	}
}

func BenchmarkUDPCommunication(b *testing.B) {
	port, err := test.FreeUDPPort()
	require.NoError(b, err)
	server := NewServer(port)
	serverOut, err := server.Listen(100)
	require.NoError(b, err)
	defer func() {
		assert.NoError(b, server.Stop())
	}()
	client := NewClient(fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(b, client.Start())
	defer func() {
		assert.NoError(b, client.Close())
	}()
	f := pbflow.Record{
		EthProtocol:   2048,
		Bytes:         456,
		Direction:     pbflow.Direction_EGRESS,
		TimeFlowStart: timestamppb.Now(),
		TimeFlowEnd:   timestamppb.Now(),
		Network: &pbflow.Network{
			SrcAddr: &pbflow.IP{
				IpFamily: &pbflow.IP_Ipv4{Ipv4: 0x11223344},
			},
			DstAddr: &pbflow.IP{
				IpFamily: &pbflow.IP_Ipv4{Ipv4: 0x55667788},
			},
		},
		DataLink: &pbflow.DataLink{
			DstMac: 0x112233445566,
			SrcMac: 0x665544332211,
		},
		Transport: &pbflow.Transport{
			Protocol: 1,
			SrcPort:  23000,
			DstPort:  443,
		},
	}
	for i := 0; i < b.N; i++ {
		if err := client.Send(&f); err != nil {
			require.Fail(b, "error", err)
		}
		<-serverOut
	}
}
