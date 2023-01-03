package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/KSerditov/Trading/api/exchange"
	"github.com/KSerditov/Trading/pkg/broker/custlog"
	"github.com/KSerditov/Trading/pkg/broker/orders"
	"github.com/KSerditov/Trading/pkg/broker/session"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type OrderHandlers struct {
	SessMgr    *session.JWTSessionManager
	OrdersRepo orders.OrdersRepository

	ExchServerAddress string
	BrokerID          int32
	ClientID          int32
	HistoryDepthMin   int32
}

func (o *OrderHandlers) CreateDeal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, _ := o.SessMgr.GetSessionFromContext(ctx)

	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()
	deal := &orders.Deal{}
	err := json.Unmarshal(body, deal)
	if err != nil {
		o.jsonMsg(w, "cant unpack payload", http.StatusBadRequest)
		return
	}

	switch strings.ToLower(deal.Type) {
	case "buy":
		balance, err := o.OrdersRepo.GetBalance(sess.UserID)
		if err != nil {
			o.jsonMsg(w, "unable to retrieve user balance", http.StatusInternalServerError)
			return
		}

		if balance < deal.Price*deal.Volume {
			o.jsonMsg(w, "insufficient balance to put buy request", http.StatusBadRequest)
			return
		}
	case "sell":
		position, err := o.OrdersRepo.GetPositionByUserId(sess.UserID, deal.Ticker)
		if err != nil {
			o.jsonMsg(w, "unable to retrieve user positions", http.StatusInternalServerError)
			return
		}
		if position.Volume < deal.Volume {
			o.jsonMsg(w, "not enough volume for position to put sell request", http.StatusBadRequest)
			return
		}
	default:
		o.jsonMsg(w, "deal type can be buy or sell only", http.StatusBadRequest)
		return
	}

	grcpConn, err := grpc.Dial(
		o.ExchServerAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		o.jsonMsg(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer grcpConn.Close()

	c := exchange.NewExchangeClient(grcpConn)
	ctx1 := context.Background()
	exchdeal := &exchange.Deal{
		BrokerID: o.BrokerID,
		ClientID: o.ClientID,
		Ticker:   deal.Ticker,
		Volume:   deal.Volume,
		Partial:  false,
		Time:     int32(time.Now().Unix()),
		Price:    float32(deal.Price),
	}
	dealid, err := c.Create(ctx1, exchdeal)
	if err != nil {
		o.jsonMsg(w, err.Error(), http.StatusInternalServerError)
		return
	}
	deal.Id = dealid.ID

	fmt.Println("NEW DEAL:")
	fmt.Println(deal)

	_, errr := o.OrdersRepo.AddDeal(sess.UserID, *deal)
	if errr != nil {
		//log error, but request has been posted to exchange, so return OK
		custlog.CtxLog(ctx).Errorw("failed to save deal to repository",
			"session", sess,
			"userid", sess.UserID,
			"deal", *deal,
			"repository error", errr.Error(),
		)
	}

	d := &orders.DealIdResponse{
		Body: dealid,
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	jsonPost, _ := json.Marshal(d)
	w.Write(jsonPost)
}

func (o *OrderHandlers) CancelDeal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, _ := o.SessMgr.GetSessionFromContext(ctx)

	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()

	dealid := &orders.DealId{}
	err := json.Unmarshal(body, dealid)
	if err != nil {
		o.jsonMsg(w, "cant unpack payload", http.StatusBadRequest)
		return
	}

	_, err1 := o.OrdersRepo.GetDealByUserAndId(sess.UserID, dealid.Id)
	if err1 != nil {
		o.jsonMsg(w, "deal does not exist", http.StatusBadRequest)
		return
	}

	grcpConn, err := grpc.Dial(
		o.ExchServerAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		o.jsonMsg(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer grcpConn.Close()

	c := exchange.NewExchangeClient(grcpConn)
	ctx1 := context.Background()
	did := &exchange.DealID{
		ID:       dealid.Id,
		BrokerID: int64(o.BrokerID),
	}

	cancel, err := c.Cancel(ctx1, did)
	cancelResp := &orders.CancelResponse{
		Body: &orders.AllOfCancelResponseBody{
			Id:     dealid.Id,
			Status: "Success",
		},
	}
	if cancel.Success {
		errr := o.OrdersRepo.DeleteDealById(dealid.Id)
		if errr != nil {
			//log error, but request has been posted to exchange, so return OK
			custlog.CtxLog(ctx).Errorw("failed to delete deal from repository",
				"session", sess,
				"userid", sess.UserID,
				"deal", dealid.Id,
			)
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		jsonPost, _ := json.Marshal(cancelResp)
		w.Write(jsonPost)
		return
	}

	if err != nil {
		cancelResp.Body.Status = fmt.Sprintf("error canceling deal: %v", err.Error())
	} else {
		cancelResp.Body.Status = "error canceling deal, no error returned from exchange"
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusBadRequest)
	jsonPost, _ := json.Marshal(cancelResp)
	w.Write(jsonPost)
}

func (o *OrderHandlers) GetHistory(w http.ResponseWriter, r *http.Request) {
	ticker := r.URL.Query().Get("ticker")
	if ticker == "" {
		o.jsonMsg(w, "ticker param not provided", http.StatusBadRequest)
		return
	}

	timelimit := time.Now().Add(-time.Duration(o.HistoryDepthMin) * time.Minute)
	ohlcvs, err := o.OrdersRepo.GetStatisticSince(timelimit, ticker)
	if err != nil {
		o.jsonMsg(w, err.Error(), http.StatusInternalServerError)
		return
	}

	history := &orders.HistoryResponse{
		Body: orders.TickerOhlcv{
			Ticker: ticker,
			Prices: ohlcvs,
		},
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	jsonPost, _ := json.Marshal(history)
	w.Write(jsonPost)
}

func (o *OrderHandlers) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, _ := o.SessMgr.GetSessionFromContext(ctx)

	balance, err := o.OrdersRepo.GetBalance(sess.UserID)
	if err != nil {
		o.jsonMsg(w, err.Error(), http.StatusInternalServerError)
		return
	}
	deals, err1 := o.OrdersRepo.GetDealsByUserId(sess.UserID)
	if err1 != nil {
		o.jsonMsg(w, err.Error(), http.StatusInternalServerError)
		return
	}
	positions, err2 := o.OrdersRepo.GetPositionsByUserId(sess.UserID)
	if err2 != nil {
		o.jsonMsg(w, err.Error(), http.StatusInternalServerError)
		return
	}

	status := &orders.StatusResponse{
		Body: &orders.StatusBody{
			Balance:    balance,
			Positions:  positions,
			OpenOrders: deals,
		},
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	jsonPost, _ := json.Marshal(status)
	w.Write(jsonPost)
}

func (o *OrderHandlers) jsonMsg(w http.ResponseWriter, msg string, status int) {
	w.WriteHeader(status)
	resp, _ := json.Marshal(map[string]interface{}{
		"message": msg,
	})
	w.Write(resp)
}
