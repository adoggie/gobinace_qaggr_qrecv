package main

// market_depth.go
// wsocket 订阅接收depth market 行情数据
// 由于订阅最大数量的限制，处理时采用多个 websocket conn 订阅不通的symbol 行情

import (
	"fmt"
	"github.com/adshao/go-binance/v2/futures"
	"sync/atomic"

	"strconv"
	"time"
)

type WsTradeClientMgrConfigVars struct {
	MaxSymbolNumPerConn int `toml:"max_symbol_num_per_conn"`
	WaitTimeReconnect   int `toml:"reconnect_wait_time"`
}

type WsTradeService struct {
	symbols             []string
	doneC, stopC        chan struct{}
	err                 error
	mgr                 *WsTradeClientManager
	updates             map[string]interface{}
	id                  string
	lastest_update_time atomic.Int64
}

func (ds *WsTradeService) reset() {
	if ds.stopC != nil {
		ds.stopC <- struct{}{}
	}

	ds.doneC = nil
	ds.stopC = nil
	ds.lastest_update_time.Store(0)
}

// 接收到盘口深度价格
func (ds *WsTradeService) handler(event *futures.WsCombinedTradeEvent) {
	//logger.Traceln("WsDepthService.handler:", binance_connector.PrettyPrint(event))
	ds.lastest_update_time.Store(time.Now().Unix())
	ds.updates[event.Data.Symbol] = nil
	//fmt.Println(time.Now().Format(time.DateTime), "market_depth.go:", "depthservice:", ds.id, " updates:", len(ds.updates))
	//fmt.Println(event.Symbol, event.Asks[0], event.Bids[0])
	if event.Data.Symbol == "ETHUSDT" && controller.config.DebugPrintTrade {
		fmt.Println(event.Data.Symbol, "Time:", event.Data.Time, "TradeTime:", event.Data.TradeTime, "TradeID:", event.Data.TradeID, "Price:", event.Data.Price, "Qty:", event.Data.Quantity)
	}
	ds.mgr.C.OnWsTradeEvent(&event.Data)
}

// 出错了，需要重新链接
func (ds *WsTradeService) error(err error) {
	logger.Errorln("WsTradeService: WebSocket Lost ..", ds.id, " Error:", err.Error())
	//ds.doneC <- struct{}{}
	ds.reset()
	time.Sleep(time.Second * time.Duration(ds.mgr.config.WaitTimeReconnect))

}

func (ds *WsTradeService) run() {
	levels := strconv.FormatInt(10, 10)
	keyValueMap := make(map[string]string)
	for _, key := range ds.symbols {
		keyValueMap[key] = levels
	}
	ds.updates = make(map[string]interface{})

	timer := time.After(time.Second * 1)
	for {
		select {
		case <-timer:
			if time.Now().Unix()-ds.lastest_update_time.Load() > int64(controller.config.RecvTradeReconnectTimeout) {
				logger.Infoln("WsTradeService:", ds.id, " Timeout Connect")
				ds.reset()
				ds.doConnect()
			}
			timer = time.After(time.Second * 1)
		}
	}

}

func (ds *WsTradeService) doConnect() {
	//ds.doneC, ds.stopC, ds.err = futures.WsCombinedDepthServe(keyValueMap, ds.handler, ds.error)
	// 250ms default
	//https://developers.binance.com/docs/zh-CN/derivatives/usds-margined-futures/websocket-market-streams/Diff-Book-Depth-Streams
	for {
		logger.Infoln("WsTradeService:", ds.id, " Start Connect ..")
		ds.doneC, ds.stopC, ds.err = futures.WsCombinedTradeServe(ds.symbols, ds.handler, ds.error)
		if ds.err != nil {
			logger.Errorln("WsTradeService Error:", ds.err.Error(), " Sleep awhile..")
			time.Sleep(time.Second * time.Duration(ds.mgr.config.WaitTimeReconnect))
			continue
		}
		break
	}
	logger.Infoln("WsTradeService:", ds.id, " Connect Success..")

	ds.lastest_update_time.Store(time.Now().Unix())
}

type WsTradeClientManager struct {
	C                    *Controller
	services             []*WsTradeService
	symbolInTradeService map[string]*WsTradeService
	config               *WsTradeClientMgrConfigVars
}

func NewWsTradeClientManager(controller *Controller) *WsTradeClientManager {
	wsClientMgr := &WsTradeClientManager{C: controller}
	return wsClientMgr
}

func (mgr *WsTradeClientManager) Config() *WsTradeClientMgrConfigVars {
	return mgr.config
}

func (mgr *WsTradeClientManager) Init(config *WsTradeClientMgrConfigVars) *WsTradeClientManager {
	mgr.config = config
	mgr.services = make([]*WsTradeService, 0)
	mgr.symbolInTradeService = make(map[string]*WsTradeService)
	return mgr
}

// Open() ws通道分配交易对象
func (mgr *WsTradeClientManager) Open() error {
	var symbols []string
	symbols = mgr.C.GetTradeSymbols()
	symarr_list := splitArrayBySliceSize(symbols, mgr.config.MaxSymbolNumPerConn)
	for n, symarr := range symarr_list {
		//logger.Traceln("Open WsDepthService:", n, symarr)
		depthSvc := &WsTradeService{id: strconv.Itoa(n), symbols: make([]string, 0), mgr: mgr}
		for _, sym := range symarr {
			mgr.symbolInTradeService[sym] = depthSvc
		}
		depthSvc.symbols = symarr
		mgr.services = append(mgr.services, depthSvc)
	}
	mgr.run()
	return nil
}

func (mgr *WsTradeClientManager) run() {
	for _, service := range mgr.services {
		go service.run() // never stop
	}
}

//add func Close
