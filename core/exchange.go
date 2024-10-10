package core

type ExchangeType int

const (
	ExchangeType_Binance ExchangeType = iota
	ExchangeType_Okex
	ExchangeType_Huobi
	ExchangeType_Bitmex
	ExchangeType_Bitfinex
	ExchangeType_Coinbase
)
