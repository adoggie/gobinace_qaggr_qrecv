package main

import (
	"github.com/adoggie/ccsuites/core"
	"github.com/adoggie/ccsuites/message"
	"github.com/bits-and-blooms/bloom/v3"
	"strconv"
	"sync"
	"time"
)

import (
	"container/list"
)

// 每次启动或运行到凌晨2:00 重新计算 所有symbols 的 bloomfilter mask
var (
	PERIOD      int64 = 3 * 1000 // sec
	DETACH_TIME int64 = 1400     // ms
)

var (
	symbol_bloomfilter *bloom.BloomFilter
)

func init() {
	symbol_bloomfilter = bloom.NewWithEstimates(1000000, 0.01)
}

type SymbolDef struct {
	name string
	//slot   Slot
	filter           *bloom.BloomFilter
	period           int64
	bornTs           int64
	linked           list.List
	mtx              sync.Mutex
	pending_inc      list.List //后续到达的
	pending_trade    list.List //后续到达的
	pending_snapshot list.List
}

func NewSymbolDef(name string) *SymbolDef {
	sdef := &SymbolDef{name: name}
	sdef.linked.Init()
	sdef.pending_inc.Init()
	sdef.pending_trade.Init()
	sdef.pending_snapshot.Init()
	//sdef.slot.sdef = sdef
	return sdef
}

func (s *SymbolDef) put_snapshot(snap *core.QuoteSnapShot) {

	var ok bool
	s.mtx.Lock()
	defer s.mtx.Unlock()
	var back, front *PeriodNode
	if s.linked.Len() == 0 {
		return
	}

	if s.linked.Len() == 1 {
		if back, ok = s.linked.Back().Value.(*PeriodNode); !ok {
			logger.Info(DETACH_TIME, "dropped snapshot, back value as PeriodNode failed ", snap.Symbol)
			return
		}
		if snap.Time < back.start || snap.Time >= back.end {
			logger.Info(DETACH_TIME, "dropped snapshot, out of range. ", "link size:", s.linked.Len(), " ", snap.Symbol, " ", snap.GetId(), " not in: ", back.start, back.end)
			return
		}
		if _, ok := back.snapshot_map[snap.GetId()]; ok { // 已存在则忽略
			//logger.Error(DETACH_TIME, trade.Symbol, "drop trade, existed. ", trade.GetId())
			return
		}
		back.snapshot_map[snap.GetId()] = snap
		back.snapshot_index = append(back.snapshot_index, snap)

	} else {
		//nodes := make([]*PeriodNode, 2)
		if front, ok = s.linked.Front().Value.(*PeriodNode); !ok {
			logger.Info(DETACH_TIME, "dropped snapshot, front value as PeriodNode failed ", snap.Symbol)
			return
		}
		//nodes = append(nodes, front)
		if back, ok = s.linked.Back().Value.(*PeriodNode); !ok {
			logger.Info(DETACH_TIME, "dropped snapshot, back value as PeriodNode failed ", snap.Symbol)
			return
		}
		//nodes = append(nodes, back)

		//if trade.Time < front.start || trade.Time >= back.end {
		//	logger.Error(DETACH_TIME, "dropped trade, out of range. ", "link size: ", s.linked.Len(), " ", trade.Symbol, " ", trade.Time, " not in: ", front.start, " ", back.end)
		//	return
		//}

		//if trade.Time < front.start || trade.Time > back.end+10000 { //超过100s丢弃
		//	logger.Error(DETACH_TIME, "dropped trade, out of range. ", "link size: ", s.linked.Len(), " ", trade.Symbol, " ", trade.Time, " not in: ", front.start, " ", back.end)
		//	return
		//}

		if snap.Time < front.start-PERIOD {
			logger.Info("dropped snapshot, out of range. snapshot earlier than first： ", PERIOD, " link size: ", s.linked.Len(), " ", snap.Symbol, " ", snap.GetId(), " time:", snap.Time, " not in: ", front.start, " ", back.end)
			return
		}
		if snap.Time >= back.end+10*1000 { // after 10 seconds
			logger.Info("dropped snapshot, out of range. snapshot later than back：10 seconds ", " link size: ", s.linked.Len(), " ", snap.Symbol, " ", snap.GetId(), " time:", snap.Time, " not in: ", front.start, " ", back.end)
			return
		}
		//if trade.Time >= front.start && trade.Time < front.end {
		if snap.Time < front.end {
			if _, ok := front.trademap[snap.GetId()]; ok {
				//logger.Error(DETACH_TIME, trade.Symbol, "drop trade, existed in front ", trade.GetId())
				return
			}
			front.snapshot_map[snap.GetId()] = snap
			front.snapshot_index = append(front.snapshot_index, snap)
			return
		}
		if snap.Time >= back.start && snap.Time < back.end {
			if _, ok := back.snapshot_map[snap.GetId()]; ok {
				//logger.Error(DETACH_TIME, trade.Symbol, "drop trade, existed in back ", trade.GetId())
				return
			}
			back.snapshot_map[snap.GetId()] = snap
			back.snapshot_index = append(back.snapshot_index, snap)
			return
		}
		//prepare for later
		s.pending_snapshot.PushBack(snap)

	}
}

func (s *SymbolDef) put_trade(trade *core.QuoteTrade) {

	var ok bool
	s.mtx.Lock()
	defer s.mtx.Unlock()
	var back, front *PeriodNode
	if s.linked.Len() == 0 {
		return
	}

	if s.linked.Len() == 1 {
		if back, ok = s.linked.Back().Value.(*PeriodNode); !ok {
			logger.Info(DETACH_TIME, "dropped trade, back value as PeriodNode failed ", trade.Symbol)
			return
		}
		if trade.Time < back.start || trade.Time >= back.end {
			logger.Info(DETACH_TIME, "dropped trade, out of range. ", "link size:", s.linked.Len(), " ", trade.Symbol, " ", trade.GetId(), " not in: ", back.start, back.end)
			return
		}
		if _, ok := back.trademap[trade.GetId()]; ok {
			//logger.Error(DETACH_TIME, trade.Symbol, "drop trade, existed. ", trade.GetId())
			return
		}
		back.trademap[trade.GetId()] = trade
		back.tradeindex = append(back.tradeindex, trade)

	} else {
		//nodes := make([]*PeriodNode, 2)
		if front, ok = s.linked.Front().Value.(*PeriodNode); !ok {
			logger.Info(DETACH_TIME, "dropped trade, front value as PeriodNode failed ", trade.Symbol)
			return
		}
		//nodes = append(nodes, front)
		if back, ok = s.linked.Back().Value.(*PeriodNode); !ok {
			logger.Info(DETACH_TIME, "dropped trade, back value as PeriodNode failed ", trade.Symbol)
			return
		}
		//nodes = append(nodes, back)

		//if trade.Time < front.start || trade.Time >= back.end {
		//	logger.Error(DETACH_TIME, "dropped trade, out of range. ", "link size: ", s.linked.Len(), " ", trade.Symbol, " ", trade.Time, " not in: ", front.start, " ", back.end)
		//	return
		//}

		//if trade.Time < front.start || trade.Time > back.end+10000 { //超过100s丢弃
		//	logger.Error(DETACH_TIME, "dropped trade, out of range. ", "link size: ", s.linked.Len(), " ", trade.Symbol, " ", trade.Time, " not in: ", front.start, " ", back.end)
		//	return
		//}

		if trade.Time < front.start-PERIOD {
			logger.Info("dropped trade, out of range. trade earlier than first： ", PERIOD, " link size: ", s.linked.Len(), " ", trade.Symbol, " ", trade.GetId(), " time:", trade.Time, " not in: ", front.start, " ", back.end)
			return
		}
		if trade.Time >= back.end+10*1000 {
			logger.Info("dropped trade, out of range. trade later than back：10 seconds ", " link size: ", s.linked.Len(), " ", trade.Symbol, " ", trade.GetId(), " time:", trade.Time, " not in: ", front.start, " ", back.end)
			return
		}
		//if trade.Time >= front.start && trade.Time < front.end {
		if trade.Time < front.end {
			if _, ok := front.trademap[trade.GetId()]; ok {
				//logger.Error(DETACH_TIME, trade.Symbol, "drop trade, existed in front ", trade.GetId())
				return
			}
			front.trademap[trade.GetId()] = trade
			front.tradeindex = append(front.tradeindex, trade)
			return
		}
		if trade.Time >= back.start && trade.Time < back.end {
			if _, ok := back.trademap[trade.GetId()]; ok {
				//logger.Error(DETACH_TIME, trade.Symbol, "drop trade, existed in back ", trade.GetId())
				return
			}
			back.trademap[trade.GetId()] = trade
			back.tradeindex = append(back.tradeindex, trade)
			return
		}
		//prepare for later
		s.pending_trade.PushBack(trade)

	}
}

func (s *SymbolDef) put_inc(inc *core.QuoteOrbIncr) {
	var back, front *PeriodNode
	var ok bool
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.linked.Len() == 0 {
		return
	}

	if s.linked.Len() == 1 {
		if back, ok = s.linked.Back().Value.(*PeriodNode); !ok {
			logger.Traceln(DETACH_TIME, "dropped inc, back value as PeriodNode failed ", inc.Symbol)
			return
		}
		if inc.Time < back.start || inc.Time >= back.end {
			logger.Traceln(DETACH_TIME, "dropped inc, out of range. ", "link size: ", s.linked.Len(), " ", inc.Symbol, " ", inc.GetId(), " not in: ", back.start, " ", back.end)
			return
		}
		if _, ok := back.incmap[inc.GetId()]; ok {
			//logger.Error(DETACH_TIME, inc.Symbol, "drop inc, existed. ", inc.GetId())
			return
		}
		back.incmap[inc.GetId()] = inc
		back.incindex = append(back.incindex, inc)

	} else {
		//nodes := make([]*PeriodNode, 2)
		if front, ok = s.linked.Front().Value.(*PeriodNode); !ok {
			logger.Traceln(DETACH_TIME, "dropped inc, front value as PeriodNode failed ", inc.Symbol)
			return
		}
		//nodes = append(nodes, front)
		if back, ok = s.linked.Back().Value.(*PeriodNode); !ok {
			logger.Traceln(DETACH_TIME, "dropped inc, back value as PeriodNode failed ", inc.Symbol)
			return
		}
		//nodes = append(nodes, back)

		//if inc.Time < front.start || inc.Time >= back.end+10*1000 {
		//	logger.Error(DETACH_TIME, "dropped inc, out of range. ", "link size: ", s.linked.Len(), " ", inc.Symbol, " ", inc.GetId(), " not in: ", front.start, " ", back.end)
		//	return
		//}

		//if inc.Time < front.start || inc.Time >= back.end {
		//	logger.Error(DETACH_TIME, "dropped inc, out of range. ", "link size: ", s.linked.Len(), " ", inc.Symbol, " ", inc.GetId(), " not in: ", front.start, " ", back.end)
		//	return
		//}

		if inc.Time < front.start-PERIOD {
			logger.Traceln("dropped inc, out of range. inc earlier than first： ", PERIOD, " link size: ", s.linked.Len(), " ", inc.Symbol, " ", inc.GetId(), " time:", inc.Time, " not in: ", front.start, " ", back.end)
			return
		}
		if inc.Time >= back.end+10*1000 {
			logger.Traceln("dropped inc, out of range. inc later than back：10 seconds ", " link size: ", s.linked.Len(), " ", inc.Symbol, " ", inc.GetId(), " time:", inc.Time, " not in: ", front.start, " ", back.end)
			return
		}

		//if inc.Time >= front.start && inc.Time < front.end {
		if inc.Time < front.end {
			if _, ok := front.incmap[inc.GetId()]; ok {
				//logger.Error(DETACH_TIME, inc.Symbol, "drop inc, existed in front ", inc.GetId())
				return
			}
			front.incmap[inc.GetId()] = inc
			front.incindex = append(front.incindex, inc)
			return
		}

		if inc.Time >= back.start && inc.Time < back.end {
			if _, ok := back.incmap[inc.GetId()]; ok {
				//logger.Error(DETACH_TIME, inc.Symbol, "drop inc, existed in back ", inc.GetId())
				return
			}
			back.incmap[inc.GetId()] = inc
			back.incindex = append(back.incindex, inc)
			return
		}

		s.pending_inc.PushBack(inc)

	}

}

// timeItr 定时摘取过时的Period, 并添加第二个Period Node
func (s *SymbolDef) timeItr(now time.Time, bornTs int64) *PeriodNode {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	tick := now.UnixMilli()
	if s.linked.Len() == 0 {
		node := NewPeriodNode()
		node.start = bornTs
		node.symbol = s
		node.end = node.start + PERIOD
		s.linked.PushBack(node)
		return nil
	}
	if s.linked.Len() == 1 {
		pn := s.linked.Front().Value.(*PeriodNode)
		node := NewPeriodNode()
		node.start = pn.end
		node.symbol = s
		node.end = node.start + PERIOD
		s.linked.PushBack(node)
	}

	pn := s.linked.Front().Value.(*PeriodNode)

	if tick > int64(pn.end+DETACH_TIME) {
		return s.detach()
	}
	return nil
}

func (s *SymbolDef) detach() *PeriodNode {
	e := s.linked.Front()
	pn := e.Value.(*PeriodNode)
	s.linked.Remove(e)
	{
		pnx := s.linked.Front().Value.(*PeriodNode)
		node := NewPeriodNode()
		node.start = pnx.end
		node.symbol = s
		node.end = node.start + PERIOD
		s.linked.PushBack(node)
	}

	// 将pending_inc,pending_trade 符合time要求的一并加入
	for e := s.pending_trade.Front(); e != nil; e = e.Next() {
		trd := e.Value.(*core.QuoteTrade)
		if trd.Time >= pn.start && trd.Time < pn.end {
			if _, ok := pn.trademap[trd.GetId()]; !ok {
				pn.tradeindex = append(pn.tradeindex, trd)
			}
			s.pending_trade.Remove(e)
		}
	}

	for e := s.pending_inc.Front(); e != nil; e = e.Next() {
		inc := e.Value.(*core.QuoteOrbIncr)
		if inc.Time >= pn.start && inc.Time < pn.end {
			if _, ok := pn.incmap[inc.GetId()]; !ok {
				pn.incindex = append(pn.incindex, inc)
			}
			s.pending_inc.Remove(e)
		}
	}

	for e := s.pending_snapshot.Front(); e != nil; e = e.Next() {
		snap := e.Value.(*core.QuoteSnapShot)
		if snap.Time >= pn.start && snap.Time < pn.end {
			if _, ok := pn.snapshot_map[snap.GetId()]; !ok {
				pn.snapshot_index = append(pn.snapshot_index, snap)
			}
			s.pending_snapshot.Remove(e)
		}
	}
	return pn
}

//
//type Slot struct {
//	linked list.List
//	cur    *PeriodNode
//	sdef   *SymbolDef
//}

//
//func (s *Slot) reInit(maxNode int, period uint64) {
//	len := s.linked.Len()
//	np := maxNode - len
//	if np <= 0 {
//		return
//	}
//
//	for n := 0; n < np; n++ {
//		back := s.linked.Back().Value.(*PeriodNode)
//		node := NewPeriodNode()
//		node.symbol = s.sdef
//		node.start = back.start + period
//		s.linked.PushBack(node)
//	}
//}

type PeriodNode struct {
	start  int64
	end    int64
	symbol *SymbolDef
	filter *bloom.BloomFilter
	//smtrade    *sortedmap.SortedMap[uint64, *core.QuoteTrade]
	//sminc      *sortedmap.SortedMap[uint64, *core.QuoteOrbIncr]
	trademap   map[string]*core.QuoteTrade
	tradeindex []*core.QuoteTrade
	incmap     map[string]*core.QuoteOrbIncr
	incindex   []*core.QuoteOrbIncr

	snapshot_map   map[string]*core.QuoteSnapShot
	snapshot_index []*core.QuoteSnapShot
}

// func (pn *PeriodNode) MarshallPb() ([]*message.TradeInfo, []*message.IncrementOrderBookInfo) {
func (pn *PeriodNode) MarshallPb(sinfo *message.SymbolInfo) {
	sinfo.Trades = make([]*message.TradeInfo, 0)
	sinfo.Incs = make([]*message.IncrementOrderBookInfo, 0)
	for _, trd := range pn.tradeindex {
		//var err error
		info := &message.TradeInfo{}
		info.Timestamp = trd.Time
		info.TradeId = trd.TradeID
		if v, err := strconv.ParseFloat(trd.Price, 64); err == nil {
			info.Price = v
		}
		if v, err := strconv.ParseFloat(trd.Quantity, 64); err == nil {
			info.Qty = v
		}
		info.BuyerOrderId = trd.BuyerOrderID
		info.SellerOrderId = trd.SellerOrderID
		info.TradeTs = trd.TradeTime
		info.IsBuyerMaker = trd.IsBuyerMaker
		info.PlaceHolder = trd.Placeholder

		sinfo.Trades = append(sinfo.Trades, info)
	}

	for _, inc := range pn.incindex {
		info := &message.IncrementOrderBookInfo{}
		info.Timestamp = inc.Time
		info.TransTs = inc.TransactionTime
		info.FirstUpdateId = inc.FirstUpdateID
		info.LastUpdateId = inc.LastUpdateID
		info.PrevLastUpdateId = inc.PrevLastUpdateID

		for _, bid := range inc.Bids {
			pl := &message.PriceLevel{}
			if v, err := strconv.ParseFloat(bid.Qty, 64); err == nil {
				pl.Amount = v
			}
			if v, err := strconv.ParseFloat(bid.Price, 64); err == nil {
				pl.Price = v
			}
			info.Bids = append(info.Bids, pl)
		}

		for _, ask := range inc.Asks {
			pl := &message.PriceLevel{}

			if v, err := strconv.ParseFloat(ask.Qty, 64); err == nil {
				pl.Amount = v
			}
			if v, err := strconv.ParseFloat(ask.Price, 64); err == nil {
				pl.Price = v
			}
			info.Asks = append(info.Asks, pl)
		}
		sinfo.Incs = append(sinfo.Incs, info)
	}

	// snapshot
	for _, snap := range pn.snapshot_index {
		info := &message.SnapShotInfo{}
		info.LastUpdateId = snap.LastUpdateID
		info.Timestamp = snap.Time
		info.TradeTs = snap.TradeTime

		for _, bid := range snap.Bids {
			pl := &message.PriceLevel{}
			if v, err := strconv.ParseFloat(bid.Qty, 64); err == nil {
				pl.Amount = v
			}
			if v, err := strconv.ParseFloat(bid.Price, 64); err == nil {
				pl.Price = v
			}
			info.Bids = append(info.Bids, pl)
		}

		for _, ask := range snap.Asks {
			pl := &message.PriceLevel{}

			if v, err := strconv.ParseFloat(ask.Qty, 64); err == nil {
				pl.Amount = v
			}
			if v, err := strconv.ParseFloat(ask.Price, 64); err == nil {
				pl.Price = v
			}
			info.Asks = append(info.Asks, pl)
		}
		sinfo.Snaps = append(sinfo.Snaps, info)
	}
}

func fx(a, b *core.QuoteTrade) bool {
	return true
}

func NewPeriodNode() *PeriodNode {
	node := &PeriodNode{}
	node.trademap = make(map[string]*core.QuoteTrade)
	node.tradeindex = make([]*core.QuoteTrade, 0)

	node.incmap = make(map[string]*core.QuoteOrbIncr)
	node.incindex = make([]*core.QuoteOrbIncr, 0)

	node.snapshot_map = make(map[string]*core.QuoteSnapShot)
	node.snapshot_index = make([]*core.QuoteSnapShot, 0)
	const n = 1

	//node.smtrade = sortedmap.New(n, fx)
	//sm := sortedmap.New(n, asc.Time)
	//x := sortedmap.New(n, asc.Time)
	return node
}

//https://github.com/Tobshub/go-sortedmap/tree/v1.0.3/examples
