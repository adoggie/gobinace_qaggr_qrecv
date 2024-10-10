package transport

import (
	"context"
	kafka2 "github.com/segmentio/kafka-go"
	// "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type KafkaGoProducer struct {
	server string
	topic  string
	//p      *kafka.Producer
	p *kafka2.Writer
}

func NewKafkaGoProducer(server string, topic string) *KafkaGoProducer {

	return &KafkaGoProducer{server, topic, nil}
}

func (kf *KafkaGoProducer) Topic() string {
	return kf.topic
}

func (kf *KafkaGoProducer) Open(ctx context.Context, stop <-chan interface{}) (err error) {
	kf.p = &kafka2.Writer{
		Addr: kafka2.TCP(kf.server),
		//Topic:       kf.topic,
		Balancer:    &kafka2.LeastBytes{},
		Compression: kafka2.Snappy,
		BatchBytes:  10e7,
	}

	return

}

func (kf *KafkaGoProducer) Send(topic string, data []byte) (err error) {
	err = kf.p.WriteMessages(context.Background(),
		kafka2.Message{
			Topic: topic,
			Key:   []byte{},
			Value: data,
		},
	)
	return nil
}

func (kf *KafkaGoProducer) Close() error {
	return nil
}
