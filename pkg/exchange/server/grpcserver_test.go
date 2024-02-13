package server

import (
	"context"
	"io"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/KSerditov/Trading/api/exchange"
	"github.com/KSerditov/Trading/pkg/exchange/tickers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	listenAddr string = "127.0.0.1:8082"
)

func getGrpcConn(t *testing.T) *grpc.ClientConn {
	grcpConn, err := grpc.Dial(
		listenAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("cant connect to grpc: %v", err)
	}
	return grcpConn
}

type TickersSourceTest struct {
	chLock *sync.RWMutex
	ch     []chan tickers.Tick
}

func (t *TickersSourceTest) GetFeedChannel() <-chan tickers.Tick {
	c := make(chan tickers.Tick, 100)

	t.chLock.Lock()
	t.ch = append(t.ch, c)
	t.chLock.Unlock()

	return c
}

func (t *TickersSourceTest) CloseFeed() {
	t.chLock.Lock()
	defer t.chLock.Unlock()

	for _, c := range t.ch {
		close(c)
	}
}

func (t *TickersSourceTest) Run(tickers []tickers.Tick) {
	for _, v := range tickers {
		for _, c := range t.ch {
			c <- v
		}
	}
}

type PlainOHLCV struct {
	Open   float32
	High   float32
	Low    float32
	Close  float32
	Volume int32
	Ticker string
}

type StatTests struct {
	tickers  []tickers.Tick
	expected PlainOHLCV
}

var (
	stattests = []StatTests{
		{
			tickers: []tickers.Tick{
				{
					Ticker:    "SPFB.RTS",
					Timestamp: time.Now().Add(time.Second * 3),
					Last:      100,
					Vol:       1,
				},
			},
			expected: PlainOHLCV{
				Open:   100,
				High:   100,
				Low:    100,
				Close:  100,
				Volume: 1,
				Ticker: "SPFB.RTS",
			},
		},
		{
			tickers: []tickers.Tick{
				{
					Ticker:    "SPFB.RTS",
					Timestamp: time.Now().Add(time.Second * 2),
					Last:      100,
					Vol:       1,
				},
				{
					Ticker:    "SPFB.RTS",
					Timestamp: time.Now().Add(time.Second * 3),
					Last:      50,
					Vol:       3,
				},
			},
			expected: PlainOHLCV{
				Open:   100,
				High:   100,
				Low:    50,
				Close:  50,
				Volume: 4,
				Ticker: "SPFB.RTS",
			},
		},
	}
)

func wait(amout int) {
	time.Sleep(time.Duration(amout) * 10 * time.Millisecond)
}

func TestStat(t *testing.T) {
	ts := &TickersSourceTest{
		chLock: &sync.RWMutex{},
		ch:     make([]chan tickers.Tick, 0, 2),
	}

	ctx, finish := context.WithCancel(context.Background())
	go Start(ctx, listenAddr, ``, ts)

	conn := getGrpcConn(t)
	defer conn.Close()

	brokerid := exchange.BrokerID{
		ID: 123,
	}
	exch := exchange.NewExchangeClient(conn)
	statStream1, err := exch.Statistic(context.Background(), &brokerid)
	if err != nil {
		t.Fatalf("cant get stat stream: %v", err)
	}

	wait(3)

	for j, v := range stattests {
		t.Logf("executing stat test %v\n", j)
		ohclv1 := PlainOHLCV{}
		//reading results
		//feed with data
		go func() {
			ts.Run(v.tickers)
		}()

		for i := 0; i < 1; i++ {
			stat, err := statStream1.Recv()
			if err == io.EOF {
				break
			}

			ohclv1 = PlainOHLCV{
				Open:   stat.Open,
				High:   stat.High,
				Low:    stat.Low,
				Close:  stat.Close,
				Volume: stat.Volume,
				Ticker: stat.Ticker,
			}
		}

		if !reflect.DeepEqual(ohclv1, v.expected) {
			t.Fatalf("ohclv1 dont match\nhave %+v\nwant %+v", ohclv1, v.expected)
		} else {
			t.Logf("stat test %v ok!\n", j)
		}

	}

	finish()
}
