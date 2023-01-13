package main

import (
	"github.com/KSerditov/Trading/pkg/tgclient/bot"
	"github.com/KSerditov/Trading/pkg/tgclient/brokerclient"
	"github.com/KSerditov/Trading/pkg/tgclient/oauth"
	"github.com/KSerditov/Trading/pkg/tgclient/router"
)

const (
	VkAppId        = "51524628"
	VkAppSecret    = "DhMi4YQxZ5M1kMyCzEOI"
	BaseURL        = "http://localhost:8084"
	BaseHost       = "localhost:8084"
	BaseHttpScheme = "http"
	BotName        = "GoLangCourse2023Bot"
	BotToken       = "5650611255:AAErRuKFCxBuzsJCCHz19UXICAHrbujxxSM"
	BrokerBaseURL  = "http://localhost:8080"
)

func main() {
	a := &oauth.OauthProviderVk{
		BotName:        BotName,
		VkAppId:        VkAppId,
		VkAppSecret:    VkAppSecret,
		BaseHost:       BaseHost,
		BaseHttpScheme: BaseHttpScheme,
	}

	br := &brokerclient.BrokerClientHttp{
		BrokerBaseURL: BrokerBaseURL,
	}

	r := &router.TgClientRouter{
		BaseHost:     BaseHost,
		TgBotName:    BotName,
		AuthProvider: a,
	}
	r.ListenAndServe()

	b := &bot.TgBot{
		BotToken:     BotToken,
		Debug:        true,
		AuthProvider: a,
		BrokerClient: br,
	}
	b.ListenAndServe()

}
