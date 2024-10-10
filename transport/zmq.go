package transport

import (
	"context"
	"fmt"
	czmq "github.com/go-zeromq/goczmq/v4"
)

type ZmqProducer struct {
	server string
	topic  string
	Sock   *czmq.Sock
}

type ZmqConsumer struct {
}

func NewZmqProducer(server string, topic string) *ZmqProducer {
	return &ZmqProducer{server, topic, nil}
}

func (p *ZmqProducer) Open(ctx context.Context, stop <-chan interface{}) error {
	return nil
}

func (p *ZmqProducer) Send(topic string, data []byte) error {
	if p.Sock == nil {
		return nil
	}
	if topic != "" {
		if err := p.Sock.SendFrame([]byte(topic), czmq.FlagMore); err != nil {
			fmt.Println("zmq send frame1 error:", err.Error())
		}
	}
	if err := p.Sock.SendFrame(data, czmq.FlagNone); err != nil {
		fmt.Println("zmq send frame2 error:", err.Error())
	}
	return nil
}

func (p *ZmqProducer) Close() error {
	return nil
}

func (p *ZmqProducer) Topic() string {
	return p.topic
}

func NewZmqConsumer() *ZmqConsumer {
	return &ZmqConsumer{}
}

func (p *ZmqConsumer) Open(data chan []byte, ctx context.Context, stop <-chan interface{}) error {
	return nil
}

func (p *ZmqConsumer) Close() error {
	return nil
}
