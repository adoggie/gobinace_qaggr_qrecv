package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/adoggie/ccsuites/core"
	binance_connector "github.com/binance/binance-connector-go"
	"github.com/gookit/goutil/maputil"
	"math"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/go-redis/redis/v8"
	"github.com/gookit/goutil/netutil"
	"github.com/shopspring/decimal"
)

const (
	UntradablePositionVolume = math.MaxInt32
)

type Controller struct {
	posChanList    map[string]chan *PositionEvent
	chanPos        chan *PositionEvent
	symbols        map[string]*Symbol
	longPositions  map[string]Position
	shortPositions map[string]Position
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
	tradingSymbols     map[string]*TradingSymbol
	wsapiClient        *binance_connector.WebsocketAPIClient
	depthclientMgr     *WsDepthClientManager
	tradeClientMgr     *WsTradeClientManager
	orderbookClientMgr *WsOrbIncClientManager
	ratelimitMgr       *RateLimitManager
	fanoutMgr          *FanOutManager
	snapshotMgr        *SnapShotManager
	qmon               *core.ServiceMonManager

	klineClientMgr    *WsKLineManager
	aggtradeClientMgr *WsAggTradeManager
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
	c.posChanList = make(map[string]chan *PositionEvent)
	c.chanPos = make(chan *PositionEvent)

	c.symbols = make(map[string]*Symbol)
	c.longPositions = make(map[string]Position)
	c.shortPositions = make(map[string]Position)

	c.config = config
	c.logChan = make(chan LogEvent, 1000)
	c.exchangeSymbols = make(map[string]futures.Symbol)
	c.tradingSymbols = make(map[string]*TradingSymbol)

	c.ctx = context.Background()
	c.depthclientMgr = NewWsDepthClientManager(c)
	c.tradeClientMgr = NewWsTradeClientManager(c)
	c.orderbookClientMgr = NewWsOrbIncClientManager(c)
	c.klineClientMgr = NewWsKLineManager(c)
	c.aggtradeClientMgr = NewWsAggTradeManager(c)

	c.fanoutMgr = NewFanOutManager(c.config)
	if c.config.RecvSnapShotEnable {
		c.snapshotMgr = NewSnapShotManager(c)
	}
	//c.ratelimitMgr = NewRateLimitManager(c.config)

	c.initRedis()
	//c.updatePositionFromRedis()
	c.initBinance()
	//c.initLeverage()
	//if c.config.PositionSubEnable {
	//	c.positionReceiver = NewPositionReceiverPureGo().Init(c.config, c.ctx)
	//}

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

func (c *Controller) OnWsAggTradeEvent(event *futures.WsAggTradeEvent) {
	trade := core.QuoteAggTrade{}
	trade.Time = event.Time
	trade.Symbol = event.Symbol
	trade.AggregateTradeID = event.AggregateTradeID
	trade.Price = event.Price
	trade.Quantity = event.Quantity
	trade.FirstTradeID = event.FirstTradeID
	trade.LastTradeID = event.LastTradeID
	trade.TradeTime = event.TradeTime
	trade.Maker = event.Maker
	msg := core.Message{Type: core.MessageType_Quote,
		Payload: core.QuoteMessage{
			Type:     core.QuoteDataType_AggTrade,
			Exchange: core.ExchangeType_Binance,
			Data:     trade}}
	c.fanoutMgr.msgCh <- &msg
}

// 深度行情返回
func (c *Controller) OnWsDepthEvent(event *futures.WsDepthEvent) {
	depth := core.QuoteDepth{

		Time:             event.Time,
		TransactionTime:  event.TransactionTime,
		Symbol:           event.Symbol,
		FirstUpdateID:    event.FirstUpdateID,
		LastUpdateID:     event.LastUpdateID,
		PrevLastUpdateID: event.PrevLastUpdateID,
		Bids:             make([]core.BidPrice, 0),
		Asks:             make([]core.AskPrice, 0),
	}
	for _, bid := range event.Bids {
		depth.Bids = append(depth.Bids, core.BidPrice{Price: bid.Price, Qty: bid.Quantity})
	}
	for _, ask := range event.Asks {
		depth.Asks = append(depth.Asks, core.AskPrice{Price: ask.Price, Qty: ask.Quantity})
	}
	//now := time.Now().UTC().UnixMilli()
	//fmt.Println(event.Symbol, now, event.Time, event.TransactionTime, "bids:", len(depth.Bids), " asks:", len(depth.Asks))
	msg := core.Message{Type: core.MessageType_Quote,
		Payload: core.QuoteMessage{
			Type:     core.QuoteDataType_Depth,
			Exchange: core.ExchangeType_Binance,
			Data:     depth}}
	_ = msg
	c.fanoutMgr.msgCh <- &msg
}

func (c *Controller) OnWsKLineEvent(event *futures.WsKlineEvent) {
	//fmt.Println("OnWsKLineEvent:", event.Symbol, event.Kline.StartTime, event.Kline.EndTime, event.Kline.Open, event.Kline.Close, event.Kline.High, event.Kline.Low, event.Kline.Volume)
	kline := core.QuoteKline{
		Time:                 event.Time,
		Symbol:               event.Symbol,
		StartTime:            event.Kline.StartTime,
		EndTime:              event.Kline.EndTime,
		Interval:             event.Kline.Interval,
		FirstTradeID:         event.Kline.FirstTradeID,
		LastTradeID:          event.Kline.LastTradeID,
		Open:                 event.Kline.Open,
		Close:                event.Kline.Close,
		High:                 event.Kline.High,
		Low:                  event.Kline.Low,
		Volume:               event.Kline.Volume,
		TradeNum:             event.Kline.TradeNum,
		IsFinal:              event.Kline.IsFinal,
		QuoteVolume:          event.Kline.QuoteVolume,
		ActiveBuyVolume:      event.Kline.ActiveBuyVolume,
		ActiveBuyQuoteVolume: event.Kline.ActiveBuyQuoteVolume,
	}
	msg := core.Message{Type: core.MessageType_Quote,
		Payload: core.QuoteMessage{
			Type:     core.QuoteDataType_Kline,
			Exchange: core.ExchangeType_Binance,
			Data:     kline}}
	c.fanoutMgr.msgCh <- &msg
}

func (c *Controller) OnWsOrbIncDepthEvent(event *futures.WsDepthEvent) {
	//c.ordermgr.OnDepthEvent(event)
	depth := core.QuoteOrbIncr{

		Time:             event.Time,
		TransactionTime:  event.TransactionTime,
		Symbol:           event.Symbol,
		FirstUpdateID:    event.FirstUpdateID,
		LastUpdateID:     event.LastUpdateID,
		PrevLastUpdateID: event.PrevLastUpdateID,
		Bids:             make([]core.BidPrice, 0),
		Asks:             make([]core.AskPrice, 0),
	}
	for _, bid := range event.Bids {
		depth.Bids = append(depth.Bids, core.BidPrice{Price: bid.Price, Qty: bid.Quantity})
	}
	for _, ask := range event.Asks {
		depth.Asks = append(depth.Asks, core.AskPrice{Price: ask.Price, Qty: ask.Quantity})
	}
	//now := time.Now().UTC().UnixMilli()
	//fmt.Println(event.Symbol, now, event.Time, event.TransactionTime, "bids:", len(depth.Bids), " asks:", len(depth.Asks))
	msg := core.Message{Type: core.MessageType_Quote,
		Payload: core.QuoteMessage{
			Type:     core.QuoteDataType_OrderIncrement,
			Exchange: core.ExchangeType_Binance,
			Data:     depth}}
	_ = msg
	c.fanoutMgr.msgCh <- &msg
	//_ = c.fanoutMgr.fanout(&msg)
}

func (c *Controller) OnWsTradeEvent(event *futures.WsTradeEvent) {
	//fmt.Println("OnWsTradeEvent:", event.Symbol, event.Price, event.Quantity)
	trade := core.QuoteTrade{
		Event:         event.Event,
		Time:          event.Time,
		Symbol:        event.Symbol,
		TradeID:       event.TradeID,
		Price:         event.Price,
		Quantity:      event.Quantity,
		BuyerOrderID:  event.BuyerOrderID,
		SellerOrderID: event.SellerOrderID,
		TradeTime:     event.TradeTime,
		IsBuyerMaker:  event.IsBuyerMaker,
		Placeholder:   event.Placeholder,
	}
	msg := core.Message{Type: core.MessageType_Quote,
		Payload: core.QuoteMessage{
			Type:     core.QuoteDataType_Trade,
			Exchange: core.ExchangeType_Binance,
			Data:     trade}}
	c.fanoutMgr.msgCh <- &msg
	//_ = c.fanoutMgr.fanout(&msg)

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
		panic("Please Corrrect Account!")
	}

	logger.Infoln("Current Account: ", c.accountConfig.Name)

	if c.accountConfig.TestNet {
		futures.UseTestnet = true
		logger.Infoln("It's RUNNING ON TestNet ..")
	}
	c.clientFuture = binance.NewFuturesClient(c.accountConfig.ApiKey, c.accountConfig.SecretKey)

	c.updateExchangeInfo()
	c.loadTradingSymbol()
	// 报单管理对象注册
	//for name, _ := range c.tradingSymbols {
	//	_ = c.ordermgr.AddSymbol(name)
	//}
	// 账户余额
	//for {
	//	if acc, e := c.clientFuture.NewGetAccountService().Do(c.ctx); e == nil {
	//		c.account = acc
	//		fmt.Println("account balance:", acc.AvailableBalance) // 可用余额
	//		for _, pos := range acc.Positions {
	//			amt, _ := decimal.NewFromString(pos.PositionAmt)
	//			ap := &AccountPosition{acpos: pos, Symbol: pos.Symbol, PositionSide: pos.PositionSide, PositionAmt: amt}
	//			c.ordermgr.OnAccountPositionUpdate(ap)
	//		}
	//		break
	//	} else {
	//		logger.Warn("GetAccountService failed, wait and retry . error:", e.Error())
	//		time.Sleep(time.Second * 2)
	//	}
	//}

	// User Stream
	//for {
	//	if listenKey, e := c.clientFuture.NewStartUserStreamService().Do(c.ctx); e == nil {
	//		c.listenKey = listenKey
	//		logger.Infoln("UserStreamListenKey:", listenKey)
	//		break
	//	} else {
	//		fmt.Println(e.Error())
	//		logger.Warnln("GetUserStreamListenKey failed. Retry ..")
	//		time.Sleep(time.Second * 2)
	//	}
	//}

	// market quote ready
	if c.config.RecvDepthEnable {
		c.depthclientMgr.Init(&c.config.DepthClientManagerConfig)
	}
	if c.config.RecvTradeEnable {
		c.tradeClientMgr.Init(&c.config.TradeClientManagerConfig)
	}
	if c.config.RecvOrbIncEnable {
		c.orderbookClientMgr.Init(&c.config.OrderbookIncClientManager)
	}

	if c.config.RecvKlineEnable {
		c.klineClientMgr.Init(&c.config.KlineClientManagerConfig)
	}

	if c.config.RecvAggTradeEnable {
		c.aggtradeClientMgr.Init(&c.config.AggTradeClientManagerConfig)
	}

	c.fanoutMgr.init()

	//c.depthclientMgr.Open()
	c.GetOrNewWsClient()

	logger.Infoln("InitBinance okay.")
}

func (c *Controller) loadTradingSymbol() {
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

	for {
		if exinfo, err := c.clientFuture.NewExchangeInfoService().Do(c.ctx); err == nil {
			c.exchangeInfo = exinfo
			for _, syn := range exinfo.Symbols {
				//fmt.Println("symbol:",syn.Symbol , " quotePrecision:" ,syn.QuotePrecision )
				c.exchangeSymbols[syn.Symbol] = syn
			}
			fmt.Println("Total Symbols :", len(exinfo.Symbols))
			break
		} else {
			fmt.Println(err.Error())
			logger.Warn("GetExchangeInfo failed, wait and retry ..")
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
		}
	}
}

func (c *Controller) GetTradeSymbols() []string {
	return maputil.Keys(c.tradingSymbols)

	//return []string{"ltcusdt", "btcusdt"}
}

func (c *Controller) run() {

	c.fanoutMgr.open(c.ctx)

	if c.config.RecvTradeEnable {
		c.tradeClientMgr.Open()
	}
	if c.config.RecvDepthEnable {
		c.depthclientMgr.Open()
	}
	if c.config.RecvOrbIncEnable {
		c.orderbookClientMgr.Open()
	}

	if c.config.RecvKlineEnable {
		c.klineClientMgr.Open()
	}

	if c.config.RecvAggTradeEnable {
		c.aggtradeClientMgr.Open()
	}

	if c.snapshotMgr != nil {
		_ = c.snapshotMgr.Open(c.ctx)
	}

	//c.discardPendingOrders()
	//c.queryAccountAndPosition(true)
	// 连续订阅 userstream
	//go c.subUserStreamData()
	//go c.ratelimitMgr.Open()

	// 定时查询账户信息（包括资金和仓位情况)
	//go c.queryAccountAndPosition(false)

	if c.config.MonConfig.Enable {
		go c.qmon.Open(c.ctx)
	}

	for {
		select {
		case <-c.ctx.Done():
			return
			//case event := <-c.chanPos:
			//	if _, ok := c.tradingSymbols[event.symbol]; !ok {
			//		break
			//	} else {
			//
			//		//c.ordermgr.OnStrategyPositionRecv(event)
			//	}
			//
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

	//}
}

// 仓位信号到达
func (c *Controller) onRecvPosition(event *PositionEvent) {

	if !c.GetConfig().PositionSubForward {
		logger.Errorln("Position Forward Disabled.")
		return
	}
	c.chanPos <- event

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
