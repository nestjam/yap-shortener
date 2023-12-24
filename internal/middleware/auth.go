package middleware

import (
	"net/http"

	"github.com/nestjam/yap-shortener/internal/auth"
	customctx "github.com/nestjam/yap-shortener/internal/context"
	"github.com/nestjam/yap-shortener/internal/domain"
)

func Auth(a *auth.UserAuth) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		log := func(w http.ResponseWriter, r *http.Request) {
			userID, isNew := createOrGetUserID(r, a)

			if isNew {
				addUserID(w, a, userID)
			}

			ctx := r.Context()
			user := customctx.NewUser(userID, isNew)
			h.ServeHTTP(w, r.WithContext(customctx.SetUser(ctx, user)))
		}
		return http.HandlerFunc(log)
	}
}

func createOrGetUserID(r *http.Request, a *auth.UserAuth) (domain.UserID, bool) {
	userID, err := a.GetUserID(r)

	if err != nil {
		return domain.NewUserID(), true
	}

	return userID, false
}

func addUserID(w http.ResponseWriter, a *auth.UserAuth, userID domain.UserID) {
	cookie, _ := a.CreateCookie(userID)
	http.SetCookie(w, cookie)
}
