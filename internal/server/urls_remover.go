package server

import (
	"context"
	"errors"

	"github.com/nestjam/yap-shortener/internal/domain"
)

type deletingURLs struct {
	shortURLs []string
	userID    domain.UserID
}

type URLRemover struct {
	deleteCh chan deletingURLs
	doneCh   <-chan struct{}
}

func NewURLRemover(ctx context.Context, doneCh <-chan struct{}, store domain.URLStore) *URLRemover {
	r := &URLRemover{
		deleteCh: make(chan deletingURLs),
		doneCh:   doneCh,
	}

	go func() {
		for {
			select {
			case <-r.doneCh:
				return
			case val := <-r.deleteCh:
				_ = store.DeleteUserURLs(ctx, val.shortURLs, val.userID)
			}
		}
	}()

	return r
}

func (r *URLRemover) DeleteURLs(shortURLs []string, userID domain.UserID) error {
	select {
	case <-r.doneCh:
		return errors.New("channel is closed")
	default:
	}

	go func() {
		urls := deletingURLs{
			shortURLs: shortURLs,
			userID:    userID,
		}
		r.deleteCh <- urls
	}()

	return nil
}
