package oauth

import (
	"net/http"

	"golang.org/x/oauth2"
)

type OauthProvider interface {
	GetOauthURL() string
	LoginOauth(w http.ResponseWriter, r *http.Request)
	GetToken(code string) (*oauth2.Token, error)
	GetProviderName() string
}
