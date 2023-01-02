package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"strings"

	"github.com/KSerditov/Trading/pkg/broker/custlog"
	"github.com/KSerditov/Trading/pkg/broker/session"
	"github.com/KSerditov/Trading/pkg/broker/user"
)

type AuthHandler struct {
	SessMgr  *session.JWTSessionManager
	UserRepo user.UserRepository
}

/*
Auth - проверяет наличие и валидность jwt у входящего запроса
*/
func (a *AuthHandler) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		custlog.CtxLog(r.Context()).Debugw("authorization middleware started")
		defer custlog.CtxLog(r.Context()).Debugw("authorization middleware completed")

		auth := r.Header.Get("Authorization")
		splitToken := strings.Split(auth, "Bearer")
		if len(splitToken) < 2 {
			a.jsonMsg(w, r, "cant retrieve token", http.StatusUnauthorized)
			return
		}
		reqToken := strings.TrimSpace(splitToken[1])
		fmt.Printf("reqToken %v\n", reqToken)
		claims, err := a.SessMgr.GetJWTClaimsFromToken(reqToken)
		if err != nil {
			a.jsonMsg(w, r, err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, session.ClaimsContextKey{}, claims) //TBD i don't need both, need to remove user from context

		custlog.CtxLog(r.Context()).Infow("request authorization success",
			"session", claims.Sid,
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
			"url", r.URL.Path,
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *AuthHandler) jsonMsg(w http.ResponseWriter, r *http.Request, msg string, status int) {
	custlog.CtxLog(r.Context()).Infow("request authorization failure",
		"method", r.Method,
		"remote_addr", r.RemoteAddr,
		"url", r.URL.Path,
		"status_code", status,
		"json_msg", msg,
	)
	w.WriteHeader(status)
	resp, _ := json.Marshal(map[string]interface{}{
		"message": msg,
	})
	w.Write(resp)
}
