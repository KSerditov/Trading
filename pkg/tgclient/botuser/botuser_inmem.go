package botuser

import (
	"fmt"
	"sync"

	"golang.org/x/oauth2"
)

type TgUserRepositoryInMem struct {
	UsersLock *sync.RWMutex
	Users     map[int64]*TgUser
}

func (t *TgUserRepositoryInMem) AddUser(chatid int64, token *oauth2.Token, providerName string) (*TgUser, error) {
	t.UsersLock.Lock()
	defer t.UsersLock.Unlock()

	useridstring := token.Extra("user_id")
	if useridstring == "" {
		return nil, ErrorUserIdIsNullOrEmpty
	}

	user := &TgUser{
		Userid:        fmt.Sprint(int(useridstring.(float64))),
		Email:         token.Extra("email").(string),
		Provider:      providerName,
		ProviderToken: token,
	}
	user.SetFQUID()
	t.Users[chatid] = user

	return t.Users[chatid], nil
}

func (t *TgUserRepositoryInMem) GetUser(chatid int64) (*TgUser, error) {
	t.UsersLock.RLock()
	defer t.UsersLock.RUnlock()

	u, ok := t.Users[chatid]
	if !ok {
		return nil, ErrorUserNotAuthenticated
	}

	return u, nil
}
