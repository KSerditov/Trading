package user

import (
	"errors"
)

type UserRepository interface {
	GetUser(username string) (User, error)
	GetUserById(userid string) (User, error)
	AddUser(username string, password string) (User, error)
	ValidatePassword(user User, password string) (bool, error)
}

var (
	ErrorUserNotFound      = errors.New("user not found")
	ErrorUserAlreadyExists = errors.New("user already exists")
)
