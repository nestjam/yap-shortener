package server

import (
	"context"

	"github.com/nestjam/yap-shortener/internal/domain"
)

type RemovingURLs struct {
	ShortURLs []string
	UserID    domain.UserID
}

type URLRemover struct {
	finalCh chan RemovingURLs
	doneCh  <-chan struct{}
}

func NewURLRemover(ctx context.Context, doneCh <-chan struct{}, store domain.URLStore) *URLRemover {
	r := &URLRemover{
		finalCh: make(chan RemovingURLs),
		doneCh:  doneCh,
	}

	go func() {
		for {
			select {
			case <-r.doneCh:
				return
			case val, ok := <-r.finalCh:
				if !ok {
					return
				}
				_ = store.DeleteUserURLs(ctx, val.ShortURLs, val.UserID)
			}
		}
	}()

	return r
}

func (r *URLRemover) Delete(shortURLs []string, userID domain.UserID) {
	go func() {
		for {
			select {
			case <-r.doneCh:
				return
			default:
				r.finalCh <- RemovingURLs{ShortURLs: shortURLs, UserID: userID}
			}
		}
	}()
}
