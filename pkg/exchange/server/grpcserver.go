package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/KSerditov/Trading/api/exchange"
	"github.com/KSerditov/Trading/pkg/exchange/tickers"

	"github.com/google/uuid"
	"google.golang.org/grpc"
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

func Start(ctx context.Context, listenAddr string, ACLData string, datasource tickers.TickersSource) error {
	/*
		auther := Authenticator{}
		errjson := json.Unmarshal([]byte(ACLData), &auther.accessList)
		if errjson != nil {
			return errjson
		}
	*/

	s := &ExchangeSrv{
		BufferSize:                  100,
		Tickers:                     datasource,
		MaxDealID:                   0,
		OrderBookLock:               &sync.RWMutex{},
		OrderBook:                   make([]*exchange.Deal, 0, 100),
		ChannelsLock:                &sync.RWMutex{},
		Channels:                    make(map[int64]chan *exchange.Deal, 10),
		UnimplementedExchangeServer: exchange.UnimplementedExchangeServer{},
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalln("cant listen port", err)
		return err
	}

	server := grpc.NewServer(
		grpc.ChainStreamInterceptor(
		//logStreamInterceptor,
		//auther.AuthStreamInterceptor,
		),
		grpc.ChainUnaryInterceptor(
		//logInterceptor,
		//auther.AuthInterceptor,
		),
	)

	exchange.RegisterExchangeServer(server, s)

	go func(s *grpc.Server) {
		for {
			<-ctx.Done()
			datasource.CloseFeed()
			s.GracefulStop()
			return
		}
	}(server)

	fmt.Println("Starting exchange server...")

	s.StartTrader()

	errs := server.Serve(lis)
	if errs != nil {
		return errs
	}

	return nil
}

// поток ценовых данных от биржи к брокеру
// мы каждую секнуду будем получать отсюда событие с ценами, которые брокер аггрегирует у себя и показывает клиентам
// устанавливается 1 раз брокером
func (e *ExchangeSrv) Statistic(brokerID *exchange.BrokerID, exchangeStatisticServer exchange.Exchange_StatisticServer) error {
	fmt.Printf("Broker connected to Statistic, brokerId: %v\n", brokerID)
	interval := time.Second * 1
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	ctx := exchangeStatisticServer.Context()
	//fmt.Printf("STATISTICS requesting feed channel\n")
	feed := e.Tickers.GetFeedChannel()

	var opents, closets time.Time
	ohlcvs := make(map[string]*exchange.OHLCV, 2)

	for {
		select {
		// new ticker from feed - collect data into ohlcv map per each ticker value
		case v := <-feed:
			//fmt.Printf("STATISTICS NEW TICKER FROM FEED %v\n", v)
			_, ok := ohlcvs[v.Ticker]
			if !ok { // add new ticker first time in interval
				opents = v.Timestamp
				closets = v.Timestamp

				newOHLCV := &exchange.OHLCV{
					ID:       atomic.AddInt64(&e.ohlcvId, 1),
					Time:     int32(v.Timestamp.Unix()),
					Interval: int32(interval.Seconds()),
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

			// aggregation
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

			//fmt.Printf("STATISTICS AGGREGATE %v\n", ohlcvs[v.Ticker])

		// broker notification interval elapsed - send collected data
		case timetick := <-ticker.C:
			//fmt.Printf("STATISTICS NEW TIME TICK\n")
			for _, v := range ohlcvs {
				v.Time = int32(timetick.Unix())
				//fmt.Printf("STATISTICS SENDING %v\n", v)
				errsend := exchangeStatisticServer.Send(v)

				if errsend != nil {
					return errsend
				}
			}
			ohlcvs = make(map[string]*exchange.OHLCV, 2)

		case <-ctx.Done():
			return nil
		}
	}
}

// Adds new Order from broker to OrderBook and returns assigned unique DealID
func (e *ExchangeSrv) Create(ctx context.Context, deal *exchange.Deal) (*exchange.DealID, error) {
	//fmt.Printf("new order received: %v\n", deal)
	//deal.ID = atomic.AddInt64(&e.MaxDealID, 1)
	deal.ID = int64(uuid.New().ID()) // since there is no persistence for exchange yet

	e.OrderBookLock.Lock()
	e.OrderBook = append(e.OrderBook, deal)
	//fmt.Printf("order book now is: %v\n", e.OrderBook)
	e.OrderBookLock.Unlock()

	dealid := &exchange.DealID{
		ID:       deal.ID,
		BrokerID: int64(deal.BrokerID),
	}

	//fmt.Printf("returning dealid: %v\n", dealid)
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
	defer e.DeleteBrokerChannel(brokerID)

	//fmt.Printf("Broker connected to Results, brokerId: %v\n", brokerID)
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

func (e *ExchangeSrv) DeleteBrokerChannel(brokerId *exchange.BrokerID) {
	e.ChannelsLock.Lock()
	defer e.ChannelsLock.Unlock()

	delete(e.Channels, brokerId.ID)
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

	go func() {
		feed := e.Tickers.GetFeedChannel()
		for t := range feed {
			// new ticker received from ticker feed
			//fmt.Printf("TRADER TICKER: %v\n", t)
			if t.Vol == 0 {
				continue
			}

			e.OrderBookLock.Lock()

			// go through incomplete orders
			for i, order := range e.OrderBook {
				// drop completed orders or orders with 0 price
				if order.Volume == 0 || order.Price == 0 {
					e.OrderBook = append(e.OrderBook[:i], e.OrderBook[i+1:]...)
					continue
				}

				fmt.Printf("TRADER ORDER: %v\n", order)
				if order.Ticker != t.Ticker {
					continue
				}

				c, err := e.GetBrokerChannel(&exchange.BrokerID{
					ID: int64(order.BrokerID),
				})
				if err != nil {
					fmt.Printf("Error getting broker channel: %v\n", err)
				}

				// prepare deal
				deal := &exchange.Deal{
					ID:       order.ID,
					BrokerID: order.BrokerID,
					ClientID: order.ClientID,
					Ticker:   order.Ticker,
					Time:     int32(t.Timestamp.Unix()),
					Price:    t.Last,
					Partial:  false,
				}

				if order.Volume > t.Vol {
					deal.Volume = t.Vol
					deal.Partial = true
				} else {
					deal.Volume = order.Volume
				}

				// make deal if price conditions are met
				// pending deal price exceeds ticker from feed, then exchange sells, broker buys
				// positive price expected if pending deal has BUY type
				if order.Price > 0 && order.Price >= t.Last {
					fmt.Printf("TRADER SELLS ORDER VOL %v\n", order.Volume)

					e.OrderBook[i].Volume -= deal.Volume
					t.Vol -= deal.Volume

					fmt.Printf("TRADER SOLD %v\n", deal)

					c <- deal
					continue
				}

				// exchange buys, broker sells
				// negative price expected if pending deal has SELL type
				//
				if order.Price < 0 && -order.Price <= t.Last {
					fmt.Printf("TRADER BUYS ORDER VOL %v\n", order.Volume)

					e.OrderBook[i].Volume -= deal.Volume
					t.Vol += deal.Volume

					fmt.Printf("TRADER BOUGHT %v\n", deal)

					c <- deal
					continue
				}
			}

			e.OrderBookLock.Unlock()
		}
	}()

	fmt.Println("Trader started...")
	return nil
}
