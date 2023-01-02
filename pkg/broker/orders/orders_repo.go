package orders

import (
	"errors"
	"time"

	"github.com/KSerditov/Trading/api/exchange"
)

type OrdersRepository interface {
	ChangeBalance(userid string, amount int32) (int32, error)
	GetBalance(userid string) (int32, error)

	AddDeal(userid string, deal Deal) (int64, error)
	GetDealById(dealid int64) (*Deal, string, error)
	GetDealByUserAndId(userid string, dealid int64) (*Deal, error)
	GetDealsByUserId(userid string) ([]Deal, error)
	DeleteDealById(id int64) error

	AddStatisticsEntity(entity *exchange.OHLCV) (int64, error)
	GetStatisticSince(since time.Time, ticker string) ([]Ohlcv, error)

	GetPositionsByUserId(userid string) ([]Position, error)
	GetPositionByUserId(userid string, ticker string) (*Position, error)
	ChangePosition(userid string, ticker string, volumeChange int32) (*Position, error)
}

type Deal struct {
	Id     int64  `json:"id,omitempty"`
	Ticker string `json:"ticker"`
	Type   string `json:"type"`
	Volume int32  `json:"volume"`
	Price  int32  `json:"price"`
	Time   int32  `json:"time,omitempty"`
}

type DealIdResponse struct {
	Body *exchange.DealID `json:"body"`
}

type DealId struct {
	Id int64 `json:"id"`
}

type AllOfCancelResponseBody struct {
	Id     int64  `json:"id"`
	Status string `json:"status,omitempty"`
}

type CancelResponse struct {
	Body *AllOfCancelResponseBody `json:"body"`
}

type Ohlcv struct {
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int32   `json:"volume"`
	Time   int32   `json:"time"`
}

type TickerOhlcv struct {
	Ticker string  `json:"ticker"`
	Prices []Ohlcv `json:"prices"`
}

type HistoryResponse struct {
	Body TickerOhlcv `json:"body"`
}

type Position struct {
	Ticker string `json:"ticker"`
	Volume int32  `json:"volume"`
}

type StatusBody struct {
	Balance    int32      `json:"balance"`
	Positions  []Position `json:"positions"`
	OpenOrders []Deal     `json:"open_orders"`
}

type StatusResponse struct {
	Body *StatusBody `json:"body,omitempty"`
}

var (
	ErrorDealNotFound = errors.New("deal not found")
)
