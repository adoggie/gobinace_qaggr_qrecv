package main

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/gob"
	"fmt"
	"github.com/adoggie/ccsuites/message"
	"google.golang.org/protobuf/proto"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	zmq "github.com/pebbe/zmq4"

	"github.com/adoggie/ccsuites/core"
	"github.com/adoggie/ccsuites/utils"
	//"github.com/adoggie/ccsuites/core"
)

type FanOutManager struct {
	config *Config
	//socks        []*zmq4.Socket
	destinations []string
	// producer_zmq []*czmq.Sock
	producer_zmq []*zmq.Socket
	msgCh        chan *core.Message
	dirty        atomic.Bool
	mtx          sync.Mutex
}

func NewFanOutManager(config *Config) *FanOutManager {
	return &FanOutManager{config: config}
}

func (f *FanOutManager) init() error {

	for _, addr := range f.config.FanOutServers {
		//func NewPub(endpoints string, options ...SockOption) (*Sock, error) {
		//	s := NewSock(Pub, options...)
		//	return s, s.Attach(endpoints, true)
		//}

		// sock := czmq.NewSock(czmq.Pub)
		var sock *zmq.Socket
		var err error
		if sock, err = zmq.NewSocket(zmq.PUB); err != nil {
			panic(err.Error())
		}
		// sock.SetOption(czmq.SockSetSndhwm(0))
		sock.SetSndhwm(0)

		f.producer_zmq = append(f.producer_zmq, sock)
		_ = sock.Connect(addr)

		//if sock, err := czmq.NewPub(addr); err == nil {
		//	f.producer_zmq = append(f.producer_zmq, sock)
		//	_ = sock.Connect(addr)
		//}
	}
	return nil
}

func (f *FanOutManager) open(ctx context.Context) error {
	for _, producer := range f.producer_zmq {
		_ = producer
	}
	f.msgCh = make(chan *core.Message, 100)
	go func() {
		ticker := time.After(time.Second * 3)
		//defer ticker.Stop()
		for {
			select {
			case msg := <-f.msgCh:
				if err := f.fanout(msg); err != nil {
					fmt.Println("Fanout Error:", err.Error())
				}
			case <-ctx.Done():
				return
			case <-ticker:
				f.dirty.Store(false)
				ticker = time.After(time.Second * 3)
			}
		}
	}()
	return nil
}

func (f *FanOutManager) close() {

}

func (f *FanOutManager) encodeGob(msg *core.Message) (res []byte, topic *string) {
	data := bytes.NewBuffer(nil) // bytes.Buffer
	enc := gob.NewEncoder(data)
	var err error
	if err = enc.Encode(msg); err == nil {
		//fmt.Println("Gob data:", data.Len())
		if f.config.DataCompressed {
			buffer := bytes.NewBuffer(nil)
			var writer *flate.Writer
			if writer, err = flate.NewWriter(buffer, flate.DefaultCompression); err == nil {
				var compressed int
				if compressed, err = writer.Write(data.Bytes()); err == nil {
					_ = compressed
					if err = writer.Flush(); err == nil {
						data = buffer
					}
				}
			}
		}
	}
	return data.Bytes(), nil
}

func (f *FanOutManager) encodePb(msg *core.Message) (res []byte, topic *string) {
	if msg.Type == core.MessageType_Quote {
		if quote, ok := msg.Payload.(core.QuoteMessage); ok {
			if quote.Type == core.QuoteDataType_Depth {
				// 深度订单记录 10 levels
				if depth, ok := quote.Data.(core.QuoteDepth); ok {
					info := message.DepthInfo{}
					info.Timestamp = depth.Time
					info.TransTs = depth.TransactionTime
					info.FirstUpdateId = depth.FirstUpdateID
					info.LastUpdateId = depth.LastUpdateID
					info.PrevLastUpdateId = depth.PrevLastUpdateID
					for _, bid := range depth.Bids {
						pl := &message.PriceLevel{}
						if v, err := strconv.ParseFloat(bid.Qty, 64); err == nil {
							pl.Amount = v
						}
						if v, err := strconv.ParseFloat(bid.Price, 64); err == nil {
							pl.Price = v
						}
						info.Bids = append(info.Bids, pl)
					}

					for _, ask := range depth.Asks {
						pl := &message.PriceLevel{}

						if v, err := strconv.ParseFloat(ask.Qty, 64); err == nil {
							pl.Amount = v
						}
						if v, err := strconv.ParseFloat(ask.Price, 64); err == nil {
							pl.Price = v
						}
						info.Asks = append(info.Asks, pl)
					}
					var err error
					if res, err = proto.Marshal(&info); err != nil {
						logger.Errorln("EncodePb Error: < message.QuoteDepth>", err.Error())
					} else {
						t := fmt.Sprintf("depth/%s", depth.Symbol)
						return res, &t
					}
				}

			} else if quote.Type == core.QuoteDataType_OrderIncrement {
				// 增量订单记录
				if inc, ok := quote.Data.(core.QuoteOrbIncr); ok {
					info := message.IncrementOrderBookInfo{}
					info.Timestamp = inc.Time
					info.TransTs = inc.TransactionTime
					info.FirstUpdateId = inc.FirstUpdateID
					info.LastUpdateId = inc.LastUpdateID
					info.PrevLastUpdateId = inc.PrevLastUpdateID
					for _, bid := range inc.Bids {
						pl := &message.PriceLevel{}
						if v, err := strconv.ParseFloat(bid.Qty, 64); err == nil {
							pl.Amount = v
						}
						if v, err := strconv.ParseFloat(bid.Price, 64); err == nil {
							pl.Price = v
						}
						info.Bids = append(info.Bids, pl)
					}

					for _, ask := range inc.Asks {
						pl := &message.PriceLevel{}

						if v, err := strconv.ParseFloat(ask.Qty, 64); err == nil {
							pl.Amount = v
						}
						if v, err := strconv.ParseFloat(ask.Price, 64); err == nil {
							pl.Price = v
						}
						info.Asks = append(info.Asks, pl)
					}
					var err error
					if res, err = proto.Marshal(&info); err != nil {
						logger.Errorln("EncodePb Error: < message.IncrementOrderBookInfo>", err.Error())
					} else {
						t := fmt.Sprintf("inc/%s", inc.Symbol)
						return res, &t
					}
				}
			} else if quote.Type == core.QuoteDataType_Trade {
				if trade, ok := quote.Data.(core.QuoteTrade); ok {
					//c.onQuoteTrade(&trade)
					info := message.TradeInfo{}
					info.Timestamp = trade.Time
					info.TradeTs = trade.TradeTime
					info.TradeId = trade.TradeID
					if v, err := strconv.ParseFloat(trade.Price, 64); err == nil {
						info.Price = v
					}
					if v, err := strconv.ParseFloat(trade.Quantity, 64); err == nil {
						info.Qty = v
					}
					info.BuyerOrderId = trade.BuyerOrderID
					info.SellerOrderId = trade.SellerOrderID
					info.IsBuyerMaker = trade.IsBuyerMaker
					info.PlaceHolder = trade.Placeholder

					var err error
					if res, err = proto.Marshal(&info); err != nil {
						logger.Errorln("EncodePb Error: < message.TradeInfo>", err.Error())
					} else {
						t := fmt.Sprintf("trade/%s", trade.Symbol)
						return res, &t
					}
				}
			} else if quote.Type == core.QuoteDataType_SnapShot {
				if snap, ok := quote.Data.(core.QuoteSnapShot); ok {
					//c.onQuoteSnapShot(&snap)
					info := message.SnapShotInfo{}
					info.LastUpdateId = snap.LastUpdateID
					info.Timestamp = snap.Time
					info.TradeTs = snap.TradeTime
					for _, bid := range snap.Bids {
						pl := &message.PriceLevel{}
						if v, err := strconv.ParseFloat(bid.Qty, 64); err == nil {
							pl.Amount = v
						}
						if v, err := strconv.ParseFloat(bid.Price, 64); err == nil {
							pl.Price = v
						}
						info.Bids = append(info.Bids, pl)
					}

					for _, ask := range snap.Asks {
						pl := &message.PriceLevel{}

						if v, err := strconv.ParseFloat(ask.Qty, 64); err == nil {
							pl.Amount = v
						}
						if v, err := strconv.ParseFloat(ask.Price, 64); err == nil {
							pl.Price = v
						}
						info.Asks = append(info.Asks, pl)
					}
					var err error
					if res, err = proto.Marshal(&info); err != nil {
						logger.Errorln("EncodePb Error: < message.SnapShotInfo>", err.Error())
					} else {
						t := fmt.Sprintf("snapshot/%s", snap.Symbol)
						return res, &t
					}
				}
			} else if quote.Type == core.QuoteDataType_Kline {
				if kline, ok := quote.Data.(core.QuoteKline); ok {
					info := message.KlineInfo{}
					info.Timestamp = kline.Time
					info.StartTime = kline.StartTime
					info.EndTime = kline.EndTime
					info.Symbol = kline.Symbol
					info.Interval = kline.Interval
					info.FirstTradeId = kline.FirstTradeID
					info.LastTradeId = kline.LastTradeID
					if v, err := strconv.ParseFloat(kline.Open, 64); err == nil {
						info.Open = v
					}
					if v, err := strconv.ParseFloat(kline.Close, 64); err == nil {
						info.Close = v
					}
					if v, err := strconv.ParseFloat(kline.High, 64); err == nil {
						info.High = v
					}
					if v, err := strconv.ParseFloat(kline.Low, 64); err == nil {
						info.Low = v
					}
					if v, err := strconv.ParseFloat(kline.Volume, 64); err == nil {
						info.Volume = v
					}
					info.TradeNum = kline.TradeNum
					info.IsFinal = kline.IsFinal

					if v, err := strconv.ParseFloat(kline.QuoteVolume, 64); err == nil {
						info.QuoteVolume = v
					}
					if v, err := strconv.ParseFloat(kline.ActiveBuyVolume, 64); err == nil {
						info.ActiveBuyVolume = v
					}
					if v, err := strconv.ParseFloat(kline.ActiveBuyQuoteVolume, 64); err == nil {
						info.ActiveBuyQuoteVolume = v
					}
					var err error
					if res, err = proto.Marshal(&info); err != nil {
						logger.Errorln("EncodePb Error: < message.QuoteKline>", err.Error())
					} else {
						t := fmt.Sprintf("kline/%s", kline.Symbol)
						return res, &t
					}
				}
			} else if quote.Type == core.QuoteDataType_AggTrade {
				if agg, ok := quote.Data.(core.QuoteAggTrade); ok {
					info := message.AggTradeInfo{}
					info.Timestamp = agg.Time
					info.Symbol = agg.Symbol
					info.AggregateTradeId = agg.AggregateTradeID
					if v, err := strconv.ParseFloat(agg.Price, 64); err == nil {
						info.Price = v
					}
					if v, err := strconv.ParseFloat(agg.Quantity, 64); err == nil {
						info.Quantity = v
					}
					info.FirstTradeId = agg.FirstTradeID
					info.LastTradeId = agg.LastTradeID
					info.TradeTime = agg.TradeTime
					info.Maker = agg.Maker

					var err error
					if res, err = proto.Marshal(&info); err != nil {
						logger.Errorln("EncodePb Error: < message.QuoteAggTrade>", err.Error())
					} else {
						t := fmt.Sprintf("aggtrade/%s", agg.Symbol)
						return res, &t
					}
				}
			}
		}
	}
	return
}

func (f *FanOutManager) fanout(msg *core.Message) (err error) {
	// zmq pub it
	var send_data []byte
	var topic *string

	if strings.ToUpper(controller.config.DataEncodeType) == "GOB" {
		send_data, topic = f.encodeGob(msg)
	}
	if strings.ToUpper(controller.config.DataEncodeType) == "PB" {
		send_data, topic = f.encodePb(msg)
	}
	if send_data == nil {
		return
	}
	if !f.dirty.Load() {
		fmt.Println(utils.FormatTimeYmdHms(time.Now()), "Data Publish:", len(send_data))
	}
	f.dirty.Store(true)
	//send_data = data.Bytes()
	flag := zmq.Flag(zmq.DONTWAIT)
	for _, dest := range f.producer_zmq {
		if topic != nil {
			flag = zmq.SNDMORE
			if _, err = dest.SendBytes([]byte(*topic), flag); err != nil {
				logger.Error("SendBytes <Topic> Error:", *topic, err.Error())
			}
		}
		flag = zmq.Flag(zmq.DONTWAIT)
		if _, err = dest.SendBytes(send_data, flag); err != nil {
			logger.Error("SendBytes Error:", err.Error(), len(send_data))
		}
	}

	return
}
