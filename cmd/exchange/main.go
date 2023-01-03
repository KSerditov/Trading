package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/KSerditov/Trading/api/exchange"
	"github.com/KSerditov/Trading/pkg/exchange/server"
	"github.com/KSerditov/Trading/pkg/exchange/tickers"
	"google.golang.org/grpc"
)

/* TBD FOR EXCHANGE
1. Use channel to translate new tickers from tickers_inmem to trader and exchange statistics sender
(tickers_inmem has infinite cycle that sends slice elements only if it fits time)

2. Add authentication (brokerid - key based?)

3. Add logging

4. Validate nonunique broker id connections

5. Write tests (consider separationg of trader layer from grpc server)

6. Add configurations

7. Last partial deal should return partial = false

8. Initialize DB instance and store everything there
*/

func main() {
	tickers := &tickers.TickersSourceInMem{
		FilePaths:    []string{`.\assets\SPFB.RTS_190517_190517.txt`, `.\assets\SPFB.Si_190517_190517.txt`},
		UseTodayDate: true,
		TickersLock:  &sync.RWMutex{},
	}
	err := tickers.Init()
	if err != nil {
		fmt.Println(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = Start(ctx, `127.0.0.1:8082`, ``, tickers)
	if err != nil {
		fmt.Println(err)
	}
}

func Start(ctx context.Context, listenAddr string, ACLData string, datasource tickers.TickersSource) error {
	/*
		auther := Authenticator{}
		errjson := json.Unmarshal([]byte(ACLData), &auther.accessList)
		if errjson != nil {
			return errjson
		}
	*/

	s := &server.ExchangeSrv{
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
