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
	fmt.Println("Historical data load completed")

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
		Tickers:       datasource,
		MaxDealID:     0,
		OrderBookLock: &sync.RWMutex{},
		OrderBook:     make([]*exchange.Deal, 0, 100),
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
			select {
			case <-ctx.Done():
				s.GracefulStop()
				return
			}
		}
	}(server)

	fmt.Println("Starting exchange server...")
	errs := server.Serve(lis)
	if errs != nil {
		return errs
	}

	return nil
}
