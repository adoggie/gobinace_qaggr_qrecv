package main

// market_depth.go
// wsocket 订阅接收depth market 行情数据
// 由于订阅最大数量的限制，处理时采用多个 websocket conn 订阅不通的symbol 行情

import (
	"github.com/adshao/go-binance/v2/futures"
	"strconv"
	"sync/atomic"
	"time"
)

type WsDepthClientMgrConfigVars struct {
	MaxSymbolNumPerConn int `toml:"max_symbol_num_per_conn"`
	WaitTimeReconnect   int `toml:"reconnect_wait_time"`
	DepthLevel          int `toml:"depth_level"`
}

type WsDepthService struct {
	symbols             []string
	doneC, stopC        chan struct{}
	err                 error
	mgr                 *WsDepthClientManager
	updates             map[string]interface{}
	id                  string
	lastest_update_time atomic.Int64
}

func (ds *WsDepthService) reset() {
	if ds.stopC != nil {
		ds.stopC <- struct{}{}
	}

	ds.doneC = nil
	ds.stopC = nil
	ds.lastest_update_time.Store(0)
}

// 接收到盘口深度价格
func (ds *WsDepthService) handler(event *futures.WsDepthEvent) {
	//logger.Traceln("WsDepthService.handler:", binance_connector.PrettyPrint(event))
	ds.lastest_update_time.Store(time.Now().Unix())
	ds.updates[event.Symbol] = nil
	//fmt.Println(time.Now().Format(time.DateTime), "market_depth.go:", "depthservice:", ds.id, " updates:", len(ds.updates))
	//fmt.Println(event.Symbol, event.Asks[0], event.Bids[0])
	ds.mgr.C.OnWsDepthEvent(event)
}

// 出错了，需要重新链接
func (ds *WsDepthService) error(err error) {
	//ds.doneC <- struct{}{}
	//time.Sleep(time.Second * time.Duration(ds.mgr.config.WaitTimeReconnect))
	//go ds.run()

	logger.Errorln("WsDepthService: WebSocket Lost ..", ds.id, " Error:", err.Error())
	//ds.doneC <- struct{}{}
	ds.reset()
	time.Sleep(time.Second * time.Duration(ds.mgr.config.WaitTimeReconnect))

}

func (ds *WsDepthService) run() {
	levels := strconv.FormatInt(int64(ds.mgr.config.DepthLevel), 10)
	keyValueMap := make(map[string]string)
	for _, key := range ds.symbols {
		keyValueMap[key] = levels
	}
	ds.updates = make(map[string]interface{})

	timer := time.After(time.Second * 1)
	for {
		select {
		case <-timer:
			if time.Now().Unix()-ds.lastest_update_time.Load() > int64(controller.config.RecvOrbIncReconnectTimeout) {
				logger.Infoln("WsOrbIncService:", ds.id, " Timeout, Connect")
				ds.reset()
				ds.doConnect()
			}
			timer = time.After(time.Second * 1)
		}
	}

	// 一次订阅所有 depth 行情

}

func (ds *WsDepthService) doConnect() {
	//ds.doneC, ds.stopC, ds.err = futures.WsCombinedDepthServe(keyValueMap, ds.handler, ds.error)
	// 250ms default
	//https://developers.binance.com/docs/zh-CN/derivatives/usds-margined-futures/websocket-market-streams/Diff-Book-Depth-Streams
	levels := strconv.FormatInt(int64(ds.mgr.config.DepthLevel), 10)
	keyValueMap := make(map[string]string)
	for _, key := range ds.symbols {
		keyValueMap[key] = levels
	}

	for {
		logger.Infoln("WsDepthService:", ds.id, " Start Connect ..")
		//ds.doneC, ds.stopC, ds.err = futures.WsCombinedDiffDepthServe(ds.symbols, ds.handler, ds.error)
		//if ds.err != nil {
		//	logger.Errorln("WsOrbIncService Error:", ds.err.Error(), " Sleep awhile..")
		//	time.Sleep(time.Second * time.Duration(ds.mgr.config.WaitTimeReconnect))
		//}
		ds.doneC, ds.stopC, ds.err = futures.WsCombinedDepthServe(keyValueMap, ds.handler, ds.error)
		//ds.doneC, ds.stopC, ds.err = futures.WsCombinedDiffDepthServe(keyValueMap, ds.handler, ds.error)
		if ds.err != nil {
			logger.Errorln("DepthSub Error:", ds.err.Error())
			time.Sleep(time.Second * time.Duration(ds.mgr.config.WaitTimeReconnect))
			continue
		}

		break
	}
	logger.Infoln("WsDepthService:", ds.id, " Connect Success..")

	ds.lastest_update_time.Store(time.Now().Unix())
}

type WsDepthClientManager struct {
	C                    *Controller
	depthServices        []*WsDepthService
	symbolInDepthService map[string]*WsDepthService
	config               *WsDepthClientMgrConfigVars
}

func NewWsDepthClientManager(controller *Controller) *WsDepthClientManager {
	wsClientMgr := &WsDepthClientManager{C: controller}
	return wsClientMgr
}

//func (mgr *WsDepthClientManager) GetApiClient(symbol string) *binance_connector.WebsocketAPIClient {
//	return nil
//}
//
//func (mgr *WsDepthClientManager) GetStreamClient(symbol string) *binance_connector.WebsocketStreamClient {
//	return nil
//}

func (mgr *WsDepthClientManager) Config() *WsDepthClientMgrConfigVars {
	return mgr.config
}

func (mgr *WsDepthClientManager) Init(config *WsDepthClientMgrConfigVars) *WsDepthClientManager {
	mgr.config = config
	mgr.depthServices = make([]*WsDepthService, 0)
	mgr.symbolInDepthService = make(map[string]*WsDepthService)
	return mgr
}

// Open() ws通道分配交易对象
func (mgr *WsDepthClientManager) Open() error {
	var symbols []string
	symbols = mgr.C.GetTradeSymbols()
	symarr_list := splitArrayBySliceSize(symbols, mgr.config.MaxSymbolNumPerConn)
	for n, symarr := range symarr_list {
		//logger.Traceln("Open WsDepthService:", n, symarr)
		depthSvc := &WsDepthService{id: strconv.Itoa(n), symbols: make([]string, 0), mgr: mgr}
		for _, sym := range symarr {
			mgr.symbolInDepthService[sym] = depthSvc
		}
		depthSvc.symbols = symarr
		mgr.depthServices = append(mgr.depthServices, depthSvc)
	}
	mgr.run()
	return nil
}

func (mgr *WsDepthClientManager) run() {
	for _, service := range mgr.depthServices {
		go service.run() // never stop
	}
}

//add func Close
