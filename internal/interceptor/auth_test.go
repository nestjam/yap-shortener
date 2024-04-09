package interceptor

import (
	"context"
	"testing"
	"time"

	"github.com/nestjam/yap-shortener/internal/auth"
	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	customctx "github.com/nestjam/yap-shortener/internal/context"
)

const (
	secret   = "12345"
	tokenExp = time.Hour
)

func TestHandle(t *testing.T) {
	t.Run("invoke handler", func(t *testing.T) {
		auth := auth.New(secret, tokenExp)
		sut := NewAuth(auth)
		ctx := context.Background()
		req := pingMessage{Message: "hello!"}
		var info *grpc.UnaryServerInfo
		h := fakePingHandler{}

		resp, err := sut.Handle(ctx, req, info, h.handle)

		require.NoError(t, err)
		assert.Equal(t, req, resp)
	})

	t.Run("metadata contains valid token", func(t *testing.T) {
		userAuth := auth.New(secret, tokenExp)
		sut := NewAuth(userAuth)
		userID := domain.NewUserID()
		token, err := userAuth.BuildJWT(userID)
		require.NoError(t, err)
		m := map[string]string{
			AuthorizationKey: token,
		}
		md := metadata.New(m)
		ctx := context.Background()
		ctx = metadata.NewIncomingContext(ctx, md)
		req := pingMessage{}
		var info *grpc.UnaryServerInfo
		h := fakePingHandler{}

		_, err = sut.Handle(ctx, req, info, h.handle)

		require.NoError(t, err)
		assertUser(t, userID, h.capturedContext)
	})

	t.Run("metadata does not contain token", func(t *testing.T) {
		userAuth := auth.New(secret, tokenExp)
		sut := NewAuth(userAuth)
		m := map[string]string{}
		md := metadata.New(m)
		ctx := context.Background()
		ctx = metadata.NewIncomingContext(ctx, md)
		req := pingMessage{}
		var info *grpc.UnaryServerInfo
		h := fakePingHandler{}

		_, err := sut.Handle(ctx, req, info, h.handle)

		require.NoError(t, err)
		assertNewUser(t, h.capturedContext)
	})
}

func assertNewUser(t *testing.T, ctx context.Context) {
	t.Helper()

	got, ok := customctx.GetUser(ctx)
	require.True(t, ok)
	assert.True(t, got.IsNew)
}

//nolint:revive //for test only
func assertUser(t *testing.T, want domain.UserID, ctx context.Context) {
	t.Helper()

	got, ok := customctx.GetUser(ctx)
	require.True(t, ok)
	assert.Equal(t, want, got.ID)
	assert.False(t, got.IsNew)
}

type fakePingHandler struct {
	capturedContext context.Context //nolint:containedctx //for test only
}

func (h *fakePingHandler) handle(ctx context.Context, req any) (any, error) {
	h.capturedContext = ctx
	return req, nil
}

type pingMessage struct {
	Message string
}
