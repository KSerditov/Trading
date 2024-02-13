package exchclient

import (
	"context"
	"time"

	"github.com/KSerditov/Trading/api/exchange"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type OrderExchClientGRPC struct {
	ExchServerAddress string
	BrokerID          int32

	client exchange.ExchangeClient
}

func (o *OrderExchClientGRPC) Init() error {
	grcpConn, err := grpc.Dial(
		o.ExchServerAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}

	o.client = exchange.NewExchangeClient(grcpConn)

	return nil
}

func (o *OrderExchClientGRPC) CreateDeal(ticker string, volume int32, price float32, clientid int32) (*exchange.DealID, error) {
	ctx := context.Background()
	deal := &exchange.Deal{
		BrokerID: o.BrokerID,
		ClientID: clientid,
		Ticker:   ticker,
		Volume:   volume,
		Partial:  false,
		Time:     int32(time.Now().Unix()),
		Price:    price,
	}
	dealid, err := o.client.Create(ctx, deal)
	if err != nil {
		return nil, err
	}

	return dealid, nil
}

func (o *OrderExchClientGRPC) CancelDeal(dealid int64) (bool, error) {
	ctx := context.Background()
	did := &exchange.DealID{
		ID:       dealid,
		BrokerID: int64(o.BrokerID),
	}

	cancel, err := o.client.Cancel(ctx, did)

	return cancel.Success, err
}
