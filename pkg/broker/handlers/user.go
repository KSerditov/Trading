package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
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

/*
func (u *UserHandlers) Authorize(lf *user.LoginForm, w http.ResponseWriter, r *http.Request) {
	user, err2 := u.UserRepo.GetUser(lf.Username)
	if err2 != nil {
		u.jsonMsg(w, err2.Error(), http.StatusUnauthorized)
		return
	}

	ok, pwderr := u.UserRepo.ValidatePassword(user, lf.Password)
	if pwderr != nil {
		u.jsonMsg(w, pwderr.Error(), http.StatusUnauthorized)
		return
	}
	if !ok {
		u.jsonMsg(w, "invalid password", http.StatusUnauthorized)
		return
	}

	tokenString, _, err := u.SessMgr.GetNewSession(user)
	if err != nil {
		u.jsonMsg(w, err.Error(), http.StatusUnauthorized)
		return
	}

	cookie := &http.Cookie{
		Name:     "session",
		Value:    tokenString,
		Expires:  time.Now().Add(90 * 24 * time.Hour),
		Path:     "/",
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)

	resp, _ := json.Marshal(map[string]interface{}{
		"token": tokenString,
	})
	w.Write(resp)
	w.Write([]byte("\n\n"))
}*/

func (u *UserHandlers) Authorize(lf *user.LoginForm) (string, error) {
	fmt.Println(lf)
	user, err2 := u.UserRepo.GetUser(lf.Username)
	if err2 != nil {
		return "", err2
	}

	ok, pwderr := u.UserRepo.ValidatePassword(user, lf.Password)
	if pwderr != nil {
		return "", pwderr
	}
	if !ok {
		return "", errors.New("invalid password")
	}

	tokenString, _, err := u.SessMgr.GetNewSession(user)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// swagger:route POST /login Performs login
//
// Login endpoint
//
// Returns JWT token on succesful authorization
//
// responses:
//   200: Token json
//   400: Message json
func (u *UserHandlers) Login(w http.ResponseWriter, r *http.Request) {
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

	token, err := u.Authorize(lf)

	resp, _ := json.Marshal(map[string]interface{}{
		"token": token,
	})
	w.Write(resp)
	w.Write([]byte("\n\n"))
}

// swagger:route GET /logout Performs logout
//
// Logout endpoint
//
// Deletes current user session
//
// responses:
//   200: empty response
func (u *UserHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	u.SessMgr.DestroyCurrent(w, r)
}

func (u *UserHandlers) Register(w http.ResponseWriter, r *http.Request) {
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

	token, err := u.Authorize(lf)
	if err != nil {
		u.jsonMsg(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, _ := json.Marshal(map[string]interface{}{
		"token": token,
	})
	w.Write(resp)
	w.Write([]byte("\n\n"))

}

func (u *UserHandlers) jsonMsg(w http.ResponseWriter, msg string, status int) {
	w.WriteHeader(status)
	resp, _ := json.Marshal(map[string]interface{}{
		"message": msg,
	})
	w.Write(resp)
}
