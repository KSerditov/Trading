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

	"github.com/google/uuid"
)

type ExchangeSrv struct {
	BufferSize int

	Tickers tickers.TickersSource

	MaxDealID     int64
	OrderBookLock *sync.RWMutex
	OrderBook     []*exchange.Deal

	ohlcvId int64

	ChannelsLock *sync.RWMutex
	Channels     map[int64]chan *exchange.Deal

	exchange.UnimplementedExchangeServer
}

// поток ценовых данных от биржи к брокеру
// мы каждую секнуду будем получать отсюда событие с ценами, которые брокер аггрегирует у себя в минуты и показывает клиентам
// устанавливается 1 раз брокером
func (e *ExchangeSrv) Statistic(brokerID *exchange.BrokerID, exchangeStatisticServer exchange.Exchange_StatisticServer) error {
	fmt.Printf("Broker connected to Statistic, brokerId: %v\n", brokerID)
	interval := time.Second * 1
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	ctx := exchangeStatisticServer.Context()

	for {
		select {
		case timetick := <-ticker.C:
			//fmt.Printf("TICK\n")
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
					//fmt.Printf("NEW TICKER IN INTERVAL %v\n", v)
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

				//fmt.Printf("SAME TICKER IN INTERVAL %v\n", v)

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

			//fmt.Printf("TICKERS AGGREGATE %v\n", ohlcvs)

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

// Adds new Order from broker to OrderBook and returns assigned unique DealID
func (e *ExchangeSrv) Create(ctx context.Context, deal *exchange.Deal) (*exchange.DealID, error) {
	fmt.Printf("new order received: %v\n", deal)
	//deal.ID = atomic.AddInt64(&e.MaxDealID, 1)
	deal.ID = int64(uuid.New().ID()) // since there is no persistence for exchange implemented

	e.OrderBookLock.Lock()
	e.OrderBook = append(e.OrderBook, deal)
	fmt.Printf("order book now is: %v\n", e.OrderBook)
	e.OrderBookLock.Unlock()

	dealid := &exchange.DealID{
		ID:       deal.ID,
		BrokerID: int64(deal.BrokerID),
	}

	fmt.Printf("returning dealid: %v\n", dealid)
	return dealid, nil
}

// Cancels existing deal or returns an error if deal does not exist
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
	fmt.Printf("Broker connected to Results, brokerId: %v\n", brokerID)
	c, err := e.GetBrokerChannel(brokerID)
	if err != nil {
		return err
	}

	for {
		d := <-c
		errsend := exchangeResultsServer.Send(d)
		if errsend != nil {
			fmt.Printf("Error sending Results: %v", errsend)
		}
	}

}

func (e *ExchangeSrv) GetBrokerChannel(brokerId *exchange.BrokerID) (chan *exchange.Deal, error) {
	e.ChannelsLock.Lock()
	defer e.ChannelsLock.Unlock()

	val, ok := e.Channels[brokerId.ID]
	if ok {
		return val, nil
	} else {
		e.Channels[brokerId.ID] = make(chan *exchange.Deal, e.BufferSize)
		return e.Channels[brokerId.ID], nil
	}
}

func (e *ExchangeSrv) StartTrader() error {
	fmt.Println("Starting trader...")

	go func() error {
		interval := time.Second * 5
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case timetick := <-ticker.C:
				tickers, err := e.Tickers.GetTickersBeforeTS(timetick, interval)
				if err != nil {
					return err
				}
				//fmt.Printf("TRADER: %v\n", tickers)

				e.OrderBookLock.Lock()
				for i, order := range e.OrderBook {
					fmt.Printf("TRADER ORDER: %v\n", order)
					if order.Volume == 0 {
						e.OrderBook = append(e.OrderBook[:i], e.OrderBook[i+1:]...)
						continue
					}

					for j, t := range tickers {
						fmt.Printf("TRADER TICKER: %v\n", t)
						if t.Ticker != order.Ticker || t.Vol == 0 {
							continue
						}

						// exchange sells, broker buys
						if order.Price > 0 && order.Price > t.Last {
							fmt.Printf("TRADER SELLS ORDER VOL %v\n", order.Volume)
							var dealvol int32
							var p bool
							if order.Volume >= t.Vol {
								dealvol = t.Vol
								p = true
							} else {
								dealvol = order.Volume
							}
							e.OrderBook[i].Volume -= dealvol
							tickers[j].Vol -= dealvol

							c, err := e.GetBrokerChannel(&exchange.BrokerID{
								ID: int64(order.BrokerID),
							})
							if err != nil {
								fmt.Printf("Error getting broker channel: %v\n", err)
							}
							d1 := &exchange.Deal{
								ID:       order.ID,
								BrokerID: order.BrokerID,
								ClientID: order.ClientID,
								Ticker:   order.Ticker,
								Volume:   dealvol,
								Partial:  p,
								Time:     int32(t.Timestamp.Unix()),
								Price:    t.Last,
							}
							c <- d1
							break
						}

						// exchange buys, broker sells
						if order.Price < 0 && -order.Price < t.Last {
							fmt.Printf(" - TRADER BUYS\n")
							fmt.Printf("ORDER VOL %v\n", order.Volume)
							var dealvol int32
							var p bool
							dealvol = order.Volume

							e.OrderBook[i].Volume -= dealvol
							tickers[j].Vol += dealvol

							c, err := e.GetBrokerChannel(&exchange.BrokerID{
								ID: int64(order.BrokerID),
							})
							if err != nil {
								fmt.Printf("Error getting broker channel: %v\n", err)
							}
							d1 := &exchange.Deal{
								ID:       order.ID,
								BrokerID: order.BrokerID,
								ClientID: order.ClientID,
								Ticker:   order.Ticker,
								Volume:   -dealvol,
								Partial:  p,
								Time:     int32(t.Timestamp.Unix()),
								Price:    t.Last,
							}
							c <- d1
							break
						}
					}
				}
				e.OrderBookLock.Unlock()
			}
		}
	}()

	return nil
}
