package interceptor

import (
	"context"

	"github.com/nestjam/yap-shortener/internal/auth"
	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	customctx "github.com/nestjam/yap-shortener/internal/context"
)

const AuthorizationKey = "authorization"

type AuthInterceptor struct {
	userAuth *auth.UserAuth
}

func NewAuth(userAuth *auth.UserAuth) *AuthInterceptor {
	return &AuthInterceptor{userAuth: userAuth}
}

//nolint:lll // default interceptor func type
func (i *AuthInterceptor) Handle(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	user := customctx.NewUser(domain.NewUserID(), true)

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(AuthorizationKey); len(values) > 0 {
			token := values[0]
			userID, _ := i.userAuth.ParseJWT(token)
			user = customctx.NewUser(userID, false)
		}
	}
	ctx = customctx.SetUser(ctx, user)

	resp, err = handler(ctx, req)
	if err != nil {
		err = errors.Wrap(err, "auth interceptor")
	}

	return
}
