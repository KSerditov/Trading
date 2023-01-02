package middleware

import (
	"encoding/json"
	"io"
	"net/http"
)

/*
JsonContent - проверяет Content-Type application/json в запросе
*/
func CheckIfJsonContent(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			JsonMsg(w, "unknown payload") //TBD logger
			return
		}
		next.ServeHTTP(w, r)
	})
}

func JsonMsg(w io.Writer, msg string) {
	resp, _ := json.Marshal(map[string]interface{}{
		"message": msg,
	})
	w.Write(resp)
}
