package botuser

import (
	"errors"
	"fmt"

	"golang.org/x/oauth2"
)

type TgUser struct {
	Fquid         string //fully qualified username - provider + id from oauth service
	Userid        string
	Email         string
	Provider      string
	ProviderToken *oauth2.Token
}

func (g *TgUser) SetFQUID() string {
	g.Fquid = fmt.Sprintf("%v:%v", g.Provider, g.Userid)
	return g.Fquid
}

type TgUserRepository interface {
	AddUser(chatid int64, token *oauth2.Token, providerName string) (*TgUser, error)
	GetUser(chatid int64) (*TgUser, error)
}

var (
	ErrorUserIdIsNullOrEmpty  = errors.New("userid is null or empty")
	ErrorUserNotAuthenticated = errors.New("user not authenticated")
)
