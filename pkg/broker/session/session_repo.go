package session

import (
	"errors"
	"time"
)

type SessionRepository interface {
	SaveSession(session *Session, duration time.Duration) error
	ValidateSession(session *Session) (bool, error)
	DeleteSession(sessionid string) error
}

var (
	ErrorSessionNotFound      = errors.New("session not found")
	ErrorSessionAlreadyExists = errors.New("session already exists")
)
