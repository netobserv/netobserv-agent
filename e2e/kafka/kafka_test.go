//go:build e2e

package basic

import (
	"context"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/mariomac/guara/pkg/test"
	"github.com/netobserv/netobserv-ebpf-agent/e2e/cluster"
	"github.com/netobserv/netobserv-ebpf-agent/e2e/cluster/tester"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	clusterNamePrefix = "basic-test-cluster"
	testTimeout       = 10 * time.Minute
	namespace         = "default"
)

var (
	testCluster *cluster.Kind
)

func TestMain(m *testing.M) {
	logrus.StandardLogger().SetLevel(logrus.DebugLevel)
	scheme.Scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "kafka.strimzi.io",
		Version: "v1beta2",
		Kind:    "Kafka",
	}, &Kafka{})

	testCluster = cluster.NewKind(
		clusterNamePrefix+time.Now().Format("20060102-150405"),
		path.Join("..", ".."),
		cluster.Timeout(testTimeout),
		cluster.Deploy("kafka-crd", cluster.Deployment{
			Order: cluster.Preconditions, ManifestFile: path.Join("manifests", "10-kafka-crd.yml"),
		}),
		cluster.Deploy("kafka-cluster", cluster.Deployment{
			Order: cluster.ExternalServices, ManifestFile: path.Join("manifests", "11-kafka-cluster.yml"),
			ReadyFunction: func(cfg *envconf.Config) error {
				client, err := cfg.NewClient()
				if err != nil {
					return fmt.Errorf("can't create k8s client: %w", err)
				}
				// wait for kafka to be ready
				kfk := Kafka{ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace, Name: "kafka-cluster",
				}}
				if err := wait.For(conditions.New(client.Resources(namespace)).
					ResourceMatch(&kfk, func(object k8s.Object) bool {
						return object.(*Kafka).Status.Ready()
					}),
					wait.WithTimeout(testTimeout),
				); err != nil {
					return fmt.Errorf("waiting for kafka cluster to be ready: %w", err)
				}
				return nil
			},
		}),
		cluster.Deploy(cluster.FlowLogsPipelineID, cluster.Deployment{
			Order: cluster.NetObservServices, ManifestFile: path.Join("manifests", "20-flp-transformer.yml"),
		}),
		cluster.Deploy(cluster.AgentID, cluster.Deployment{
			Order: cluster.Agent, ManifestFile: path.Join("manifests", "30-agent.yml"),
		}),
		cluster.Deploy("traffic-generators", cluster.Deployment{
			Order: cluster.AfterAgent, ManifestFile: path.Join("manifests", "pods.yml"),
		}),
	)
	testCluster.Run(m)
}

// TestBasicFlowCapture checks that the agent is correctly capturing the request/response flows
// between the pods/service deployed from the manifests/pods.yml file
func TestBasicFlowCapture(t *testing.T) {
	var pci podsConnectInfo
	f1 := features.New("basic flow capture").Setup(
		func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			pci = fetchPodsConnectInfo(ctx, t, cfg)
			logrus.Debugf("fetched connect info: %+v", pci)
			return ctx
		},
	).Assess("correctness of client -> server (as Service) request flows",
		func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			lq := lokiQuery(t,
				`{DstK8S_OwnerName="server",SrcK8S_OwnerName="client"}`+
					`|="\"DstAddr\":\"`+pci.serverServiceIP+`\""`)
			require.NotEmpty(t, lq.Values)
			flow, err := lq.Values[0].FlowData()
			require.NoError(t, err)

			assert.Equal(t, pci.clientIP, flow["SrcAddr"])
			assert.NotZero(t, flow["SrcPort"])
			assert.Equal(t, pci.serverServiceIP, flow["DstAddr"])
			assert.EqualValues(t, 80, flow["DstPort"])

			// At the moment, the result of the client Pod Mac seems to be CNI-dependant, so we will
			// only check that it is well-formed.
			assert.Regexp(t, "^[\\da-fA-F]{2}(:[\\da-fA-F]{2}){5}$", flow["SrcMac"])
			// Same for DstMac when the flow is towards the service
			assert.Regexp(t, "^[\\da-fA-F]{2}(:[\\da-fA-F]{2}){5}$", flow["DstMac"])

			assert.Regexp(t, "^[01]$", lq.Stream["FlowDirection"])
			assert.EqualValues(t, 2048, flow["Etype"])
			assert.EqualValues(t, 6, flow["Proto"])

			// For the values below, we just check that they have reasonable/safe values
			assert.NotZero(t, flow["Bytes"])
			assert.Less(t, flow["Bytes"], float64(600))
			assert.NotZero(t, flow["Packets"])
			assert.Less(t, flow["Packets"], float64(10))
			assert.Less(t, time.Since(asTime(flow["TimeFlowEndMs"])), 15*time.Second)
			assert.Less(t, time.Since(asTime(flow["TimeFlowStartMs"])), 15*time.Second)

			assert.NotEmpty(t, flow["Interface"])
			return ctx
		},
	).Assess("correctness of client -> server (as Pod) request flows",
		func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			lq := lokiQuery(t,
				`{DstK8S_OwnerName="server",SrcK8S_OwnerName="client"}`+
					`|="\"DstAddr\":\"`+pci.serverPodIP+`\""`)
			require.NotEmpty(t, lq.Values)
			flow, err := lq.Values[0].FlowData()
			require.NoError(t, err)

			assert.Equal(t, pci.clientIP, flow["SrcAddr"])
			assert.NotZero(t, flow["SrcPort"])
			assert.Equal(t, pci.serverPodIP, flow["DstAddr"])
			assert.EqualValues(t, 80, flow["DstPort"])

			// At the moment, the result of the client Pod Mac seems to be CNI-dependant, so we will
			// only check that it is well-formed.
			assert.Regexp(t, "^[\\da-fA-F]{2}(:[\\da-fA-F]{2}){5}$", flow["SrcMac"])
			assert.Equal(t, pci.serverMAC, flow["DstMac"])

			assert.Regexp(t, "^[01]$", lq.Stream["FlowDirection"])
			assert.EqualValues(t, 2048, flow["Etype"])

			assert.NotZero(t, flow["Bytes"])
			assert.Less(t, flow["Bytes"], float64(600))
			assert.NotZero(t, flow["Packets"])
			assert.Less(t, flow["Packets"], float64(10))
			assert.Less(t, time.Since(asTime(flow["TimeFlowEndMs"])), 15*time.Second)
			assert.Less(t, time.Since(asTime(flow["TimeFlowStartMs"])), 15*time.Second)

			assert.NotEmpty(t, flow["Interface"])
			return ctx
		},
	).Assess("correctness of server (from Service) -> client response flows",
		func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			lq := lokiQuery(t,
				`{DstK8S_OwnerName="client",SrcK8S_OwnerName="server"}`+
					`|="\"SrcAddr\":\"`+pci.serverServiceIP+`\""`)
			require.NotEmpty(t, lq.Values)
			flow, err := lq.Values[0].FlowData()
			require.NoError(t, err)

			assert.Equal(t, pci.serverServiceIP, flow["SrcAddr"])
			assert.EqualValues(t, 80, flow["SrcPort"])
			assert.Equal(t, pci.clientIP, flow["DstAddr"])
			assert.NotZero(t, flow["DstPort"])

			// When the source is the service, MAC is not well parsed in all CNIs
			assert.Regexp(t, "^[\\da-fA-F]{2}(:[\\da-fA-F]{2}){5}$", flow["SrcMac"])
			assert.Equal(t, pci.clientMAC, flow["DstMac"])

			assert.Regexp(t, "^[01]$", lq.Stream["FlowDirection"])
			assert.EqualValues(t, 2048, flow["Etype"])
			assert.EqualValues(t, 6, flow["Proto"])

			assert.NotZero(t, flow["Bytes"])
			assert.Less(t, flow["Bytes"], float64(1300))
			assert.NotZero(t, flow["Packets"])
			assert.Less(t, flow["Packets"], float64(10))

			assert.Less(t, time.Since(asTime(flow["TimeFlowEndMs"])), 15*time.Second)
			assert.Less(t, time.Since(asTime(flow["TimeFlowStartMs"])), 15*time.Second)

			assert.NotEmpty(t, flow["Interface"])
			return ctx
		},
	).Assess("correctness of server (from Pod) -> client response flows",
		func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			lq := lokiQuery(t,
				`{DstK8S_OwnerName="client",SrcK8S_OwnerName="server"}`+
					`|="\"SrcAddr\":\"`+pci.serverPodIP+`\""`)
			require.NotEmpty(t, lq.Values)
			flow, err := lq.Values[0].FlowData()
			require.NoError(t, err)

			assert.Equal(t, pci.serverPodIP, flow["SrcAddr"])
			assert.EqualValues(t, 80, flow["SrcPort"])
			assert.Equal(t, pci.clientIP, flow["DstAddr"])
			assert.NotZero(t, flow["DstPort"])

			assert.Regexp(t, pci.serverMAC, flow["SrcMac"])
			// At the moment, the result of the client Pod Mac seems to be CNI-dependant, so we will
			// only check that it is well-formed.
			assert.Regexp(t, "^[\\da-fA-F]{2}(:[\\da-fA-F]{2}){5}$", flow["DstMac"])

			assert.Regexp(t, "^[01]$", lq.Stream["FlowDirection"])
			assert.EqualValues(t, 2048, flow["Etype"])
			assert.EqualValues(t, 6, flow["Proto"])

			assert.NotZero(t, flow["Bytes"])
			assert.Less(t, flow["Bytes"], float64(1300))
			assert.NotZero(t, flow["Packets"])
			assert.Less(t, flow["Packets"], float64(10))

			assert.Less(t, time.Since(asTime(flow["TimeFlowEndMs"])), 15*time.Second)
			assert.Less(t, time.Since(asTime(flow["TimeFlowStartMs"])), 15*time.Second)

			assert.NotEmpty(t, flow["Interface"])
			return ctx
		},
	).Feature()
	testCluster.TestEnv().Test(t, f1)
}

type podsConnectInfo struct {
	clientIP        string
	serverServiceIP string
	serverPodIP     string
	clientMAC       string
	serverMAC       string
}

// fetchPodsConnectInfo gets client and server's IP and MAC addresses
func fetchPodsConnectInfo(
	ctx context.Context, t *testing.T, cfg *envconf.Config,
) podsConnectInfo {
	pci := podsConnectInfo{}
	kclient, err := kubernetes.NewForConfig(cfg.Client().RESTConfig())
	require.NoError(t, err)
	var serverPodName string
	// extract source Pod information from kubernetes
	test.Eventually(t, testTimeout, func(t require.TestingT) {
		client, err := kclient.CoreV1().Pods(namespace).
			Get(ctx, "client", metav1.GetOptions{})
		require.NoError(t, err)
		require.NotEmpty(t, client.Status.PodIP)
		pci.clientIP = client.Status.PodIP
	}, test.Interval(time.Second))
	// extract destination pod information from kubernetes
	test.Eventually(t, testTimeout, func(t require.TestingT) {
		server, err := kclient.CoreV1().Pods(namespace).
			List(ctx, metav1.ListOptions{LabelSelector: "app=server"})
		require.NoError(t, err)
		require.Len(t, server.Items, 1)
		require.NotEmpty(t, server.Items)
		require.NotEmpty(t, server.Items[0].Status.PodIP)
		pci.serverPodIP = server.Items[0].Status.PodIP
		serverPodName = server.Items[0].Name
	}, test.Interval(time.Second))
	// extract destination service information from kubernetes
	test.Eventually(t, testTimeout, func(t require.TestingT) {
		server, err := kclient.CoreV1().Services(namespace).
			Get(ctx, "server", metav1.GetOptions{})
		require.NoError(t, err)
		require.NotEmpty(t, server.Spec.ClusterIP)
		pci.serverServiceIP = server.Spec.ClusterIP
	}, test.Interval(time.Second))

	// extract MAC addresses
	pods, err := tester.NewPods(cfg)
	require.NoError(t, err, "instantiating pods' tester")

	test.Eventually(t, testTimeout, func(t require.TestingT) {
		cmac, err := pods.MACAddress(ctx, namespace, "client", "eth0")
		require.NoError(t, err, "getting client's MAC")
		pci.clientMAC = cmac.String()

		smac, err := pods.MACAddress(ctx, namespace, serverPodName, "eth0")
		require.NoError(t, err, "getting server's MAC")
		pci.serverMAC = smac.String()
	})

	return pci
}

func lokiQuery(t *testing.T, logQL string) tester.LokiQueryResult {
	var query *tester.LokiQueryResponse
	test.Eventually(t, testTimeout, func(t require.TestingT) {
		var err error
		query, err = testCluster.Loki().
			Query(1, logQL)
		require.NoError(t, err)
		require.NotNil(t, query)
		require.NotEmpty(t, query.Data.Result)
	}, test.Interval(time.Second))
	require.NotEmpty(t, query.Data.Result)
	result := query.Data.Result[0]
	return result
}

func asTime(t interface{}) time.Time {
	if i, ok := t.(float64); ok {
		return time.UnixMilli(int64(i))
	}
	return time.UnixMilli(0)
}

const conditionReady = "Ready"

var klog = logrus.WithField("component", "Kafka")

// Kafka meta object for its usage within the API
type Kafka struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            *KafkaStatus `json:"status,omitempty"`
}

type KafkaStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (k *Kafka) DeepCopyObject() runtime.Object {
	return &(*k)
}

func (ks *KafkaStatus) Ready() bool {
	if ks == nil {
		return false
	}
	for _, cond := range ks.Conditions {
		klog.WithFields(logrus.Fields{
			"reason": cond.Reason,
			"msg":    cond.Message,
			"type":   cond.Type,
			"status": cond.Status,
		}).Debug("Waiting for kafka to be up and running")
		if cond.Type == conditionReady {
			return cond.Status == metav1.ConditionTrue
		}
	}
	return false
}
