package server

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/KSerditov/Trading/api/exchange"
	"github.com/KSerditov/Trading/pkg/exchange/tickers"
)

type ExchangeSrv struct {
	BufferSize int

	Tickers tickers.TickersSource

	MaxDealID     int64
	OrderBookLock *sync.RWMutex
	OrderBook     []*exchange.Deal

	ohlcvId int64

	ChannelsLock *sync.RWMutex
	Channels     map[*exchange.BrokerID]chan *exchange.Deal

	exchange.UnimplementedExchangeServer
}

// поток ценовых данных от биржи к брокеру
// мы каждую секнуду будем получать отсюда событие с ценами, которые брокер аггрегирует у себя в минуты и показывает клиентам
// устанавливается 1 раз брокером
func (e *ExchangeSrv) Statistic(brokerID *exchange.BrokerID, exchangeStatisticServer exchange.Exchange_StatisticServer) error {
	fmt.Println(brokerID)
	interval := time.Second * 1
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	ctx := exchangeStatisticServer.Context()

	for {
		select {
		//case ev := <-ch:

		case timetick := <-ticker.C:
			t, err := e.Tickers.GetTickersBeforeTS(timetick, interval)
			if err != nil {
				return err
			}

			var ohlcvs map[string]*exchange.OHLCV
			ohlcvs = make(map[string]*exchange.OHLCV, 2)
			var opents, closets time.Time

			for _, v := range t {
				_, ok := ohlcvs[v.Ticker]
				if !ok {
					opents = v.Timestamp
					closets = v.Timestamp

					newOHLCV := &exchange.OHLCV{
						ID:       atomic.AddInt64(&e.ohlcvId, 1),
						Time:     int32(timetick.Unix()),
						Interval: int32(interval),
						Open:     v.Last,
						High:     v.Last,
						Low:      v.Last,
						Close:    v.Last,
						Volume:   v.Vol,
						Ticker:   v.Ticker,
					}
					ohlcvs[v.Ticker] = newOHLCV

					continue
				}

				atomic.AddInt32(&ohlcvs[v.Ticker].Volume, v.Vol)

				if v.Last > ohlcvs[v.Ticker].High {
					ohlcvs[v.Ticker].High = v.Last
				}

				if v.Last < ohlcvs[v.Ticker].Low {
					ohlcvs[v.Ticker].Low = v.Last
				}

				if v.Timestamp.After(closets) {
					ohlcvs[v.Ticker].Close = v.Last
					closets = v.Timestamp
				}

				if v.Timestamp.Before(opents) {
					ohlcvs[v.Ticker].Open = v.Last
					opents = v.Timestamp
				}
			}

			for _, v := range ohlcvs {
				errsend := exchangeStatisticServer.Send(v)

				if errsend != nil {
					return errsend
				}
			}

		case <-ctx.Done():
			return nil
		}
	}
}

// отправка на биржу заявки от брокера
func (e *ExchangeSrv) Create(ctx context.Context, deal *exchange.Deal) (*exchange.DealID, error) {
	deal.ID = atomic.AddInt64(&e.MaxDealID, 1)

	e.OrderBookLock.Lock()
	e.OrderBook = append(e.OrderBook, deal)
	fmt.Println(e.OrderBook)
	e.OrderBookLock.Unlock()

	dealid := &exchange.DealID{
		ID:       deal.ID,
		BrokerID: int64(deal.BrokerID),
	}

	return dealid, nil
}

// отмена заявки
func (e *ExchangeSrv) Cancel(ctx context.Context, deal *exchange.DealID) (*exchange.CancelResult, error) {
	cancelResult := &exchange.CancelResult{Success: false}

	e.OrderBookLock.Lock()
	defer e.OrderBookLock.Unlock()

	for i := len(e.OrderBook) - 1; i >= 0; i-- {
		if deal.ID == e.OrderBook[i].ID {
			e.OrderBook = append(e.OrderBook[:i], e.OrderBook[i+1:]...)
			cancelResult.Success = true
			break
		}
	}

	//fmt.Println(e.OrderBook)

	if !cancelResult.Success {
		return cancelResult, errors.New("no such deal id found")
	}
	return cancelResult, nil
}

// исполнение заявок от биржи к брокеру
// устанавливается 1 раз брокером и при исполнении какой-то заявки
func (e *ExchangeSrv) Results(brokerID *exchange.BrokerID, exchangeResultsServer exchange.Exchange_ResultsServer) error {
	return nil
}

func (e *ExchangeSrv) GetBrokerChannel(brokerId *exchange.BrokerID) (chan *exchange.Deal, error) {
	e.ChannelsLock.Lock()
	defer e.ChannelsLock.Unlock()

	val, ok := e.Channels[brokerId]
	if ok {
		return val, nil
	} else {
		e.Channels[brokerId] = make(chan *exchange.Deal, e.BufferSize)
		return e.Channels[brokerId], nil
	}
}

func (e *ExchangeSrv) StartTrader() error {
	interval := time.Second * 1
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	/*
		for {
			select {
			case timetick := <-ticker.C:
				_, err := e.Tickers.GetTickersBeforeTS(timetick, interval)
				if err != nil {
					return err
				}
				for i, order := range e.OrderBook {
					for j, ticker := range tickers {

					}
				}

			}
		}
	*/
	/*
		1. Get new tickers for period
		2. For each ticker, iterate over order book
			if fits - perform deal: set deal params; send to broker channel
			if not - skip order
	*/
	return nil
}
