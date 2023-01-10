package exchclient

import "github.com/KSerditov/Trading/api/exchange"

type OrderExchClient interface {
	CreateDeal(ticker string, volume int32, price float32, clientid int32) (*exchange.DealID, error)
	CancelDeal(dealid int64) (bool, error)
}
