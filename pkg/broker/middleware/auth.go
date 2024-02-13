package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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

		fmt.Printf("REQ H: %v\n", r.Header)
		fmt.Printf("REQ B: %v\n", r.Body)

		token := ""
		var claims *session.JWTClaims
		// get jwt token from auth header
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer") || strings.HasPrefix(auth, "Basic") {
			splitToken := strings.Split(auth, " ")
			if len(splitToken) < 2 {
				a.jsonMsg(w, r, "cant retrieve token", http.StatusUnauthorized)
				return
			}
			token = strings.TrimSpace(splitToken[1])
		} else {
			sessionCookie, err := r.Cookie("session")
			if err != http.ErrNoCookie {
				token = sessionCookie.Value
			} else {
				a.jsonMsg(w, r, "no token provided", http.StatusUnauthorized)
				return
			}
		}

		if !strings.HasPrefix(auth, "Basic") {
			cl, err := a.SessMgr.GetJWTClaimsFromToken(token)
			if err != nil {
				a.jsonMsg(w, r, err.Error(), http.StatusUnauthorized)
				return
			}
			claims = cl
		} else {
			//workaround to disable authentication for api but still have userid in context
			//need to refactor so that session id is consistent
			usernameDec, _ := base64.URLEncoding.DecodeString(token)
			username := string(usernameDec)
			u, err := a.UserRepo.GetUser(username)
			if err != nil {
				a.jsonMsg(w, r, err.Error(), http.StatusUnauthorized)
				return
			}
			claims = &session.JWTClaims{
				Sid: &session.Session{
					ID:     fmt.Sprintf("%v_%v", u.ID, time.Now().Format(time.RFC3339Nano)),
					UserID: u.ID,
				},
				User: user.User{},
			}
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
