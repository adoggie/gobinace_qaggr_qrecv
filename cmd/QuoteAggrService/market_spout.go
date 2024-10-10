package main

import (
	"context"
	"fmt"
	"time"

	czmq "github.com/go-zeromq/goczmq/v4"
	"google.golang.org/protobuf/proto"

	"github.com/adoggie/ccsuites/message"
	"github.com/adoggie/ccsuites/transport"
	"github.com/adoggie/ccsuites/utils"
)

type FanOutManager struct {
	config   *Config
	producer []transport.Producer
}

func NewFanOutManager(config *Config) *FanOutManager {
	return &FanOutManager{config: config}
}

func (f *FanOutManager) init() error {
	var cnt int
	for _, fs := range config.FanOutServers {
		if fs.Enable {
			cnt = cnt + 1
		} else {
			continue
		}

		var pp transport.Producer
		if fs.Name == "zmq" {
			p := transport.NewZmqProducer(fs.Server, fs.Topic)

			var err error

			if p.Sock = czmq.NewSock(czmq.Pub); p.Sock == nil {
				panic("NewSock failed : Pub ")
			}
			if err = p.Sock.Connect(fs.Server); err != nil {
				panic(err.Error())
			}
			pp = p
		}
		if fs.Name == "kafka" {
			pp = transport.NewKafkaGoProducer(fs.Server, fs.Topic)
		}

		f.producer = append(f.producer, pp)
	}
	if cnt == 0 {
		logger.Infoln("No Fanout Endpoint Defined!")
	}
	return nil
}

func (f *FanOutManager) open(ctx context.Context) (err error) {
	stop := make(chan interface{})
	for _, p := range f.producer {
		if err = p.Open(ctx, stop); err != nil {
			logger.Fatalln(err.Error())
		}
	}

	return
}

func (f *FanOutManager) close() {

}

func (f *FanOutManager) send(msg *message.PeriodMessage) (err error) {
	//data := bytes.NewBuffer(nil) // bytes.Buffer
	fmt.Println(utils.FormatTimeYmdHms(time.Now()), "Data Publish. symbols:", len(msg.SymbolInfos),
		msg.SymbolInfos[0].Symbol, "trades:", len(msg.SymbolInfos[0].Trades),
		" incs:", len(msg.SymbolInfos[0].Incs),
		" snaps:", len(msg.SymbolInfos[0].Snaps))
	var bs []byte
	//var data *bytes.Buffer
	if bs, err = proto.Marshal(msg); err == nil {
		//data = bytes.NewBuffer(bs)
		//fmt.Println(data.Len(), " ", data)
		//if f.config.DataCompressed {
		//	buffer := bytes.NewBuffer(nil)
		//	var writer *flate.Writer
		//	if writer, err = flate.NewWriter(buffer, flate.DefaultCompression); err == nil {
		//		var compressed int
		//		if compressed, err = writer.Write(data.Bytes()); err == nil {
		//			_ = compressed
		//			if err = writer.Flush(); err == nil {
		//				data = buffer
		//			}
		//		}
		//	} else {
		//		return
		//	}
		//}

		for _, p := range f.producer {
			//fmt.Println("Data Publish:", p)
			topic := p.Topic()
			fmt.Println("sent data size:", len(bs))
			_ = topic
			if err = p.Send(topic, bs); err != nil {
				logger.Error(err.Error())
			}
		}

		bs = []byte{}
	}
	return
}
