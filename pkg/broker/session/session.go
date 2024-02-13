package session

import "errors"

type Session struct {
	ID     string
	UserID string
}

var (
	ErrorTokenInvalid          = errors.New("invalid token")
	ErrNoSession               = errors.New("no session found")
	ErrGetContextClaimsFailure = errors.New("unable to retrieve entity from context")
)
