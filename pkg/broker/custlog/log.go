package custlog

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

type ClaimsLoggerKey struct {
}

type ClaimsRequestIDKey struct {
}

var (
	defaultLogger *zap.SugaredLogger
)

type Logger struct {
	Zap   *zap.Logger
	Level int
}

func (ac *Logger) SetupReqID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = RandBytesHex(16)
			r.Header.Set("X-Request-ID", requestID)
			r.Header.Set("trace-id", requestID)
			w.Header().Set("trace-id", requestID)
			w.Header().Set("X-Request-ID", requestID)
		}
		ctx := context.WithValue(r.Context(), ClaimsRequestIDKey{}, requestID)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func GetRequestIDFromContext(ctx context.Context) string {
	requestID, ok := ctx.Value(ClaimsRequestIDKey{}).(string)
	if !ok {
		return "-"
	}
	return requestID
}

func (ac *Logger) SetupLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxlogger := ac.Zap.With(
			zap.String("logger", "ctxlog"),
			zap.String("trace-id", GetRequestIDFromContext(r.Context())),
		).WithOptions(
			//zap.IncreaseLevel(minLevel),
			zap.AddCaller(),
			// zap.AddCallerSkip(1),
			zap.AddStacktrace(zap.ErrorLevel),
		).Sugar()

		ctx := context.WithValue(r.Context(), ClaimsLoggerKey{}, ctxlogger)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func CtxLog(ctx context.Context) *zap.SugaredLogger {
	if ctx == nil {
		return defaultLogger
	}
	zap, ok := ctx.Value(ClaimsLoggerKey{}).(*zap.SugaredLogger)
	if !ok || zap == nil {
		return defaultLogger
	}
	return zap
}

func RandBytesHex(n int) string {
	return fmt.Sprintf("%x", RandBytes(n))
}

func RandBytes(n int) []byte {
	res := make([]byte, n)
	rand.Read(res)
	return res
}
