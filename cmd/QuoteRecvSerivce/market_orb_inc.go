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

type WsOrbIncClientMgrConfigVars struct {
	MaxSymbolNumPerConn int `toml:"max_symbol_num_per_conn"`
	WaitTimeReconnect   int `toml:"reconnect_wait_time"`
}

type WsOrbIncService struct {
	symbols             []string
	doneC, stopC        chan struct{}
	err                 error
	mgr                 *WsOrbIncClientManager
	updates             map[string]interface{}
	id                  string
	lastest_update_time atomic.Int64
}

func (ds *WsOrbIncService) reset() {
	if ds.stopC != nil {
		ds.stopC <- struct{}{}
	}

	ds.doneC = nil
	ds.stopC = nil
	ds.lastest_update_time.Store(0)
}

// 接收到盘口深度价格
func (ds *WsOrbIncService) handler(event *futures.WsDepthEvent) {
	//logger.Traceln("WsOrbIncService.handler:", binance_connector.PrettyPrint(event))
	ds.lastest_update_time.Store(time.Now().Unix())
	ds.updates[event.Symbol] = nil
	//fmt.Println(time.Now().Format(time.DateTime), "market_depth.go:", "depthservice:", ds.id, " updates:", len(ds.updates))
	//fmt.Println(event.Symbol, "Time:", event.Time, "TransactTime:", event.TransactionTime, event.Asks[0], event.Bids[0])
	ds.mgr.C.OnWsOrbIncDepthEvent(event)
}

// 出错了，需要重新链接
func (ds *WsOrbIncService) error(err error) {
	logger.Errorln("WsOrbIncService: WebSocket Lost ..", ds.id, " Error:", err.Error())
	//ds.doneC <- struct{}{}
	ds.reset()
	time.Sleep(time.Second * time.Duration(ds.mgr.config.WaitTimeReconnect))
	//go ds.run()
}

func (ds *WsOrbIncService) run() {
	levels := strconv.FormatInt(10, 10)
	keyValueMap := make(map[string]string)
	for _, key := range ds.symbols {
		keyValueMap[key] = levels
	}
	ds.updates = make(map[string]interface{})
	//ds.reset()
	//ds.doConnect()
	//timer := time.After(time.Second * time.Duration(ds.mgr.config.WaitTimeReconnect))
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

}

func (ds *WsOrbIncService) doConnect() {
	//ds.doneC, ds.stopC, ds.err = futures.WsCombinedDepthServe(keyValueMap, ds.handler, ds.error)
	// 250ms default
	//https://developers.binance.com/docs/zh-CN/derivatives/usds-margined-futures/websocket-market-streams/Diff-Book-Depth-Streams
	for {
		logger.Infoln("WsOrbIncService:", ds.id, " Start Connect ..")
		ds.doneC, ds.stopC, ds.err = futures.WsCombinedDiffDepthServe(ds.symbols, ds.handler, ds.error)
		if ds.err != nil {
			logger.Errorln("WsOrbIncService Error:", ds.err.Error(), " Sleep awhile..")
			time.Sleep(time.Second * time.Duration(ds.mgr.config.WaitTimeReconnect))
			continue
		}
		break
	}
	logger.Infoln("WsOrbIncService:", ds.id, " Connect Success..")

	if controller.config.RecvSnapShotEnable {
		// do snapshot
		for _, symbol := range ds.symbols {
			logger.Infoln("SnapShot Init From WsOrbIncService:", ds.id, " symbol:", symbol)
			controller.snapshotMgr.doSnapShot(symbol)
		}
	}
	ds.lastest_update_time.Store(time.Now().Unix())
}

type WsOrbIncClientManager struct {
	C                    *Controller
	services             []*WsOrbIncService
	symbolInDepthService map[string]*WsOrbIncService
	config               *WsOrbIncClientMgrConfigVars
}

func NewWsOrbIncClientManager(controller *Controller) *WsOrbIncClientManager {
	wsClientMgr := &WsOrbIncClientManager{C: controller}
	return wsClientMgr
}

func (mgr *WsOrbIncClientManager) Config() *WsOrbIncClientMgrConfigVars {
	return mgr.config
}

func (mgr *WsOrbIncClientManager) Init(config *WsOrbIncClientMgrConfigVars) *WsOrbIncClientManager {
	mgr.config = config
	mgr.services = make([]*WsOrbIncService, 0)
	mgr.symbolInDepthService = make(map[string]*WsOrbIncService)
	return mgr
}

// Open() ws通道分配交易对象
func (mgr *WsOrbIncClientManager) Open() error {
	var symbols []string
	symbols = mgr.C.GetTradeSymbols()
	symarr_list := splitArrayBySliceSize(symbols, mgr.config.MaxSymbolNumPerConn)
	for n, symarr := range symarr_list {
		//logger.Traceln("Open WsOrbIncService:", n, symarr)
		depthSvc := &WsOrbIncService{id: strconv.Itoa(n), symbols: make([]string, 0), mgr: mgr}
		for _, sym := range symarr {
			mgr.symbolInDepthService[sym] = depthSvc
		}
		depthSvc.symbols = symarr
		mgr.services = append(mgr.services, depthSvc)
	}
	mgr.run()
	return nil
}

func (mgr *WsOrbIncClientManager) run() {
	for _, service := range mgr.services {
		go service.run() // never stop
	}
}

//add func Close
