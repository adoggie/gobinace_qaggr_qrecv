package transport

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type Producer interface {
	Open(ctx context.Context, stop <-chan interface{}) error
	Send(topic string, data []byte) error
	Close() error
	Topic() string
}

type KafkaProducer struct {
	server string
	topic  string
	p      *kafka.Producer
}

type KafkaConsumer struct {
}

func NewKafkaProducer(server string, topic string) *KafkaProducer {
	return &KafkaProducer{server, topic, nil}
}

func (kf *KafkaProducer) Topic() string {
	return kf.topic
}

func (kf *KafkaProducer) Open(ctx context.Context, stop <-chan interface{}) (err error) {
	if kf.p, err = kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": kf.server}); err == nil {

	}
	return

}

func (kf *KafkaProducer) Send(topic string, data []byte) (err error) {

	if true {
		buffer := bytes.NewBuffer(nil)
		var writer *zlib.Writer
		writer = zlib.NewWriter(buffer)
		//if writer, err = flate.NewWriter(buffer, flate.DefaultCompression); err == nil {
		if writer != nil {
			var compressed int
			if compressed, err = writer.Write(data); err == nil {
				_ = compressed
				if err = writer.Close(); err == nil {
					data = buffer.Bytes()
				}
			}
		} else {
			fmt.Println("compress error:", err.Error())
		}
	}

	err = kf.p.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          data,
		//Headers:        []kafka.Header{{Key: "myTestHeader", Value: []byte("header values are binary")}},
	}, nil)
	fmt.Println("kafka send data size:", len(data))
	if err != nil {
		if err.(kafka.Error).Code() == kafka.ErrQueueFull {

			// Producer queue is full, wait 1s for messages
			// to be delivered then try again.
			//time.Sleep(time.Second)
			//continue
		}
		fmt.Printf("Failed to produce message: %v\n", err)
	}
	return nil
}

func (kf *KafkaProducer) Close() error {
	return nil
}

func NewKafkaConsumer() *KafkaConsumer {
	return &KafkaConsumer{}
}

func (kf *KafkaConsumer) Open(data chan []byte, ctx context.Context, stop <-chan interface{}) error {
	return nil
}

func (kf *KafkaConsumer) Close() error {
	return nil
}
