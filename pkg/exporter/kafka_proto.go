package exporter

import (
	"context"

	"github.com/netobserv/netobserv-ebpf-agent/pkg/flow"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

var klog = logrus.WithField("component", "exporter/KafkaProto")

type kafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafkago.Message) error
}

// KafkaProto exports flows over Kafka, encoded as a protobuf that is understandable by the
// Flowlogs-Pipeline collector
type KafkaProto struct {
	Writer kafkaWriter
}

func (kp *KafkaProto) ExportFlows(input <-chan []*flow.Record) {
	klog.Info("starting Kafka exporter")
	for records := range input {
		kp.batchAndSubmit(records)
	}
}

func (kp *KafkaProto) batchAndSubmit(records []*flow.Record) {
	klog.Debugf("sending %d records", len(records))
	msgs := make([]kafkago.Message, 0, len(records))
	for _, record := range records {
		pbBytes, err := proto.Marshal(flowToPB(record))
		if err != nil {
			klog.WithError(err).Debug("can't encode protobuf message. Ignoring")
			continue
		}
		msgs = append(msgs, kafkago.Message{Value: pbBytes})
	}

	if err := kp.Writer.WriteMessages(context.TODO(), msgs...); err != nil {
		klog.WithError(err).Error("can't write messages into Kafka")
	}
}

type JSONRecord struct {
	*flow.Record
	TimeFlowStart   int64
	TimeFlowEnd     int64
	TimeFlowStartMs int64
	TimeFlowEndMs   int64
}
