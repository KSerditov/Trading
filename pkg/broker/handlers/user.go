package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/KSerditov/Trading/pkg/broker/custlog"
	"github.com/KSerditov/Trading/pkg/broker/orders"
	"github.com/KSerditov/Trading/pkg/broker/session"
	"github.com/KSerditov/Trading/pkg/broker/user"
)

type UserHandlers struct {
	SessMgr    *session.JWTSessionManager
	UserRepo   user.UserRepository
	OrdersRepo orders.OrdersRepository
}

type ErrorDetails struct {
	Location string `json:"location"`
	Param    string `json:"param"`
	Value    string `json:"value"`
	Msg      string `json:"msg"`
}

func (u *UserHandlers) performLogin(lf *user.LoginForm, user user.User, w http.ResponseWriter, r *http.Request) {
	custlog.CtxLog(r.Context()).Debugw("Performing login", "user_id", user.ID)
	ok, pwderr := u.UserRepo.ValidatePassword(user, lf.Password)
	if pwderr != nil {
		u.jsonMsg(w, pwderr.Error(), http.StatusUnauthorized)
		return
	}
	if !ok {
		u.jsonMsg(w, "invalid password", http.StatusUnauthorized)
		return
	}
	custlog.CtxLog(r.Context()).Debugw("Password validated", "user_id", user.ID)

	tokenString, sess, err := u.SessMgr.GetNewToken(user)
	if err != nil {
		u.jsonMsg(w, err.Error(), http.StatusUnauthorized)
		return
	}

	custlog.CtxLog(r.Context()).Debugw("new token issued", "token", tokenString)

	resp, _ := json.Marshal(map[string]interface{}{
		"token": tokenString,
	})
	w.Write(resp)
	w.Write([]byte("\n\n"))

	custlog.CtxLog(r.Context()).Infow("login success", "session", sess)
}

func (u *UserHandlers) Login(w http.ResponseWriter, r *http.Request) {
	custlog.CtxLog(r.Context()).Debugw("login handler started")
	defer custlog.CtxLog(r.Context()).Debugw("login handler completed")

	if r.Header.Get("Content-Type") != "application/json" {
		u.jsonMsg(w, "unknown payload", http.StatusBadRequest)
		return
	}

	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()

	lf := &user.LoginForm{}

	err := json.Unmarshal(body, lf)
	if err != nil {
		u.jsonMsg(w, "cant unpack payload", http.StatusBadRequest)
		return
	}

	user, err2 := u.UserRepo.GetUser(lf.Username)
	if err2 != nil {
		u.jsonMsg(w, err2.Error(), http.StatusUnauthorized)
		return
	}

	u.performLogin(lf, user, w, r)
}

func (u *UserHandlers) Register(w http.ResponseWriter, r *http.Request) {
	custlog.CtxLog(r.Context()).Debugw("registration handler started")
	defer custlog.CtxLog(r.Context()).Debugw("registration handler completed")

	if r.Header.Get("Content-Type") != "application/json" {
		u.jsonMsg(w, "unknown payload", http.StatusBadRequest)
		return
	}

	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()

	lf := &user.LoginForm{}
	err := json.Unmarshal(body, lf)
	if err != nil {
		u.jsonMsg(w, "cant unpack payload", http.StatusBadRequest)
		return
	}

	user, err := u.UserRepo.AddUser(lf.Username, lf.Password)
	if err != nil {
		errDet := ErrorDetails{
			Location: "body",
			Param:    "username",
			Value:    lf.Username,
			Msg:      err.Error(),
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		err, _ := json.Marshal(map[string]interface{}{
			"errors": []ErrorDetails{errDet},
		})
		w.Write(err)
		return
	}

	_, errbalance := u.OrdersRepo.ChangeBalance(user.ID, 100500)
	if errbalance != nil {
		errDet := ErrorDetails{
			Location: "body",
			Param:    "username",
			Value:    lf.Username,
			Msg:      errbalance.Error(),
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		err, _ := json.Marshal(map[string]interface{}{
			"errors": []ErrorDetails{errDet},
		})
		w.Write(err)
		return
	}

	custlog.CtxLog(r.Context()).Infow("registration success", "userid", user.ID, "username", user.Username)

	u.performLogin(lf, user, w, r)
}

func (u *UserHandlers) jsonMsg(w http.ResponseWriter, msg string, status int) {
	w.WriteHeader(status)
	resp, _ := json.Marshal(map[string]interface{}{
		"message": msg,
	})
	w.Write(resp)
}
