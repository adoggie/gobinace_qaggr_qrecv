package main

// market_kline.go
//https://developers.binance.com/docs/zh-CN/derivatives/usds-margined-futures/websocket-market-streams/Continuous-Contract-Kline-Candlestick-Streams

import (
	"github.com/adshao/go-binance/v2/futures"
	"strconv"
	"sync/atomic"
	"time"
)

type WsKLineMgrConfigVars struct {
	MaxSymbolNumPerConn int    `toml:"max_symbol_num_per_conn"`
	WaitTimeReconnect   int    `toml:"reconnect_wait_time"`
	Interval            string `toml:"interval"`
}

type WsKLineService struct {
	symbols             []string
	doneC, stopC        chan struct{}
	err                 error
	mgr                 *WsKLineManager
	updates             map[string]interface{}
	id                  string
	lastest_update_time atomic.Int64
}

func (ds *WsKLineService) reset() {
	if ds.stopC != nil {
		ds.stopC <- struct{}{}
	}

	ds.doneC = nil
	ds.stopC = nil
	ds.lastest_update_time.Store(0)
}

// 接收到盘口深度价格
func (ds *WsKLineService) handler(event *futures.WsKlineEvent) {
	//logger.Traceln("WsDepthService.handler:", binance_connector.PrettyPrint(event))
	ds.lastest_update_time.Store(time.Now().Unix())
	ds.updates[event.Symbol] = nil
	//fmt.Println(time.Now().Format(time.DateTime), "market_depth.go:", "depthservice:", ds.id, " updates:", len(ds.updates))
	//fmt.Println(event.Symbol, event.Asks[0], event.Bids[0])
	ds.mgr.C.OnWsKLineEvent(event)
}

// 出错了，需要重新链接
func (ds *WsKLineService) error(err error) {
	//ds.doneC <- struct{}{}
	//time.Sleep(time.Second * time.Duration(ds.mgr.config.WaitTimeReconnect))
	//go ds.run()

	logger.Errorln("WsKLineService: WebSocket Lost ..", ds.id, " Error:", err.Error())
	//ds.doneC <- struct{}{}
	ds.reset()
	time.Sleep(time.Second * time.Duration(ds.mgr.config.WaitTimeReconnect))
}

func (ds *WsKLineService) run() {
	//levels := strconv.FormatInt(10, 10)

	ds.updates = make(map[string]interface{})

	timer := time.After(time.Second * 1)
	for {
		select {
		case <-timer:
			if time.Now().Unix()-ds.lastest_update_time.Load() > int64(controller.config.RecvOrbIncReconnectTimeout) {
				logger.Infoln("WsKLineService:", ds.id, " Timeout, Connect")
				ds.reset()
				ds.doConnect()
			}
			timer = time.After(time.Second * 1)
		}
	}
}

func (ds *WsKLineService) doConnect() {

	//https: //developers.binance.com/docs/zh-CN/derivatives/usds-margined-futures/websocket-market-streams/Kline-Candlestick-Streams
	keyValueMap := make(map[string]string)
	for _, key := range ds.symbols {
		keyValueMap[key] = ds.mgr.config.Interval
	}

	for {
		logger.Infoln("WsKLineService:", ds.id, " Start Connect ..")
		//ds.doneC, ds.stopC, ds.err = futures.WsCombinedDiffDepthServe(ds.symbols, ds.handler, ds.error)
		//if ds.err != nil {
		//	logger.Errorln("WsOrbIncService Error:", ds.err.Error(), " Sleep awhile..")
		//	time.Sleep(time.Second * time.Duration(ds.mgr.config.WaitTimeReconnect))
		//}
		//ds.doneC, ds.stopC, ds.err = futures.WsCombinedDepthServe(keyValueMap, ds.handler, ds.error)
		ds.doneC, ds.stopC, ds.err = futures.WsCombinedKlineServe(keyValueMap, ds.handler, ds.error)
		if ds.err != nil {
			logger.Errorln("WsKLineService Error:", ds.err.Error())
			time.Sleep(time.Second * time.Duration(ds.mgr.config.WaitTimeReconnect))
			continue
		}

		break
	}
	logger.Infoln("WsKLineService:", ds.id, " Connect Success..")

	ds.lastest_update_time.Store(time.Now().Unix())
}

type WsKLineManager struct {
	C                    *Controller
	depthServices        []*WsKLineService
	symbolInDepthService map[string]*WsKLineService
	config               *WsKLineMgrConfigVars
}

func NewWsKLineManager(controller *Controller) *WsKLineManager {
	wsClientMgr := &WsKLineManager{C: controller}
	return wsClientMgr
}

//func (mgr *WsDepthClientManager) GetApiClient(symbol string) *binance_connector.WebsocketAPIClient {
//	return nil
//}
//
//func (mgr *WsDepthClientManager) GetStreamClient(symbol string) *binance_connector.WebsocketStreamClient {
//	return nil
//}

func (mgr *WsKLineManager) Config() *WsKLineMgrConfigVars {
	return mgr.config
}

func (mgr *WsKLineManager) Init(config *WsKLineMgrConfigVars) *WsKLineManager {
	mgr.config = config
	mgr.depthServices = make([]*WsKLineService, 0)
	mgr.symbolInDepthService = make(map[string]*WsKLineService)
	return mgr
}

// Open() ws通道分配交易对象
func (mgr *WsKLineManager) Open() error {
	var symbols []string
	symbols = mgr.C.GetTradeSymbols()
	symarr_list := splitArrayBySliceSize(symbols, mgr.config.MaxSymbolNumPerConn)
	for n, symarr := range symarr_list {
		//logger.Traceln("Open WsDepthService:", n, symarr)
		depthSvc := &WsKLineService{id: strconv.Itoa(n), symbols: make([]string, 0), mgr: mgr}
		for _, sym := range symarr {
			mgr.symbolInDepthService[sym] = depthSvc
		}
		depthSvc.symbols = symarr
		mgr.depthServices = append(mgr.depthServices, depthSvc)
	}
	mgr.run()
	return nil
}

func (mgr *WsKLineManager) run() {
	for _, service := range mgr.depthServices {
		go service.run() // never stop
	}
}

//add func Close
