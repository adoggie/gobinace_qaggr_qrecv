package main

import (
	"errors"
	"github.com/adshao/go-binance/v2/futures"
	"sync"
	"time"
)

const (
	WEIGHT_OrderPlace          = 0
	WEIGHT_OrderCancel         = 1
	WEIGHT_OpenOrdersCancelAll = 1 //撤销单一交易对的所有挂单, 包括订单列表
	WEIGHT_OrderListStatus     = 4 //查询所有挂订单的执行状态
)

type LimitConfig struct {
	interval int64 //
	count    int64
	limit    int64
	current  int64 // current time
}

type RateLimitManager struct {
	mtx                                   sync.RWMutex
	config                                *Config
	close                                 chan struct{}
	order_intervals, req_weight_intervals map[string]*LimitConfig
	// order_intervals['SECOND', 'MINUTE','HOUR', 'DAY']
	// req_weight_intervals['SECOND', 'MINUTE','HOUR', 'DAY']
	inner *futures.ExchangeInfo
}

func NewRateLimitManager(config *Config) *RateLimitManager {
	return &RateLimitManager{config: config, order_intervals: make(map[string]*LimitConfig),
		req_weight_intervals: make(map[string]*LimitConfig)}
}

func (rlm *RateLimitManager) UpdateRateLimitManager(exchange *futures.ExchangeInfo) {
	rlm.mtx.Lock()
	rlm.inner = exchange
	defer rlm.mtx.Unlock()
	for _, rl := range exchange.RateLimits {
		if rl.RateLimitType == "REQUEST_WEIGHT" {
			switch rl.Interval {
			case "SECOND":
				rlm.req_weight_intervals["SECOND"] = &LimitConfig{interval: rl.IntervalNum, limit: rl.Limit}
			case "MINUTE":
				rlm.req_weight_intervals["MINUTE"] = &LimitConfig{interval: rl.IntervalNum, limit: rl.Limit}
			case "HOUR":
				rlm.req_weight_intervals["HOUR"] = &LimitConfig{interval: rl.IntervalNum, limit: rl.Limit}
			case "DAY":
				rlm.req_weight_intervals["DAY"] = &LimitConfig{interval: rl.IntervalNum, limit: rl.Limit}
			}
		} else if rl.RateLimitType == "ORDERS" {
			switch rl.Interval {
			case "SECOND":
				rlm.order_intervals["SECOND"] = &LimitConfig{interval: rl.IntervalNum, limit: rl.Limit}
			case "MINUTE":
				rlm.order_intervals["MINUTE"] = &LimitConfig{interval: rl.IntervalNum, limit: rl.Limit}
			case "HOUR":
				rlm.order_intervals["HOUR"] = &LimitConfig{interval: rl.IntervalNum, limit: rl.Limit}
			case "DAY":
				rlm.order_intervals["DAY"] = &LimitConfig{interval: rl.IntervalNum, limit: rl.Limit}
			}
		}
	}

}

func (rlm *RateLimitManager) Open() {
	timer := time.After(time.Millisecond)
	for {
		select {
		case <-rlm.close:
			return
		case <-timer:
			rlm.mtx.Lock()
			now := time.Now()
			var secs int64
			secs = int64(now.Hour()*3600 + now.Minute()*60 + now.Second())
			if tv, ok := rlm.order_intervals["SECOND"]; ok {
				intv := tv.interval * 1
				if secs%intv == 0 {
					tv.count = 0 // reset
				}
				tv.current = secs / intv * intv // 0 ,10,20,30
			}
			if tv, ok := rlm.order_intervals["MINUTE"]; ok {
				intv := tv.interval * 60
				if secs%intv == 0 {
					tv.count = 0 // reset
				}
				tv.current = secs / intv * intv // 0 ,10,20,30
			}

			if tv, ok := rlm.order_intervals["HOUR"]; ok {
				_ = tv
				panic("Ratelimit.go: rlm.order_intervals[\"HOUR\"] not implemented!")
			}
			//// request_weight
			if tv, ok := rlm.req_weight_intervals["SECOND"]; ok {
				intv := tv.interval * 1
				if secs%intv == 0 {
					tv.count = 0 // reset
				}
				tv.current = secs / intv * intv // 0 ,10,20,30
			}
			if tv, ok := rlm.req_weight_intervals["MINUTE"]; ok {
				intv := tv.interval * 60
				if secs%intv == 0 {
					tv.count = 0 // reset
				}
				tv.current = secs / intv * intv // 0 ,10,20,30
			}

			if tv, ok := rlm.req_weight_intervals["HOUR"]; ok {
				_ = tv
				panic("Ratelimit.go: rlm.req_weight_intervals[\"HOUR\"] not implemented!")
			}
			rlm.mtx.Unlock()
			timer = time.After(time.Millisecond)
		}
	}
}

func (rlm *RateLimitManager) Close() {
	rlm.close <- struct{}{}
}

func (rlm *RateLimitManager) Consume(weight int64, limit int64) (bool, error) {
	// 调用权重限制 weight / like GAS
	rlm.mtx.Lock()
	defer rlm.mtx.Unlock()

	if weight > 0 {
		if tv, ok := rlm.req_weight_intervals["MINUTE"]; ok {
			if tv.count+weight > tv.limit {
				logger.Warnln("RateLimitManager: Consume() failed, weight:", weight, " limit:", tv.limit, " count:", tv.count)
				return false, errors.New("REQWEIGHT_MINUTE_LIMIT_EXCEEDED")
			}
			tv.count += weight
		}

		if tv, ok := rlm.req_weight_intervals["SECOND"]; ok {
			if tv.count+weight > tv.limit {
				logger.Warnln("RateLimitManager: Consume() failed, weight:", weight, " limit:", tv.limit, " count:", tv.count)
				return false, errors.New("REQWEIGHT_SECOND_LIMIT_EXCEEDED")
			}
			tv.count += weight
		}
	}
	// 调用次数限制
	if limit > 0 {
		if tv, ok := rlm.order_intervals["MINUTE"]; ok {
			if tv.count+1 > tv.limit {
				logger.Warnln("RateLimitManager: Consume() failed, order_intervals[\"MINUTE\"]:", " limit:", tv.limit, " count:", tv.count)
				return false, errors.New("ORDER_MINUTE_LIMIT_EXCEEDED")
			}
			tv.count += limit
		}

		if tv, ok := rlm.req_weight_intervals["SECOND"]; ok {
			if tv.count+1 > tv.limit {
				logger.Warnln("RateLimitManager: Consume() failed, req_weight_intervals[\"SECOND\"]:", " limit:", tv.limit, " count:", tv.count)
				return false, errors.New("ORDER_SECOND_LIMIT_EXCEEDED")
			}
			tv.count += limit
		}
	}

	return true, nil

}
