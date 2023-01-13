package brokerclient

import "github.com/KSerditov/Trading/pkg/broker/orders"

type BrokerClient interface {
	Register(userid string) error
	History(ticker string, userid string) (*orders.HistoryResponse, error)
	Positions(userid string) (*orders.StatusResponse, error)
	Deal(deal *orders.Deal, userid string) (*orders.DealIdResponse, error)
	Cancel(dealid int64, userid string) (bool, error)
}
