package middleware

import (
	"net/http"
	"time"

	"github.com/KSerditov/Trading/pkg/broker/custlog"
)

type responseData struct {
	status   int
	size     int
	response []byte
}

/*
Реализация http.ResponseWriter интерфейса с возможностью получить информацию о записанных в ответ данных
*/
type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.response = b
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

/*
Логирование запросов
*/
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lrw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}

		start := time.Now()

		next.ServeHTTP(&lrw, r)

		custlog.CtxLog(r.Context()).Infow("request processed",
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
			"url", r.URL.Path,
			"time", time.Since(start),
			"response_code", responseData.status,
			"response_content", string(responseData.response),
		)
	})
}
