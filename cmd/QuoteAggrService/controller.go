package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	binance_connector "github.com/binance/binance-connector-go"
	"github.com/go-redis/redis/v8"
	"github.com/gookit/goutil/maputil"
	"github.com/gookit/goutil/netutil"
	"github.com/shopspring/decimal"

	"github.com/adoggie/ccsuites/core"
	"github.com/adoggie/ccsuites/message"
)

const (
	UntradablePositionVolume = math.MaxInt32
)

type Controller struct {
	posChanList map[string]chan *PositionEvent
	chanPos     chan *PositionEvent
	symbols     map[string]*Symbol

	periodSymbols map[string]*SymbolDef
	psMtx         sync.RWMutex

	//longPositions  map[string]Position
	//shortPositions map[string]Position
	//positions      map[string]float64

	config        *Config
	rdb           *redis.Client
	logChan       chan LogEvent
	mtxSelf       sync.RWMutex
	wsmtx         sync.RWMutex
	ctx           context.Context
	clientFuture  *futures.Client
	exchangeInfo  *futures.ExchangeInfo
	accountConfig *AccountConfig
	account       *futures.Account
	listenKey     string
	//positionReceiver *PositionReceiverPureGo
	//ordermgr         *OrderManager
	exchangeSymbols map[string]futures.Symbol // 交易所交易标的
	//tradingSymbols   map[string]futures.Symbol
	tradingSymbols map[string]*TradingSymbol
	wsapiClient    *binance_connector.WebsocketAPIClient
	//depthclientMgr     *WsDepthClientManager
	//tradeClientMgr     *WsTradeClientManager
	//orderbookClientMgr *WsOrbIncClientManager
	ratelimitMgr   *RateLimitManager
	fanoutMgr      *FanOutManager
	quoteDataCh    chan []byte
	quoteRecver    *QuoteReceiver
	periodCounter  int64
	wsserver       *WsServer
	qmon           *core.ServiceMonManager
	last_period_ts int64
}

func (c *Controller) GetExchangeInfo() *futures.ExchangeInfo {
	return c.exchangeInfo
}

func (c *Controller) GetConfig() *Config {
	return c.config
}

func (c *Controller) RedisDB() *redis.Client {
	return c.rdb
}

func (c *Controller) GetRateLimitManager() *RateLimitManager {
	return c.ratelimitMgr
}

func (c *Controller) init(config *Config) *Controller {
	c.periodCounter = 0
	//c.quoteDataCh = make(chan []byte, c.config.QuoteServeChanSize)
	c.quoteDataCh = make(chan []byte)
	c.posChanList = make(map[string]chan *PositionEvent)
	c.chanPos = make(chan *PositionEvent)

	c.symbols = make(map[string]*Symbol)
	//c.longPositions = make(map[string]Position)
	//c.shortPositions = make(map[string]Position)

	c.periodSymbols = make(map[string]*SymbolDef)

	c.config = config
	c.logChan = make(chan LogEvent, 1000)
	c.exchangeSymbols = make(map[string]futures.Symbol)
	c.tradingSymbols = make(map[string]*TradingSymbol)

	c.ctx = context.Background()
	//c.depthclientMgr = NewWsDepthClientManager(c)
	//c.tradeClientMgr = NewWsTradeClientManager(c)
	//c.orderbookClientMgr = NewWsOrbIncClientManager(c)

	c.fanoutMgr = NewFanOutManager(c.config)

	//c.ratelimitMgr = NewRateLimitManager(c.config)

	c.initRedis()
	//c.updatePositionFromRedis()
	c.initBinance()
	//c.initLeverage()
	//if c.config.PositionSubEnable {
	//	c.positionReceiver = NewPositionReceiverPureGo().Init(c.config, c.ctx)
	//}
	c.quoteRecver = NewQuoteReceiver()
	c.quoteRecver.Init(c.config)

	if c.config.WsServerConfig.Enable {
		c.wsserver = NewWsServer(&c.config.WsServerConfig)
	}
	if c.config.MonConfig.Enable {
		c.qmon = core.NewServiceMonManager(&c.config.MonConfig, logger)

		c.qmon.ServiceType(c.config.ServiceType)
		c.qmon.ServiceId(c.config.ThunderId)
		c.qmon.Version(VERSION)
		c.qmon.Ip(c.config.Ip)
		_ = c.qmon.Init()
	}
	return c
}

func (c *Controller) initLeverage() {
	// 杠杆设置
	for name, sym := range c.symbols {
		if _, err := c.clientFuture.NewChangeLeverageService().Symbol(name).Leverage(sym.Leverage).Do(c.ctx); err != nil {
			logger.Error("setLeverage Failed:", name, " leverage:", sym.Leverage)
			panic("setLeverage failed!")
		}
		logger.Info("setLeverage okay:", name, " leverage:", sym.Leverage, " ..")
	}
}

func (c *Controller) initRedis() bool {
	c.rdb = redis.NewClient(&redis.Options{
		Addr:     c.config.RedisServer.Addr,
		Password: c.config.RedisServer.Password,
		DB:       c.config.RedisServer.DB,
	})
	return true
}

func (c *Controller) onAggTradeEvent(event *futures.WsAggTradeEvent) {
	if sym, ok := c.symbols[event.Symbol]; ok {
		//fmt.Println(event.Time, event.Symbol, event.Price, event.Quantity)
		sym.mtx.Lock()
		//sym.impl = event
		sym.aggTrade = event
		sym.mtx.Unlock()

	}

}

// 1. account prepare
// 2. load symbols
// 3. user stream set

func (c *Controller) GetOrNewWsClient() *binance_connector.WebsocketAPIClient {
	c.wsmtx.Lock()
	defer c.wsmtx.Unlock()

	if c.wsapiClient == nil {
		for {
			wsapiClient := binance_connector.NewWebsocketAPIClient(c.accountConfig.ApiKey, c.accountConfig.SecretKey, GetFutureNetUrl(c.accountConfig.TestNet))
			if err := wsapiClient.Connect(); err != nil {
				logger.Errorln("WsApiClient UnConnected. Retry ..")
				time.Sleep(time.Second * 2)
				continue
			} else {
				c.wsapiClient = wsapiClient
				logger.Infoln("GetOrNewWsClient  Websocket API client connected!")
				break
			}
		}
	}

	return c.wsapiClient
}

func (c *Controller) ResetWsClient() {
	c.wsmtx.Lock()
	defer c.wsmtx.Unlock()
	if c.wsapiClient != nil {
		c.wsapiClient.Close()
	}
}

func (c *Controller) initBinance() {
	futures.WebsocketKeepalive = true

	for _, acc := range c.config.Accounts {
		if acc.Name == c.config.Account {
			c.accountConfig = acc
		}
	}
	if c.accountConfig == nil {
		panic("Please check Account!")
	}

	logger.Infoln("Current Account: ", c.accountConfig.Name)

	if c.accountConfig.TestNet {
		futures.UseTestnet = true
		logger.Infoln("It's RUNNING ON TestNet ..")
	}
	c.clientFuture = binance.NewFuturesClient(c.accountConfig.ApiKey, c.accountConfig.SecretKey)

	c.updateExchangeInfo()
	c.loadTradingSymbol()
	c.updateAggrSymbols()

	_ = c.fanoutMgr.init()

	c.GetOrNewWsClient()

	logger.Infoln("InitBinance okay.")
}

func (c *Controller) updateAggrSymbols() {
	c.psMtx.Lock()
	defer c.psMtx.Unlock()
	c.periodSymbols = make(map[string]*SymbolDef)
	for name, _ := range c.tradingSymbols {
		sym := NewSymbolDef(name)
		c.periodSymbols[name] = sym
	}

	ioutil.WriteFile(fmt.Sprintf("%s_aggr_symbols.json", c.config.ThunderId), []byte(binance_connector.PrettyPrint(maputil.Keys(c.periodSymbols))), 0644)
}

func (c *Controller) loadTradingSymbol() {
	c.tradingSymbols = make(map[string]*TradingSymbol)

	if len(c.config.TradingSymbols) != 0 {
		for _, name := range c.config.TradingSymbols {
			if s, ok := c.exchangeSymbols[name]; ok {
				if s.Status == string(futures.SymbolStatusTypeTrading) {
					c.tradingSymbols[name] = NewTradingSymbol(s)
					logger.Traceln("Trading Symobl:", name)
				}
			}
		}
	} else {
		for name, symbol := range c.exchangeSymbols {
			if symbol.Status == string(futures.SymbolStatusTypeTrading) {
				c.tradingSymbols[name] = NewTradingSymbol(symbol)
			}
		}
	}

}

func (c *Controller) GetTradingSymbolInfo(name string) *TradingSymbol {
	if symbol, ok := c.tradingSymbols[name]; ok {
		return symbol
	}
	return nil
}

func (c *Controller) updateExchangeInfo() {
	// 交易所信息
	c.exchangeSymbols = make(map[string]futures.Symbol)
	for {
		if exinfo, err := c.clientFuture.NewExchangeInfoService().Do(c.ctx); err == nil {
			c.exchangeInfo = exinfo
			for _, syn := range exinfo.Symbols {
				//fmt.Println("symbol:",syn.Symbol , " quotePrecision:" ,syn.QuotePrecision )
				c.exchangeSymbols[syn.Symbol] = syn
			}
			logger.Infoln("Total Symbols :", len(exinfo.Symbols))
			break
		} else {
			logger.Warn("GetExchangeInfo failed, wait and retry ..", err.Error())
			time.Sleep(time.Second * 2)
		}
	}
	//c.ratelimitMgr.UpdateRateLimitManager(c.exchangeInfo)

}

func (c *Controller) open() {

	c.run()
}

func (c *Controller) newServiceMessage(type_ string) *ServiceMessage {
	message := &ServiceMessage{Type: type_}
	message.Source.Type = c.config.ServiceType
	message.Source.Id = c.config.ThunderId
	message.Source.Ip = netutil.InternalIP()
	message.Source.Datetime = time.Now().Format(time.RFC3339)
	message.Source.Timestamp = time.Now().Unix()
	message.Content = make(map[string]interface{})
	//message.Delta.UpdateKeys = make(map[string]interface{})
	return message
}

func (m *ServiceMessage) SetValue(key string, value interface{}) *ServiceMessage {
	m.Content[key] = value
	return m
}

func (m *ServiceMessage) Database(dbname string) *ServiceMessage {
	m.Delta.DB = dbname
	return m
}

func (m *ServiceMessage) Table(name string) *ServiceMessage {
	m.Delta.Table = name
	return m
}

func (m *ServiceMessage) UpdateKeys(keys ...string) *ServiceMessage {
	m.Delta.UpdateKeys = keys
	return m
}

// 投递到redis-pub
func (c *Controller) sendMessage(m *ServiceMessage) {
	if detail, err := json.Marshal(m); err == nil {
		c.rdb.Publish(c.ctx, c.config.MessagePubChan, detail)
	} else {
		fmt.Println(err.Error())
		logger.Error(err.Error())
	}
}

// Balance和Position更新推送
func (c *Controller) onUserData(event *futures.WsUserDataEvent) {
	//fmt.Println("onUserData: ", event.Event)
	////fmt.Println(binance_connector.PrettyPrint(event))
	//if event.Event == futures.UserDataEventTypeAccountUpdate {
	//	for _, pos := range event.AccountUpdate.Positions {
	//		// wsPosition
	//		amt, _ := decimal.NewFromString(pos.Amount)
	//		ap := &AccountPosition{wspos: &pos, Symbol: pos.Symbol, PositionSide: pos.Side, PositionAmt: amt}
	//		c.ordermgr.OnAccountPositionUpdate(ap)
	//		logger.Traceln("WsUserData", pos.Symbol, "Pos update:", amt)
	//	}
	//} else if event.Event == futures.UserDataEventTypeOrderTradeUpdate {
	//	logger.Traceln("UserDataEventTypeOrderTradeUpdate..")
	//	//logger.Traceln(binance_connector.PrettyPrint(event.OrderTradeUpdate))
	//
	//	c.ordermgr.OnOrderUpdate(&event.OrderTradeUpdate)
	//} else if event.Event == futures.UserDataEventTypeAccountConfigUpdate {
	//
	//}
}

func (c *Controller) subUserStreamData() {
START:
	var (
		doneC, stopC chan struct{}
		err          error
	)
	for {
		doneC, stopC, err = futures.WsUserDataServe(c.listenKey, c.onUserData, func(err error) {})
		if err != nil {
			logger.Warn("WsUserDataServe Error , wait and retry..", err.Error())
			time.Sleep(time.Second * 5)
		}
		break
	}
	logger.Debugln("WsUserData Connection Established!")
	for {
		select {
		case <-c.ctx.Done():
			stopC <- struct{}{} //系统终止，准备全部退出
			return
		case <-doneC:
			// ws 断开需要重新连接
			logger.Warn("WsUserData Connection lost , Wait and retry..")
			time.Sleep(time.Second * 5)
			goto START
		case <-time.After(time.Second * time.Duration(c.config.UserStreamKeepInterval)):
			logger.Info("UserStream Keep ..")
			if err := c.clientFuture.NewKeepaliveUserStreamService().Do(c.ctx); err != nil {

			}
			//case q := <-c.quoteDataCh:
			//	logger.Traceln("QuoteData Incoming..", len(q))
		}
	}
}

func (c *Controller) quoteDataRecvHandler() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case q := <-c.quoteDataCh:
			//logger.Traceln("QuoteData Incoming..", len(q))
			c.onQuoteDataRecv(q)
		}
	}
}

const ORIGIN_UTC_TIME = 1704067200 // seconds

func (c *Controller) timedPeriodCollect() {
	c.last_period_ts = time.Now().UTC().Unix()
	period_size := PERIOD
	logger.Infoln("timedPeriodCollect Time Interval:", c.config.TimeIterateInterval)
	for {
		select {
		case <-time.After(time.Millisecond * time.Duration(c.config.TimeIterateInterval)):
			pm := &message.PeriodMessage{}

			pm.SymbolInfos = make([]*message.SymbolInfo, 0)

			c.psMtx.RLock()

			now := time.Now().UTC()
			//bornTs := now.Unix() * 1000

			dist := (time.Now().UTC().Unix() - ORIGIN_UTC_TIME) * 1000
			period_now := dist / period_size
			bornTs := period_now*period_size + ORIGIN_UTC_TIME*1000
			//fmt.Println(period_now, bornTs, period_size)
			//在好友中找一个当前Period的出生时间
			if len(c.periodSymbols) > 0 {
				for _, symbol := range c.periodSymbols {
					if symbol.linked.Len() != 0 {
						pn := symbol.linked.Front().Value.(*PeriodNode)
						bornTs = pn.start
						break
					}
				}
			}

			var pn *PeriodNode
			for name, symbol := range c.periodSymbols {
				_ = name
				sinfo := &message.SymbolInfo{}
				sinfo.Symbol = name
				if pn = symbol.timeItr(now, bornTs); pn != nil {
					pm.Ts = pn.start // 2024.08.20
					//sinfo.Trades, sinfo.Incs = pn.MarshallPb(sinfo)
					pn.MarshallPb(sinfo)
					if c.config.SkipEmptyQuote {
						if len(sinfo.Trades) == 0 && len(sinfo.Incs) == 0 && len(sinfo.Snaps) == 0 {
							continue
						}
					}

					pm.SymbolInfos = append(pm.SymbolInfos, sinfo)
				} else {
					//pm = nil
					_ = pm
				}

			}
			c.psMtx.RUnlock()

			if pm != nil {
				//if c.config.SkipEmptyQuote {
				if pm.SymbolInfos == nil || len(pm.SymbolInfos) == 0 {
					//logger.Traceln("SkipEmptyQuote..")
					if c.config.MonConfig.Enable {
						if time.Now().UTC().Unix()-c.last_period_ts > 20 {
							c.qmon.SendLog("WARN", "PeriodCollect", "EmptyQuote", "EmptyQuote", 5)
						}

					}
					continue
				} else {
					c.last_period_ts = time.Now().UTC().Unix()
				}
				//}

				pm.Period = period_now - 1 // 越过一个周期 触发的detach操作 ，故 period = period -1
				//pm.Period = c.periodCounter
				//c.periodCounter = c.periodCounter + 1

				pm.PosterId = c.config.ThunderId
				//pm.Ts = time.Now().UnixMilli()
				//pm.Ts = time.Now().UTC().Unix() * 1000
				pm.PostTs = time.Now().UTC().UnixMilli()

				_ = c.fanoutMgr.send(pm)
				if c.wsserver != nil {
					c.wsserver.enqueue(pm)
				}
			}

		}
	}
}

func (c *Controller) GetTradeSymbols() []string {
	return maputil.Keys(c.tradingSymbols)

	//return []string{"ltcusdt", "btcusdt"}
}

func (c *Controller) run() {
	stop := make(chan struct{})
	_ = c.fanoutMgr.open(c.ctx)

	go c.quoteDataRecvHandler()
	_ = c.quoteRecver.Open(c.ctx, stop)

	if c.wsserver != nil {
		go c.wsserver.open(c.ctx, stop)
	}

	go c.timedPeriodCollect()
	if c.config.MonConfig.Enable {
		go c.qmon.Open(c.ctx)
	}

	//timer := time.After(time.Hour * time.Duration(c.config.UpdateTradingSymbolInterval))
	for {
		select {
		case <-c.ctx.Done():
			return
		//case <-timer:
		//	logger.Infoln("Start UpdateTradingSymbolInterval..")
		//	c.updateExchangeInfo()
		//	c.loadTradingSymbol()
		//	c.updateAggrSymbols()
		//	logger.Infoln("Finished UpdateTradingSymbolInterval.")
		//	timer = time.After(time.Hour * time.Duration(c.config.UpdateTradingSymbolInterval))
		case event := <-c.chanPos:
			if _, ok := c.tradingSymbols[event.symbol]; !ok {
				break
			} else {

			}
		}
	}

}

func (c *Controller) discardPendingOrders() {
	if !c.config.CancelOrderOnStart {
		return
	}
	// 撤销所有挂单
	logger.Info("discardAllOrders()..")
	//for symbol, _ := range c.tradingSymbols {
	orders, _ := c.clientFuture.NewListOpenOrdersService().Do(c.ctx)
	var orderIds []int64
	var symbols []string
	orderIds = Map(orders, func(from *futures.Order) int64 {
		return from.OrderID
	})
	_ = orderIds
	symbols = Map(orders, func(from *futures.Order) string {
		return from.Symbol
	})

	if len(symbols) != 0 {
		for _, symbol := range symbols {
			logger.Infoln("cancelOrder: ", symbols, " orderIds:", orderIds)
			if err := c.clientFuture.NewCancelAllOpenOrdersService().Symbol(symbol).Do(c.ctx); err != nil {
				logger.Traceln(binance_connector.PrettyPrint(err.Error()))

			}
		}
	}
}

func (c *Controller) updatePositionFromRedis() {
	logger.Info("load positions from redis..")
	redis := c.rdb
	ctx := context.Background()
	var (
		val string
		err error
	)
	for name, _ := range c.symbols {
		if val, err = redis.HGet(ctx, c.config.ThunderId, name).Result(); err != nil {
			continue
		}

		if pos, e := decimal.NewFromString(val); e == nil {
			fmt.Println(pos.String())
			if sym, ok := c.symbols[name]; ok {
				sym.target = &pos
				logger.Info("load position:", name, pos)
			}
		} else {
			//fmt.Print(e.Error())
			logger.Error(e.Error())
		}
	}
}

func (c *Controller) onQuoteDataRecv(data []byte) {
	var buffer = bytes.NewBuffer(data)
	dec := gob.NewDecoder(buffer)
	{
		var msg core.Message
		if err := dec.Decode(&msg); err != nil {
			fmt.Println(err.Error())
		} else {
			if msg.Type == core.MessageType_Quote {
				if quote, ok := msg.Payload.(core.QuoteMessage); ok {
					if quote.Type == core.QuoteDataType_Depth {
						if depth, ok := quote.Data.(core.QuoteDepth); ok {
							//fmt.Println("", depth)
							c.onQuoteDepth(&depth)
						}
					} else if quote.Type == core.QuoteDataType_OrderIncrement {
						if inc, ok := quote.Data.(core.QuoteOrbIncr); ok {
							c.onQuoteOrderbookInc(&inc)
							//_ = inc
						}
					} else if quote.Type == core.QuoteDataType_Trade {
						if trade, ok := quote.Data.(core.QuoteTrade); ok {
							c.onQuoteTrade(&trade)
						}
					} else if quote.Type == core.QuoteDataType_SnapShot {
						if snap, ok := quote.Data.(core.QuoteSnapShot); ok {
							c.onQuoteSnapShot(&snap)
						}
					}

				}

			}
		}
	}
}

func (c *Controller) onQuoteDepth(depth *core.QuoteDepth) error {
	return nil
}

func (c *Controller) onQuoteOrderbookInc(inc *core.QuoteOrbIncr) error {
	c.psMtx.RLock()
	//defer c.psMtx.RUnlock()
	if symbol, ok := c.periodSymbols[inc.Symbol]; ok {
		c.psMtx.RUnlock()
		symbol.put_inc(inc)
	} else {
		c.psMtx.RUnlock()
	}
	return nil
}

func (c *Controller) onQuoteTrade(trade *core.QuoteTrade) error {
	c.psMtx.RLock()
	//defer c.psMtx.RUnlock()
	if symbol, ok := c.periodSymbols[trade.Symbol]; ok {
		c.psMtx.RUnlock()
		symbol.put_trade(trade)
	} else {
		c.psMtx.RUnlock()
	}
	return nil
}

func (c *Controller) onQuoteSnapShot(snap *core.QuoteSnapShot) error {
	c.psMtx.RLock()
	if symbol, ok := c.periodSymbols[snap.Symbol]; ok {
		c.psMtx.RUnlock()
		symbol.put_snapshot(snap)
	} else {
		c.psMtx.RUnlock()
	}
	return nil
}
