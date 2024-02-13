package oauth

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/vk"
)

type OauthProviderVk struct {
	BotName        string
	VkAppId        string
	VkAppSecret    string
	BaseHost       string
	BaseHttpScheme string
}

func (o *OauthProviderVk) GetProviderName() string {
	return "VK"
}

func (o *OauthProviderVk) GetOauthURL() string {
	return fmt.Sprintf("https://oauth.vk.com/authorize?client_id=%v&redirect_uri=%v://%v/login_oauth&response_type=code&scope=email&grant_type=client_credentials",
		o.VkAppId,
		o.BaseHttpScheme,
		o.BaseHost)
}

func (o *OauthProviderVk) LoginOauth(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	tg := fmt.Sprintf("https://t.me/%v?start=%v", o.BotName, code)
	http.Redirect(w, r, tg, http.StatusSeeOther)
}

func (o *OauthProviderVk) GetToken(code string) (*oauth2.Token, error) {
	conf := oauth2.Config{
		ClientID:     o.VkAppId,
		ClientSecret: o.VkAppSecret,
		RedirectURL:  fmt.Sprintf("%v://%v/login_oauth", o.BaseHttpScheme, o.BaseHost),
		Endpoint:     vk.Endpoint,
	}

	token, err := conf.Exchange(context.TODO(), code)
	if err != nil {
		return nil, err
	}
	return token, nil
}
