package main

import (
	"context"
	"fmt"
	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"strings"
	"time"
)

func set_leverage(config *Config, symbols *string, leverage int) {
	// ChangeLeverageService change user's initial leverage of specific symbol market
	var ac *AccountConfig
	for _, acc := range config.Accounts {
		if acc.Name == config.Account {
			ac = acc
		}
	}
	clientFuture := binance.NewFuturesClient(ac.ApiKey, ac.SecretKey)

	var exchangeSymbols map[string]futures.Symbol
	exchangeSymbols = make(map[string]futures.Symbol)
	//tradingSymbols := make(map[string]futures.Symbol)

	if exinfo, err := clientFuture.NewExchangeInfoService().Do(context.Background()); err == nil {
		for _, syn := range exinfo.Symbols {
			exchangeSymbols[syn.Symbol] = syn
		}
	} else {
		panic(err.Error())
	}
	leverage_symbols := make([]string, 0)
	if *symbols != "" {
		leverage_symbols = strings.Split(strings.ToUpper(*symbols), ",")
	}
	if len(leverage_symbols) == 0 {
		for name, symbol := range exchangeSymbols {
			if symbol.Status == string(futures.SymbolStatusTypeTrading) {
				leverage_symbols = append(leverage_symbols, name)
			}
		}
	}

	for _, symbol := range leverage_symbols {
		if _, err := clientFuture.NewChangeLeverageService().Symbol(symbol).Leverage(leverage).Do(context.Background()); err != nil {
			logger.Errorln("setLeverage:", symbol, " leverage:", leverage, err.Error())
			panic(err.Error())
		}
		fmt.Println("set leverage:", symbol, "leverage:", leverage, "okay!")
		time.Sleep(time.Millisecond * 100)
	}

}
