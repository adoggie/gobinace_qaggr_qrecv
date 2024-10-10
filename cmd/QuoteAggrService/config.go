package main

import (
	"github.com/adoggie/ccsuites/core"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/shopspring/decimal"
	"sync"
	"time"
)

type RedisServerAddress struct {
	Addr     string `toml:"addr"`
	Password string `toml:"password"`
	DB       int    `toml:"db"`
}

type Symbol struct {
	Name          string  `toml:"name"`
	Enable        bool    `toml:"enable"`         //
	LimitPosition float64 `toml:"limit_position"` //最大持仓
	Timeout       int64   `toml:"timeout"`
	OrderType     string  `toml:"order_type"`
	Ticks         int64   `toml:"ticks"`
	Leverage      int     `toml:"leverage"` // 开仓杠杆

	target   *decimal.Decimal
	expect   *decimal.Decimal
	position Position
	impl     interface{}              //
	aggTrade *futures.WsAggTradeEvent // 聚合成交记录
	detail   futures.Symbol

	//slices     []*OrderSlice
	orderStart int64 // 报单开始时间

	mtx sync.RWMutex

	// binance.com/zh-CN/futures/trading-rules/perpetual/
	maxPrice    decimal.Decimal
	minPrice    decimal.Decimal
	tickSize    decimal.Decimal
	minNotional decimal.Decimal // 最小名义价值
	maxQty      decimal.Decimal
	minQty      decimal.Decimal
}

func (s Symbol) LastPrice() *decimal.Decimal {
	if s.aggTrade == nil {
		return nil
	}
	if price, e := decimal.NewFromString(s.aggTrade.Price); e == nil {
		return &price
	}
	return nil
}

type TradingSymbol struct {
	Inner   futures.Symbol
	Name    string
	Pair    string
	Filters struct {
		Price struct {
			max      decimal.Decimal
			min      decimal.Decimal
			tickSize decimal.Decimal
		}
		LotSize struct {
			maxQty   decimal.Decimal
			minQty   decimal.Decimal
			stepSize decimal.Decimal
		}
		MarketLotSize struct {
			maxQty   decimal.Decimal
			minQty   decimal.Decimal
			stepSize decimal.Decimal
		}
		MaxNumOrders struct {
			limit decimal.Decimal
		}
		MinNotional  decimal.Decimal
		PercentPrice struct {
			multiplierDecimal int
			multiplierDown    decimal.Decimal
			multiplierUp      decimal.Decimal
		}
		MarketTakeBound decimal.Decimal
	}
}

func NewTradingSymbol(symbol futures.Symbol) *TradingSymbol {
	ts := &TradingSymbol{Inner: symbol}
	ts.Name = symbol.Symbol
	ts.Pair = symbol.Pair

	//utils.Map(symbol.Filters, func(kv map[string]interface{}) interface{} {
	for _, kv := range symbol.Filters {
		switch kv["filterType"] {
		case "PRICE_FILTER":
			ts.Filters.Price.max, _ = decimal.NewFromString(kv["maxPrice"].(string))
			ts.Filters.Price.min, _ = decimal.NewFromString(kv["minPrice"].(string))
			ts.Filters.Price.tickSize, _ = decimal.NewFromString(kv["tickSize"].(string))
		case "LOT_SIZE":
			ts.Filters.LotSize.maxQty, _ = decimal.NewFromString(kv["maxQty"].(string))
			ts.Filters.LotSize.minQty, _ = decimal.NewFromString(kv["minQty"].(string))
			ts.Filters.LotSize.stepSize, _ = decimal.NewFromString(kv["stepSize"].(string))
		case "MARKET_LOT_SIZE":
			ts.Filters.MarketLotSize.maxQty, _ = decimal.NewFromString(kv["maxQty"].(string))
			ts.Filters.MarketLotSize.minQty, _ = decimal.NewFromString(kv["minQty"].(string))
			ts.Filters.MarketLotSize.stepSize, _ = decimal.NewFromString(kv["stepSize"].(string))
		case "MAX_NUM_ORDERS":
			if v, ok := kv["limit"].(int64); ok {
				ts.Filters.MaxNumOrders.limit = decimal.NewFromInt(v)
			}

		case "MIN_NOTIONAL":
			if v, ok := kv["notional"].(string); ok {
				ts.Filters.MinNotional, _ = decimal.NewFromString(v)
			}
		case "PERCENT_PRICE":
			ts.Filters.PercentPrice.multiplierDecimal, _ = kv["multiplierDecimal"].(int)
			ts.Filters.PercentPrice.multiplierDown, _ = decimal.NewFromString(kv["multiplierDown"].(string))
			ts.Filters.PercentPrice.multiplierUp, _ = decimal.NewFromString(kv["multiplierUp"].(string))
		case "MARKET_TAKE":
			ts.Filters.MarketTakeBound, _ = decimal.NewFromString(kv["multiplierUp"].(string))

		}
	}
	return ts
}

type AccountConfig struct {
	Name      string `toml:"name"`
	ApiKey    string `toml:"api_key"`
	SecretKey string `toml:"secret_key"`
	TestNet   bool   `toml:"testnet,omitempty"`
}

type TradingExchangeInfo struct {
	Inner *futures.ExchangeInfo
	//RateLimits map[string]*futures.RateLimit // REQUEST_WEIGHT.MINUTE
}

func NewTradingExchangeInfo(info *futures.ExchangeInfo) *TradingExchangeInfo {
	tei := &TradingExchangeInfo{Inner: info}
	return tei

}

type FanOutDestination struct {
	Name   string `toml:"name"`
	Enable bool   `toml:"enable"`
	Server string `toml:"server"`
	Topic  string `toml:"topic"`
}

type Config struct {
	ThunderId   string `toml:"thunder_id" json:"thunder_id,omitempty" :"thunder_id"`
	ServiceType string `toml:"service_type" json:"service_type,omitempty" :"service_type"`
	Ip          string `toml:"ip"`
	//PositionBroadcastAddress string `toml:"pos_mx_addr" json:"position_broadcast_address,omitempty" :"position_broadcast_address"`
	//PositionSubMode          string `toml:"pos_sub_mode" json:"pos_sub_mode,omitempty" :"pos_sub_mode"`
	//PositionSubEnable        bool   `toml:"pos_sub_enable" json:"pos_sub_enable,omitempty" :"pos_sub_enable"`
	//PositionSubForward       bool   `toml:"pos_sub_forward" json:"pos_sub_forward,omitempty" :"position_sub_deliver`

	QuoteServeAddr     string `toml:"quote_serve_addr" json:"quote_serve_addr" `
	QuoteServeMode     string `toml:"quote_serve_mode" json:"quote_serve_mode" `
	QuoteServeChanSize uint   `toml:"quote_serve_chan_size" json:"quote_serve_chan_size" `

	RedisServer RedisServerAddress `toml:"redis_server" json:"redis_server" :"redis_server"`
	LogFile     string             `toml:"logfile" json:"log_file,omitempty" :"log_file"`
	LogLevel    string             `toml:"log_level" json:"log_level,omitempty" :"log_level"`
	LogDir      string             `toml:"log_dir" json:"log_dir,omitempty" :"log_dir"`
	Symbols     []*Symbol          `toml:"symbols" json:"symbols,omitempty" :"symbols"`
	Accounts    []*AccountConfig   `toml:"accounts" json:"accounts,omitempty" :"accounts"`
	//ApiKey                   string             `toml:"api_key"`
	//SecretKey                string             `toml:"secret_key"`
	PsSigLogDir            string `toml:"pssig_dir" json:"pssig_dir,omitempty" :"ps_log_dir"` // 仓位信号日志
	PsSigLogEnable         bool   `toml:"pssig_log_enable" json:"pssig_log_enable,omitempty" :"ps_log_enable"`
	ReportStatusInterval   int64  `toml:"report_status_interval" json:"report_status_interval,omitempty" :"report_status_interval"` // 上报服务运行状态的时间间隔
	Account                string `toml:"account" json:"account,omitempty" :"account"`
	UserStreamKeepInterval int64  `toml:"userstream_keep_interval" json:"user_stream_keep_interval,omitempty" :"user_stream_keep_interval"`
	MessagePubChan         string `toml:"message_pub_chan" json:"message_pub_chan,omitempty" :"message_pub_chan"`
	//OrderManagerConfig       OrderManagerConfigVars `toml:"order_manager" json:"order_manager_config" :"order_manager_config"`
	TradingSymbols []string `toml:"trading_symbols" json:"trading_symbols,omitempty" :"trading_symbols"`
	//DepthClientManagerConfig  WsDepthClientMgrConfigVars  `toml:"depth_client_manager" json:"depth_client_manager_config" :"depth_client_manager_config"`
	//TradeClientManagerConfig  WsTradeClientMgrConfigVars  `toml:"trade_client_manager" json:"trade_client_manager_config" :"trade_client_manager_config"`
	//OrderbookIncClientManager WsOrbIncClientMgrConfigVars `toml:"orb_inc_client" jorb_inc_clientson:",omitempty" :"orb_inc_client"`

	CancelOrderOnStart bool `toml:"cancel_order_on_start,omitempty" :"cancel_order_on_start"`
	// 撤单偏移
	OrderCancelDeviation      float64  `toml:"order_cancel_deviation" json:"order_cancel_deviation"`
	OrderPricePlaceIndex      uint64   `toml:"order_price_place_index" json:"order_price_place_index"`
	OrderPricePlaceOffset     float32  `toml:"order_price_place_offset" json:"order_price_place_offset"`
	OrderPricePlaceTickOffset float32  `toml:"order_price_place_tick_offset" json:"order_price_place_tick_offset"`
	OrderFillTimeout          int      `toml:"order_fill_timeout" json:"order_fill_timeout"`
	SymbolBlackList           []string `toml:"symbol_blacklist" json:"symbol_blacklist,omitempty" :"symbol_blacklist`
	SigPosMultiplier          float64  `toml:"sigpos_multiplier" json:"sigpos_multiplier,omitempty" :"sigpos_multiplier"`

	RecvDepthEnable  bool `toml:"recv_depth_enable"`
	RecvOrbIncEnable bool `toml:"recv_orbinc_enable"`
	RecvTradeEnable  bool `toml:"recv_trade_enable"`

	FanOutServers  []FanOutDestination `toml:"fanout_servers"`
	DataCompressed bool                `toml:"data_compressed"`

	PeriodValue      int64 `toml:"period_value"`
	PeriodDetachTime int64 `toml:"period_detach_time"`

	WsServerConfig              WsServerConfigVars        `toml:"wsserver"`
	UpdateTradingSymbolInterval int64                     `toml:"update_trading_symbol_interval"`
	TimeIterateInterval         int                       `toml:"time_iterate_interval"`
	SkipEmptyQuote              bool                      `toml:"skip_empty_quote"`
	MonConfig                   core.ServiceMonConfigVars `toml:"qmon_manager"`
}

type Position struct {
	direction string          // long ,short
	volume    decimal.Decimal //
	impl      *futures.AccountPosition
}

type PositionEvent struct {
	thunderId string          // 下单程序id
	symbol    string          // 交易对
	volume    decimal.Decimal // 仓位
	quantity  decimal.Decimal
	qty       decimal.Decimal
}

type Account struct {
	thunderId string  `json:"thunder_id"`
	balance   float64 `json:"balance"`
}

type LogEvent interface {
}

type ServiceSource struct {
	Type      string `json:"type"`
	Id        string `json:"id"`
	Ip        string `json:"ip"`
	Datetime  string `json:"datetime"`
	Timestamp int64  `json:"timestamp"`
}

type ServiceDelta struct {
	DB         string   `json:"db"`
	Table      string   `json:"table"`
	UpdateKeys []string `json:"update_keys"`
}

type ServiceMessage struct {
	Type    string                 `json:"type"`
	Source  ServiceSource          `json:"source"`
	Content map[string]interface{} `json:"content"`
	Delta   ServiceDelta           `json:"delta"`
}

type AccountPosition struct {
	wspos        *futures.WsPosition      // ws 仓位变动
	acpos        *futures.AccountPosition // 查询仓位
	Symbol       string
	PositionSide futures.PositionSideType
	PositionAmt  decimal.Decimal
}

type Timer struct {
	lastTs  int64
	period  int64
	onTimer func(timer *Timer)
	delta   interface{}
}

func NewTimer(period int64, onTimer func(timer *Timer), delta interface{}) *Timer {
	return &Timer{lastTs: time.Now().Unix(), period: period, onTimer: onTimer, delta: delta}
}

func (t *Timer) tick() {
	if t.lastTs+t.period < time.Now().Unix() {
		t.onTimer(t)
		t.lastTs = time.Now().Unix()
	}
}

const FutureLiveNet string = "wss://ws-fapi.binance.com:443/ws-fapi/v1"
const FutureTestNet string = "wss://testnet.binancefuture.com/ws-fapi/v1"

func GetFutureNetUrl(testnet bool) string {
	if testnet {
		return FutureTestNet
	}
	return FutureLiveNet
}
