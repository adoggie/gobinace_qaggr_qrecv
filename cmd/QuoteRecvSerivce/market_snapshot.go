package main

// market_depth.go
// wsocket 订阅接收depth market 行情数据
// 由于订阅最大数量的限制，处理时采用多个 websocket conn 订阅不通的symbol 行情

import (
	"context"
	"github.com/adoggie/ccsuites/core"
	"github.com/adshao/go-binance/v2/futures"
	binance_connector "github.com/binance/binance-connector-go"
	"strconv"
	"sync"
	"time"
)

type SnapShotManagerConfigVars struct {
	TimerIntervalSecs int  `toml:"timer_interval_secs"`
	OnReconnect       bool `toml:"on_reconnect"`
	OnTimer           bool `toml:"on_timer"`
	OnSequenceCheck   bool `toml:"on_sequence_check"`
	TaskQueueSize     int  `toml:"task_queue_size"`
	DepthLimit        int  `toml:"depth_limit"`

	RequestWait int `toml:"request_wait"`
}

type SnapShotService struct {
	symbols      []string
	doneC, stopC chan struct{}
	err          error
	mgr          *WsDepthClientManager
	updates      map[string]interface{}
	id           string
}

func (ds *SnapShotService) reset() {
	ds.stopC <- struct{}{}
}

// 接收到盘口深度价格
func (ds *SnapShotService) handler(event *futures.WsDepthEvent) {
	//logger.Traceln("WsDepthService.handler:", binance_connector.PrettyPrint(event))
	ds.updates[event.Symbol] = nil
	//fmt.Println(time.Now().Format(time.DateTime), "market_depth.go:", "depthservice:", ds.id, " updates:", len(ds.updates))
	//fmt.Println(event.Symbol, event.Asks[0], event.Bids[0])
	ds.mgr.C.OnWsDepthEvent(event)
}

// 出错了，需要重新链接
func (ds *SnapShotService) error(err error) {
	ds.doneC <- struct{}{}
	time.Sleep(time.Second * time.Duration(ds.mgr.config.WaitTimeReconnect))
	go ds.run()
}

func (ds *SnapShotService) run() {
	levels := strconv.FormatInt(10, 10)
	keyValueMap := make(map[string]string)
	for _, key := range ds.symbols {
		keyValueMap[key] = levels
	}
	ds.updates = make(map[string]interface{})
	// 一次订阅所有 depth 行情
	ds.doneC, ds.stopC, ds.err = futures.WsCombinedDepthServe(keyValueMap, ds.handler, ds.error)
	if ds.err != nil {
		logger.Errorln("DepthSub Error:", ds.err.Error())
	}

}

type SnapShotManager struct {
	C                    *Controller
	depthServices        []*SnapShotService
	symbolInDepthService map[string]*SnapShotService
	config               *SnapShotManagerConfigVars
	ctx                  context.Context
	snapShotChan         chan string
	mtx                  sync.Mutex
}

func NewSnapShotManager(controller *Controller) *SnapShotManager {
	mgr := &SnapShotManager{C: controller}
	mgr.config = &controller.config.SnapShotManagerConfig
	mgr.snapShotChan = make(chan string, mgr.config.TaskQueueSize)
	return mgr
}

func (mgr *SnapShotManager) GetApiClient(symbol string) *binance_connector.WebsocketAPIClient {
	return nil
}

func (mgr *SnapShotManager) GetStreamClient(symbol string) *binance_connector.WebsocketStreamClient {
	return nil
}

func (mgr *SnapShotManager) Config() *SnapShotManagerConfigVars {
	return mgr.config
}

func (mgr *SnapShotManager) Init(config *SnapShotManagerConfigVars) *SnapShotManager {
	mgr.config = config
	mgr.depthServices = make([]*SnapShotService, 0)
	mgr.symbolInDepthService = make(map[string]*SnapShotService)
	return mgr
}

// Open() ws通道分配交易对象
func (mgr *SnapShotManager) Open(ctx context.Context) error {
	logger.Infoln("SnapShotManager Open..")
	mgr.ctx = ctx

	//var symbols []string
	//symbols = mgr.C.GetTradeSymbols()
	//symarr_list := splitArrayBySliceSize(symbols, mgr.config.MaxSymbolNumPerConn)
	//for n, symarr := range symarr_list {
	//	//logger.Traceln("Open WsDepthService:", n, symarr)
	//	depthSvc := &WsDepthService{id: strconv.Itoa(n), symbols: make([]string, 0), mgr: mgr}
	//	for _, sym := range symarr {
	//		mgr.symbolInDepthService[sym] = depthSvc
	//	}
	//	depthSvc.symbols = symarr
	//	mgr.depthServices = append(mgr.depthServices, depthSvc)
	//}
	go mgr.run()
	return nil
}

func (mgr *SnapShotManager) run() {
	timer := time.After(time.Second * time.Duration(mgr.config.TimerIntervalSecs))
	for {
		select {
		case <-mgr.ctx.Done():
			logger.Infoln("SnapShotManager ctx done.")
			return
		case <-timer:
			symbols := mgr.C.GetTradeSymbols()
			for _, symbol := range symbols {
				//mgr.snapShotChan <- symbol
				mgr.doSnapShot(symbol)
			}
			timer = time.After(time.Second * time.Duration(mgr.config.TimerIntervalSecs))
		case symbol := <-mgr.snapShotChan:
			go func() {
				mgr.mtx.Lock()
				defer mgr.mtx.Unlock()
				if mgr.config.RequestWait == 0 {
					mgr.config.RequestWait = 1000
				}
				time.Sleep(time.Millisecond * time.Duration(mgr.config.RequestWait))
				if err := mgr.snapshot(symbol); err != nil {
					logger.Errorln("SnapShotManager snapshot error: ", symbol, " detail:", err.Error())
				}
			}()
		}
	}
}

func (mgr *SnapShotManager) doSnapShot(symbol string) {
	go func() {
		mgr.snapShotChan <- symbol
	}()
}

func (mgr *SnapShotManager) snapshot(symbol string) (err error) {
	logger.Infoln("SnapShot Start symbol ..", symbol)
	var res *futures.DepthResponse
	if res, err = controller.clientFuture.NewDepthService().Symbol(symbol).Limit(mgr.config.DepthLimit).Do(context.Background()); err != nil {
		return
	}
	qs := core.QuoteSnapShot{
		Bids: make([]core.BidPrice, 0),
		Asks: make([]core.AskPrice, 0),
	}
	qs.Symbol = symbol
	qs.LastUpdateID = res.LastUpdateID
	qs.Time = res.Time
	qs.TradeTime = res.TradeTime
	logger.Infoln("SnapShot Ok:", symbol, " LastUpdateID:", res.LastUpdateID, " Time:", res.Time, " TradeTime:", res.TradeTime,
		" Bids:", len(res.Bids), " Asks:", len(res.Asks))

	for _, bid := range res.Bids {
		qs.Bids = append(qs.Bids, core.BidPrice{Price: bid.Price, Qty: bid.Quantity})
	}
	for _, ask := range res.Asks {
		qs.Asks = append(qs.Asks, core.AskPrice{Price: ask.Price, Qty: ask.Quantity})
	}
	msg := core.Message{Type: core.MessageType_Quote,
		Payload: core.QuoteMessage{
			Type:     core.QuoteDataType_SnapShot,
			Exchange: core.ExchangeType_Binance,
			Data:     qs}}
	_ = msg
	controller.fanoutMgr.msgCh <- &msg
	return
}
