package core

import "strconv"

type QuoteDataType int

const (
	QuoteDataType_Unknown QuoteDataType = iota
	QuoteDataType_Depth
	QuoteDataType_Trade
	QuoteDataType_Kline
	QuoteDataType_Ticker
	QutoeDataType_Index
	QuoteDataType_OrderIncrement
	QuoteDataType_SnapShot
	QuoteDataType_AggTrade
)

type AskPrice struct {
	Price string
	Qty   string
}
type BidPrice AskPrice

type QuoteMessage struct {
	Type     QuoteDataType
	Exchange ExchangeType
	Data     interface{}
}

type QuoteDepth struct {
	Time             int64
	TransactionTime  int64
	Symbol           string
	FirstUpdateID    int64
	LastUpdateID     int64
	PrevLastUpdateID int64
	Bids             []BidPrice
	Asks             []AskPrice
}

type QuoteSnapShot struct {
	Symbol       string
	LastUpdateID int64
	Time         int64
	TradeTime    int64
	Bids         []BidPrice
	Asks         []AskPrice
}

func (s QuoteSnapShot) GetId() string {
	return strconv.FormatInt(s.Time, 10)
}

type QuoteTrade struct {
	//QuoteMessage
	Event         string `json:"e"`
	Time          int64  `json:"E"`
	Symbol        string `json:"s"`
	TradeID       int64  `json:"t"`
	Price         string `json:"p"`
	Quantity      string `json:"q"`
	BuyerOrderID  int64  `json:"b"`
	SellerOrderID int64  `json:"a"`
	TradeTime     int64  `json:"T"`
	IsBuyerMaker  bool   `json:"m"`
	Placeholder   bool   `json:"M"`
}

func (t QuoteTrade) GetId() string {
	return strconv.FormatInt(t.TradeID, 10)
}

// 增量订单
type QuoteOrbIncr struct {
	//int64 timestamp = 1;
	//bool is_snapshot = 2;
	//string side = 3;
	//float price = 4;
	//float amount = 5;

	//Event         string `json:"e"`
	//Time          int64  `json:"E"`
	//Symbol        string `json:"s"`
	//TradeID       int64  `json:"t"`
	//Price         string `json:"p"`
	//Quantity      string `json:"q"`
	//BuyerOrderID  int64  `json:"b"`
	//SellerOrderID int64  `json:"a"`
	//TradeTime     int64  `json:"T"`
	//IsBuyerMaker  bool   `json:"m"`
	//Placeholder   bool   `json:"M"`

	Time             int64
	TransactionTime  int64
	Symbol           string
	FirstUpdateID    int64
	LastUpdateID     int64
	PrevLastUpdateID int64
	Bids             []BidPrice
	Asks             []AskPrice
}

func (t QuoteOrbIncr) GetId() string {
	return strconv.FormatInt(t.Time, 10)
}

type QuoteKline struct {
	Time int64
	//Kline  WsKline `json:"k"`
	StartTime            int64  `json:"t"`
	EndTime              int64  `json:"T"`
	Symbol               string `json:"s"`
	Interval             string `json:"i"`
	FirstTradeID         int64  `json:"f"`
	LastTradeID          int64  `json:"L"`
	Open                 string `json:"o"`
	Close                string `json:"c"`
	High                 string `json:"h"`
	Low                  string `json:"l"`
	Volume               string `json:"v"`
	TradeNum             int64  `json:"n"`
	IsFinal              bool   `json:"x"`
	QuoteVolume          string `json:"q"`
	ActiveBuyVolume      string `json:"V"`
	ActiveBuyQuoteVolume string `json:"Q"`
}

type QuoteAggTrade struct {
	Time             int64  `json:"E"`
	Symbol           string `json:"s"`
	AggregateTradeID int64  `json:"a"`
	Price            string `json:"p"`
	Quantity         string `json:"q"`
	FirstTradeID     int64  `json:"f"`
	LastTradeID      int64  `json:"l"`
	TradeTime        int64  `json:"T"`
	Maker            bool   `json:"m"`
}
