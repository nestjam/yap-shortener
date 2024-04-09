package auth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/nestjam/yap-shortener/internal/domain"
)

const (
	userAuthCookieName = "userauth"
	SecretKey          = "supersecretkey"
	TokenExp           = time.Hour * 3
)

// Claims определяет зарегистрированные утверждения и данные пользователя.
type Claims struct {
	jwt.RegisteredClaims
	UserID uuid.UUID
}

// UserAuth выполняет аутентификацию пользователя.
type UserAuth struct {
	secret   string
	tokenExp time.Duration
}

// New создает экземпляр UserAuth с указанным секретом и временем жизни токена.
func New(secret string, tokenExp time.Duration) *UserAuth {
	return &UserAuth{
		secret:   secret,
		tokenExp: tokenExp,
	}
}

// GetUserID возвращает идентификатор пользователя из запроса, если они успешно извлечены.
func (a *UserAuth) GetUserID(r *http.Request) (domain.UserID, error) {
	const op = "get user id from request"
	cookie, err := r.Cookie(userAuthCookieName)

	if err != nil {
		return domain.UserID{}, errors.Wrap(err, op)
	}

	userID, err := a.ParseJWT(cookie.Value)

	if err != nil {
		return domain.UserID{}, errors.Wrap(err, op)
	}

	return userID, nil
}

// ParseJWT выполняет парсинг JWT и возвращает идентификатор пользователя в случае успеха.
func (a *UserAuth) ParseJWT(tokenString string) (domain.UserID, error) {
	const op = "parse jwt"
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(a.secret), nil
		})

	if err != nil {
		return domain.UserID{}, errors.Wrap(err, op)
	}

	if !token.Valid {
		return domain.UserID{}, errors.Wrap(err, op)
	}

	return domain.UserID(claims.UserID), nil
}

// CreateCookie возвращает Cookie с идентификатором пользователя.
func (a *UserAuth) CreateCookie(userID domain.UserID) (*http.Cookie, error) {
	const op = "create cookie"
	token, err := a.BuildJWT(userID)

	if err != nil {
		return nil, errors.Wrap(err, op)
	}

	cookie := http.Cookie{
		Name:     userAuthCookieName,
		Value:    token,
		MaxAge:   int(a.tokenExp / time.Second),
		HttpOnly: true,
	}
	return &cookie, nil
}

func (a *UserAuth) BuildJWT(userID domain.UserID) (string, error) {
	const op = "build jwt"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.tokenExp)),
		},
		UserID: uuid.UUID(userID),
	})

	tokenString, err := token.SignedString([]byte(a.secret))

	if err != nil {
		return "", errors.Wrap(err, op)
	}

	return tokenString, nil
}
