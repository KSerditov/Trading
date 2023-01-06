package tickers

import "time"

type Tick struct {
	Ticker    string
	Timestamp time.Time
	Last      float32
	Vol       int32
}

type TickersSource interface {
	GetFeedChannel() <-chan Tick
	CloseFeed()
}
