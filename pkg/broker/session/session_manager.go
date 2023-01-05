package session

import (
	"context"
	"math/rand"
	"net/http"
	"time"

	"github.com/KSerditov/Trading/pkg/broker/user"

	"github.com/golang-jwt/jwt/v4"
)

type ClaimsContextKey struct {
}

var (
	letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

type JWTSessionManager struct {
	Secret      []byte
	SessionRepo SessionRepository
}

type JWTClaims struct {
	Sid  *Session  `json:"sid"`
	User user.User `json:"user"`
	jwt.RegisteredClaims
}

func (sm *JWTSessionManager) GetJWTClaimsFromToken(token string) (*JWTClaims, error) {
	parsedtoken, err := jwt.ParseWithClaims(token, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(sm.Secret), nil
	})
	if err != nil {
		return &JWTClaims{}, err
	}

	claims, ok := parsedtoken.Claims.(*JWTClaims)
	if !ok || !parsedtoken.Valid {
		return &JWTClaims{}, ErrorTokenInvalid
	}

	valid, serr := sm.SessionRepo.ValidateSession(claims.Sid)
	if serr != nil {
		return &JWTClaims{}, serr
	}
	if !valid {
		return &JWTClaims{}, ErrNoSession
	}

	return claims, nil
}

func (sm *JWTSessionManager) GetNewSession(user user.User) (string, *Session, error) {
	sid := GetNewSession(user.ID)

	var now = time.Now()
	var duration = 24 * time.Hour

	claims := JWTClaims{
		Sid:  sid,
		User: user,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(sm.Secret)
	if err != nil {
		return "", &Session{}, err
	}

	rerr := sm.SessionRepo.SaveSession(sid, duration)
	if rerr != nil {
		return "", &Session{}, rerr
	}

	return tokenString, sid, nil
}

func GetNewSession(userID string) *Session {
	return &Session{
		ID:     RandStringRunes(16),
		UserID: userID,
	}
}

func (sm *JWTSessionManager) GetSessionFromContext(ctx context.Context) (*Session, error) {
	jwtClaims, ok := ctx.Value(ClaimsContextKey{}).(*JWTClaims)
	if !ok {
		return nil, ErrGetContextClaimsFailure
	}

	return jwtClaims.Sid, nil
}

func (sm *JWTSessionManager) GetUserFromContext(ctx context.Context) (*user.User, error) {
	jwtClaims, ok := ctx.Value(ClaimsContextKey{}).(*JWTClaims)
	if !ok {
		return nil, ErrGetContextClaimsFailure
	}

	return &jwtClaims.User, nil
}

func (sm *JWTSessionManager) DestroyCurrent(w http.ResponseWriter, r *http.Request) error {
	sess, err := sm.GetSessionFromContext(r.Context())
	if err == nil {
		err1 := sm.SessionRepo.DeleteSession(sess.ID)
		if err1 != nil {
			return err1
		}
	}

	cookie := &http.Cookie{
		Name:    "session",
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),

		HttpOnly: true,
	}

	http.SetCookie(w, cookie)

	return nil
}

func RandStringRunes(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
