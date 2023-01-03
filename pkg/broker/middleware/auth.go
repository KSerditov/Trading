package middleware

import (
	"context"
	"encoding/json"
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

var (
	noAuthUrls = map[string]struct{}{
		"/register":        {},
		"/login":           {},
		"/":                {},
		"/api/v1/register": {},
		"/api/v1//login":   {},
	}
)

/*
Auth - проверяет наличие и валидность jwt у входящего запроса
*/
func (a *AuthHandler) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := noAuthUrls[r.URL.Path]; ok {
			next.ServeHTTP(w, r)
			return
		}

		token := ""
		// get jwt token from auth header
		auth := r.Header.Get("Authorization")
		if auth != "" {
			splitToken := strings.Split(auth, "Bearer")
			if len(splitToken) < 2 {
				a.jsonMsg(w, r, "cant retrieve token", http.StatusUnauthorized)
				return
			}
			token = strings.TrimSpace(splitToken[1])
		} else { // try to get from cookie
			sessionCookie, err := r.Cookie("session")
			if err != http.ErrNoCookie {
				token = sessionCookie.Value
			}
		}

		if token == "" {
			a.jsonMsg(w, r, "no token provided", http.StatusUnauthorized)
			return
		}

		// parse token
		claims, err := a.SessMgr.GetJWTClaimsFromToken(token)
		if err != nil {
			a.jsonMsg(w, r, err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, session.ClaimsContextKey{}, claims)

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
