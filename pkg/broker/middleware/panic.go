package middleware

import (
	"net/http"

	"github.com/KSerditov/Trading/pkg/broker/custlog"
	"go.uber.org/zap"
)

func Panic(logger *zap.SugaredLogger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				http.Error(w, "internal server error", 500)

				custlog.CtxLog(r.Context()).Errorf("panic, error: %v", err)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
