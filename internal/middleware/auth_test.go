package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nestjam/yap-shortener/internal/auth"
	customctx "github.com/nestjam/yap-shortener/internal/context"
	"github.com/nestjam/yap-shortener/internal/domain"
)

const secretKey = "supersecretkey"
const tokenExp = time.Hour * 3

func TestAuth(t *testing.T) {
	t.Run("get user id from cookie", func(t *testing.T) {
		wantUserID := domain.NewUserID()
		request := newRequestWithUserID(t, wantUserID)
		response := httptest.NewRecorder()
		var user customctx.User
		var ok bool
		noOpHandlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			user, ok = customctx.GetUser(ctx)
		})
		a := auth.New(secretKey, tokenExp)
		sut := Auth(a)(noOpHandlerFunc)

		sut.ServeHTTP(response, request)

		assert.True(t, ok)
		assert.False(t, user.IsNew)
		assert.Equal(t, wantUserID, user.ID)
	})

	t.Run("no cookie with user id", func(t *testing.T) {
		request := newRequest(t)
		response := httptest.NewRecorder()
		var ok bool
		var user customctx.User
		noOpHandlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			user, ok = customctx.GetUser(ctx)
		})
		a := auth.New(secretKey, tokenExp)
		sut := Auth(a)(noOpHandlerFunc)

		sut.ServeHTTP(response, request)

		assert.True(t, ok)
		assert.True(t, user.IsNew)
		assertResponseContainsUserID(t, user.ID, response)
	})

	t.Run("cookie with user id is not valid", func(t *testing.T) {
		userID := domain.NewUserID()
		request := newRequestWithInvalidToken(t, userID)
		response := httptest.NewRecorder()
		var ok bool
		var user customctx.User
		noOpHandlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			user, ok = customctx.GetUser(ctx)
		})
		a := auth.New(secretKey, tokenExp)
		sut := Auth(a)(noOpHandlerFunc)

		sut.ServeHTTP(response, request)

		assert.True(t, ok)
		assert.True(t, user.IsNew)
		assertResponseContainsUserID(t, user.ID, response)
	})
}

func newRequestWithUserID(t *testing.T, userID domain.UserID) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	a := auth.New(secretKey, tokenExp)

	cookie, err := a.CreateCookie(userID)
	require.NoError(t, err)

	r.AddCookie(cookie)
	return r
}

func newRequest(t *testing.T) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	return r
}

func newRequestWithInvalidToken(t *testing.T, userID domain.UserID) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	a := auth.New("wrong_secret", tokenExp)

	cookie, err := a.CreateCookie(userID)
	require.NoError(t, err)

	r.AddCookie(cookie)
	return r
}

func assertResponseContainsUserID(t *testing.T, userID domain.UserID, w *httptest.ResponseRecorder) {
	t.Helper()
	resp := w.Result()
	cookies := resp.Cookies()
	defer func() { _ = resp.Body.Close() }()

	a := auth.New(secretKey, tokenExp)
	got, err := a.ParseJWT(cookies[0].Value)
	require.NoError(t, err)

	assert.Equal(t, userID, got)
}
