package handlers

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/KSerditov/Trading/pkg/broker/orders"
	"github.com/KSerditov/Trading/pkg/broker/session"
	"github.com/KSerditov/Trading/pkg/broker/user"

	"go.uber.org/zap"
)

var (
	tickersTabs = []string{"SPFB.RTS", "SPFB.Si"}
)

type UserClientHandler struct {
	BrokerBaseUrl string

	Tmpl   *template.Template
	Logger *zap.SugaredLogger

	UserAPI   *UserHandlers
	OrdersAPI *OrderHandlers
}

type OhlcvString struct {
	Open   string
	High   string
	Low    string
	Close  string
	Volume string
	Time   string
}

func (u *UserClientHandler) Positions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, err0 := u.UserAPI.SessMgr.GetSessionFromContext(ctx)
	if err0 != nil {
		u.Error(w, r, err0.Error())
		return
	}

	balance, err1 := u.OrdersAPI.OrdersRepo.GetBalance(sess.UserID)
	if err1 != nil {
		u.Error(w, r, err1.Error())
		return
	}

	positions, err2 := u.OrdersAPI.OrdersRepo.GetPositionsByUserId(sess.UserID)
	if err2 != nil {
		u.Error(w, r, err2.Error())
		return
	}

	deals, err3 := u.OrdersAPI.OrdersRepo.GetDealsByUserId(sess.UserID)
	if err3 != nil {
		u.Error(w, r, err3.Error())
		return
	}

	err := u.Tmpl.ExecuteTemplate(w, "positions.html", struct {
		Positions []orders.Position
		Deals     []orders.Deal
		Balance   string
	}{
		Positions: positions,
		Deals:     deals,
		Balance:   fmt.Sprint(balance),
	})
	if err != nil {
		u.Logger.Error("ExecuteTemplate err", err)
		http.Error(w, `Template errror`, http.StatusInternalServerError)
		return
	}
}

func (u *UserClientHandler) Deal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		err := u.Tmpl.ExecuteTemplate(w, "deal.html", struct {
			PostResult []string
		}{
			PostResult: make([]string, 0),
		})
		if err != nil {
			u.Logger.Error("ExecuteTemplate err", err)
			http.Error(w, `Template errror`, http.StatusInternalServerError)
			return
		}
		return
	}

	ctx := r.Context()
	sess, err0 := u.UserAPI.SessMgr.GetSessionFromContext(ctx)
	if err0 != nil {
		u.Error(w, r, err0.Error())
		return
	}

	//data validations
	postResult := make([]string, 0)

	tickerRaw := r.FormValue("ticker")
	if tickerRaw == "" {
		postResult = append(postResult, "Ticker cannot be empty")
	}

	typeRaw := r.FormValue("type")
	if tickerRaw == "" {
		postResult = append(postResult, "Type cannot be empty")
	}

	var volume int32
	volumeRaw := r.FormValue("volume")
	v, err := strconv.ParseInt(volumeRaw, 10, 32)
	if err != nil {
		postResult = append(postResult, "Volume value should be int32")
	} else {
		volume = int32(v)
	}

	var price int32
	priceRaw := r.FormValue("price")
	p, err1 := strconv.ParseInt(priceRaw, 10, 32)
	if err1 != nil {
		postResult = append(postResult, "Price value should be int32")
	} else {
		price = int32(p)
	}

	deal := &orders.Deal{
		Ticker: tickerRaw,
		Type:   typeRaw,
		Volume: volume,
		Price:  price,
	}

	if len(postResult) == 0 {
		dealid, _, derr := u.OrdersAPI.CreateDeal(sess.UserID, deal)
		if derr != nil {
			postResult = append(postResult, fmt.Sprintf("Error creating new deal: %v\n", derr.Error()))
		} else {
			postResult = append(postResult, fmt.Sprintf("Deal created %v\n", dealid))
		}
	}

	errd := u.Tmpl.ExecuteTemplate(w, "deal.html", struct {
		PostResult []string
	}{
		PostResult: postResult,
	})
	if errd != nil {
		u.Logger.Error("ExecuteTemplate err", errd)
		http.Error(w, `Template error`, http.StatusInternalServerError)
		return
	}
}

func (u *UserClientHandler) History(w http.ResponseWriter, r *http.Request) {
	ticker := r.URL.Query().Get("ticker")
	if ticker == "" {
		ticker = `SPFB.RTS`
	}

	timelimit := time.Now().Add(-time.Duration(u.OrdersAPI.HistoryDepthMin) * time.Minute)
	ohlcvs, err := u.OrdersAPI.OrdersRepo.GetStatisticSince(timelimit, ticker)
	if err != nil {
		u.Error(w, r, err.Error())
		return
	}
	elems := make([]OhlcvString, 0, len(ohlcvs))
	for _, v := range ohlcvs {
		e := OhlcvString{
			Open:   fmt.Sprintf("%.2f", v.Open),
			High:   fmt.Sprintf("%.2f", v.High),
			Low:    fmt.Sprintf("%.2f", v.Low),
			Close:  fmt.Sprintf("%.2f", v.Close),
			Volume: fmt.Sprint(v.Volume),
			Time:   time.Unix(int64(v.Time), 0).Format("2006-01-02 15:04:05"),
		}
		elems = append(elems, e)
	}

	err1 := u.Tmpl.ExecuteTemplate(w, "history.html", struct {
		Items      []OhlcvString
		TickerTabs []string
		Ticker     string
	}{
		Items:      elems,
		TickerTabs: tickersTabs,
		Ticker:     ticker,
	})
	if err1 != nil {
		u.Logger.Error("ExecuteTemplate err", err1)
		http.Error(w, `Template errror`, http.StatusInternalServerError)
		return
	}
}

func (u *UserClientHandler) Index(w http.ResponseWriter, r *http.Request) {
	var token string
	sessionCookie, err := r.Cookie("session")
	if err != http.ErrNoCookie {
		token = sessionCookie.Value
		claims, err := u.UserAPI.SessMgr.GetJWTClaimsFromToken(token)
		if err != nil {
			u.Error(w, r, err.Error())
			return
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, session.ClaimsContextKey{}, claims)
		r = r.WithContext(ctx)

		_, err2 := u.UserAPI.SessMgr.GetSessionFromContext(r.Context())
		if err2 == nil {
			http.Redirect(w, r, "/positions", http.StatusFound)
			return
		}
	}

	err = u.Tmpl.ExecuteTemplate(w, "login.html", nil)
	if err != nil {
		http.Error(w, `Template errror`, http.StatusInternalServerError)
		return
	}
}

func (u *UserClientHandler) Login(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.FormValue("username"))
	lf := &user.LoginForm{
		Username: r.FormValue("username"),
		Password: r.FormValue("password"),
	}

	token, err := u.UserAPI.Authorize(lf)
	if err != nil {
		u.Logger.Errorw("failed to read api login response")
		u.Error(w, r, "authorization failed")
		return
	}

	cookie := &http.Cookie{
		Name:     "session",
		Value:    token,
		Expires:  time.Now().Add(90 * 24 * time.Hour),
		Path:     "/",
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)

	r.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	http.Redirect(w, r, "/", http.StatusFound)
}

func (u *UserClientHandler) Logout(w http.ResponseWriter, r *http.Request) {
	u.UserAPI.SessMgr.DestroyCurrent(w, r)

	cookie := &http.Cookie{
		Name:    "session",
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),

		HttpOnly: true,
	}

	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (u *UserClientHandler) Error(w http.ResponseWriter, r *http.Request, msg string) {
	err := u.Tmpl.ExecuteTemplate(w, "error.html", msg)
	if err != nil {
		http.Error(w, `Template error`, http.StatusInternalServerError)
		return
	}
}
