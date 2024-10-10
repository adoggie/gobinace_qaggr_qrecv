package main

import (
	"context"
	"fmt"
	"sync"

	zmq "github.com/pebbe/zmq4"
	"github.com/shopspring/decimal"
)

// github.com/pebbe/zmq4

/*

go get https://github.com/go-zeromq/zmq4
go get github.com/zeromq/goczmq
解释
*/

type QuoteEndpoint struct {
}

type QuoteReceiver struct {
	config        *Config
	mtx           sync.RWMutex
	positionCache map[string]decimal.Decimal
	//rdb           *redis.Client
	//subsock zmq4.Socket
	//subsock goczmq.Sock

	//sock *czmq.Sock
	sock *zmq.Socket
	ctx  context.Context
}

func NewQuoteReceiver() *QuoteReceiver {
	pr := &QuoteReceiver{}
	return pr
}

func (p *QuoteReceiver) Init(config *Config) *QuoteReceiver {
	p.positionCache = make(map[string]decimal.Decimal)
	p.config = config

	return p
}

func (r *QuoteReceiver) Open(ctx context.Context, stopC <-chan struct{}) (err error) {
	stopC = make(chan struct{})
	err = nil
	if r.sock, err = zmq.NewSocket(zmq.SUB); err != nil {
		panic(fmt.Sprintln("QuoteReceiver.Open %s", err.Error()))
	}
	if err = r.sock.SetSubscribe(""); err != nil {
		panic(fmt.Sprintln("QuoteReceiver.Open %s", err.Error()))
	}
	go func() {

		logger.Info("QuoteReceiver..")
		//p.subsock = zmq4.NewSub(p.ctx)
		//topic := ""

		//var err error

		//r.sock = czmq.NewSock(czmq.Sub)

		//if r.sock, err = czmq.NewSub(r.config.QuoteServeAddr, topic); err != nil {
		//	panic("NewSock failed : Sub ")
		//}
		//r.sock.SetOption(czmq.SockSetRcvhwm(0))
		//r.sock.SetOption(czmq.SockSetSubscribe(""))

		// if err = r.sock.SetMaxmsgsize(0); err != nil {
		// 	panic(err.Error())
		// }
		// if err = r.sock.SetSndhwm(0); err != nil {
		// 	panic(err.Error())
		// }
		// if err = r.sock.SetRcvhwm(0); err != nil {
		// 	panic(err.Error())
		// }

		if r.config.QuoteServeMode == "bind" {
			//if _, err = r.sock.Bind(r.config.QuoteServeAddr); err != nil {
			//	panic(err.Error())
			//}
			if err = r.sock.Bind(r.config.QuoteServeAddr); err != nil {
				panic(err.Error())
			}
			logger.Info("zmq bind:", r.config.QuoteServeAddr)
		} else { // connect
			//if err = r.sock.Connect(r.config.QuoteServeAddr); err != nil {
			//	panic(err.Error())
			//}
			if err = r.sock.Connect(r.config.QuoteServeAddr); err != nil {
				panic(err.Error())
			}
		}

		for {
			var msg [][]byte
			var err error
			//if msg, _, err = p.subsock.RecvFrame(); err != nil {

			//r.sock.RecvBytes()
			if msg, err = r.sock.RecvMessageBytes(0); err != nil {
				logger.Error("QuoteRecv Failed:", err.Error(), " msg:", msg)
				// panic(err.Error())
				continue
			}
			//fmt.Println("msg:", len(msg))
			controller.quoteDataCh <- msg[0]

		}
	}()
	return
}
