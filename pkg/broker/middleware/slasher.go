package middleware

import (
	"net/http"
	"strings"
)

/*
Slasher - middleware, позволяющее отрезать хвостовой / в URL входящего запроса.
Теперь достаточно роутера определенного без слэша в конце, для работы в обоих случаях:
api.HandleFunc("/login", UserHandlers.Login) сработает и для /login и для /login/
*/
func Slasher(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
		}

		next.ServeHTTP(w, r)
	})
}
