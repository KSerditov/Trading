package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/KSerditov/Trading/api/exchange"
	"github.com/KSerditov/Trading/pkg/broker/custlog"
	"github.com/KSerditov/Trading/pkg/broker/exchclient"
	"github.com/KSerditov/Trading/pkg/broker/orders"
	"github.com/KSerditov/Trading/pkg/broker/session"
)

type OrderHandlers struct {
	SessMgr    *session.JWTSessionManager
	OrdersRepo orders.OrdersRepository
	ExchClient exchclient.OrderExchClient

	ClientID        int32
	HistoryDepthMin int32
}

func (o *OrderHandlers) CreateDeal(userid string, deal *orders.Deal) (*exchange.DealID, int, error) {
	switch strings.ToLower(deal.Type) {
	case "buy":
		balance, err := o.OrdersRepo.GetBalance(userid)
		if err != nil {
			return nil, http.StatusInternalServerError, errors.New("unable to retrieve user balance")
		}
		if balance < deal.Price*deal.Volume {
			return nil, http.StatusBadRequest, errors.New("insufficient balance to put buy request")
		}
	case "sell":
		position, err := o.OrdersRepo.GetPositionByUserId(userid, deal.Ticker)
		if err != nil {
			return nil, http.StatusInternalServerError, errors.New("unable to retrieve user positions")
		}
		if position.Volume < deal.Volume {
			return nil, http.StatusInternalServerError, errors.New("not enough volume for position to put sell request")
		}
	default:
		return nil, http.StatusBadRequest, errors.New("deal type can be buy or sell only")
	}

	dealid, err := o.ExchClient.CreateDeal(deal.Ticker, deal.Volume, float32(deal.Price), o.ClientID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	deal.Id = dealid.ID

	fmt.Printf("NEW DEAL: %v\n", deal)

	_, errr := o.OrdersRepo.AddDeal(userid, *deal)
	if errr != nil {
		//log error, but request has been posted to exchange, so return OK
		custlog.CtxLog(context.TODO()).Errorw("failed to save deal to repository",
			"userid", userid,
			"deal", *deal,
			"repository error", errr.Error(),
		)
	}
	return dealid, http.StatusAccepted, nil
}

func (o *OrderHandlers) CreateDealHr(w http.ResponseWriter, r *http.Request) {
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

	dealid, statucode, err := o.CreateDeal(sess.UserID, deal)
	if err != nil {
		o.jsonMsg(w, err.Error(), statucode)
		return
	}

	d := &orders.DealIdResponse{
		Body: dealid,
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	jsonPost, _ := json.Marshal(d)
	w.Write(jsonPost)
}

func (o *OrderHandlers) CancelDeal(userid string, dealid *orders.DealId) (int, bool, error) {
	_, err1 := o.OrdersRepo.GetDealByUserAndId(userid, dealid.Id)
	if err1 != nil {
		return http.StatusBadRequest, false, errors.New("deal does not exist")
	}

	cancelled, err := o.ExchClient.CancelDeal(dealid.Id)
	if cancelled {
		errr := o.OrdersRepo.DeleteDealById(dealid.Id)
		if errr != nil {
			//log error, but request has been posted to exchange, so return OK
			custlog.CtxLog(context.TODO()).Errorw("failed to delete deal from repository",
				"userid", userid,
				"deal", dealid.Id,
			)
		}
		return http.StatusOK, true, nil
	}

	return http.StatusOK, false, err
}

func (o *OrderHandlers) CancelDealHr(w http.ResponseWriter, r *http.Request) {
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

	cancelResp := &orders.CancelResponse{
		Body: &orders.AllOfCancelResponseBody{
			Id:     dealid.Id,
			Status: "Success",
		},
	}
	statuscode, cancelled, err := o.CancelDeal(sess.UserID, dealid)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if cancelled {
		w.WriteHeader(http.StatusOK)
		jsonPost, _ := json.Marshal(cancelResp)
		w.Write(jsonPost)
		return
	}

	cancelResp.Body.Status = err.Error()
	w.WriteHeader(statuscode)
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
