package handlers

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/KSerditov/Trading/pkg/broker/orders"
	"github.com/KSerditov/Trading/pkg/broker/session"
	"github.com/KSerditov/Trading/pkg/broker/user"

	"go.uber.org/zap"
)

type UserClientHandler struct {
	BrokerBaseUrl string

	Tmpl   *template.Template
	Logger *zap.SugaredLogger

	SessMgr    *session.JWTSessionManager
	UserRepo   user.UserRepository
	OrdersRepo orders.OrdersRepository

	UserAPI *UserHandlers
}

func (u *UserClientHandler) Positions(w http.ResponseWriter, r *http.Request) {
	err := u.Tmpl.ExecuteTemplate(w, "positions.html", nil)
	if err != nil {
		u.Logger.Error("ExecuteTemplate err", err)
		http.Error(w, `Template errror`, http.StatusInternalServerError)
		return
	}
}

func (u *UserClientHandler) Index(w http.ResponseWriter, r *http.Request) {
	var token string
	sessionCookie, err := r.Cookie("session")
	if err != http.ErrNoCookie {
		token = sessionCookie.Value
		claims, err := u.SessMgr.GetJWTClaimsFromToken(token)
		if err != nil {
			u.Error(w, r, err.Error())
			return
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, session.ClaimsContextKey{}, claims)
		r = r.WithContext(ctx)

		_, err2 := u.SessMgr.GetSessionFromContext(r.Context())
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
	/*
		err = u.Tmpl.ExecuteTemplate(w, "positions.html", nil)
		if err != nil {
			http.Error(w, `Template errror`, http.StatusInternalServerError)
			return
		}*/
}

func (u *UserClientHandler) Logout(w http.ResponseWriter, r *http.Request) {
	u.SessMgr.DestroyCurrent(w, r)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (u *UserClientHandler) Error(w http.ResponseWriter, r *http.Request, msg string) {
	err := u.Tmpl.ExecuteTemplate(w, "error.html", msg)
	if err != nil {
		http.Error(w, `Template error`, http.StatusInternalServerError)
		return
	}
}
