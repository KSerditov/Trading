package router

import (
	"net/http"

	"github.com/KSerditov/Trading/pkg/tgclient/oauth"

	"github.com/gorilla/mux"
)

type TgClientRouter struct {
	BaseHost     string
	TgBotName    string
	AuthProvider oauth.OauthProvider
}

func (t *TgClientRouter) ListenAndServe() {
	r := mux.NewRouter()
	r.HandleFunc("/login_oauth", t.AuthProvider.LoginOauth)

	go func() {
		http.ListenAndServe(t.BaseHost, r)
	}()

}
