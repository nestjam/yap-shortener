package domain

import (
	"context"
	"fmt"
)

// A URLStoreDelegate allows to extend the behavior of the test double for negative scenarios
// for URLStore consumers.
type URLStoreDelegate struct {
	GetOriginalURLFunc func(ctx context.Context, shortURL string) (string, error)
	AddURLFunc         func(ctx context.Context, pair URLPair, userID UserID) error
	AddURLsFunc        func(ctx context.Context, pairs []URLPair, userID UserID) error
	IsAvailableFunc    func(ctx context.Context) bool
	GetUserURLsFunc    func(ctx context.Context, userID UserID) ([]URLPair, error)
	DeleteUserURLsFunc func(ctx context.Context, shortURLs []string, userID UserID) error
	delegate           URLStore
}

func NewURLStoreDelegate(delegate URLStore) *URLStoreDelegate {
	return &URLStoreDelegate{delegate: delegate}
}

func (u *URLStoreDelegate) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	if u.GetOriginalURLFunc != nil {
		return u.GetOriginalURLFunc(ctx, shortURL)
	}
	url, err := u.delegate.GetOriginalURL(ctx, shortURL)

	if err != nil {
		return "", fmt.Errorf("get url from store delegate: %w", err)
	}

	return url, nil
}

func (u *URLStoreDelegate) AddURL(ctx context.Context, pair URLPair, userID UserID) error {
	if u.AddURLFunc != nil {
		return u.AddURLFunc(ctx, pair, userID)
	}
	err := u.delegate.AddURL(ctx, pair, userID)

	if err != nil {
		return fmt.Errorf("add url to store delegate: %w", err)
	}

	return nil
}

func (u *URLStoreDelegate) IsAvailable(ctx context.Context) bool {
	if u.IsAvailableFunc != nil {
		return u.IsAvailableFunc(ctx)
	}

	return u.delegate.IsAvailable(ctx)
}

func (u *URLStoreDelegate) AddURLs(ctx context.Context, pairs []URLPair, userID UserID) error {
	if u.AddURLsFunc != nil {
		return u.AddURLsFunc(ctx, pairs, userID)
	}

	err := u.delegate.AddURLs(ctx, pairs, userID)

	if err != nil {
		return fmt.Errorf("add batch of urls to store delegate: %w", err)
	}

	return nil
}

func (u *URLStoreDelegate) GetUserURLs(ctx context.Context, userID UserID) ([]URLPair, error) {
	if u.GetUserURLsFunc != nil {
		return u.GetUserURLsFunc(ctx, userID)
	}

	urls, err := u.delegate.GetUserURLs(ctx, userID)

	if err != nil {
		return nil, fmt.Errorf("get user urls from store delegate: %w", err)
	}

	return urls, nil
}

func (u *URLStoreDelegate) DeleteUserURLs(ctx context.Context, shortURLs []string, userID UserID) error {
	if u.DeleteUserURLsFunc != nil {
		return u.DeleteUserURLsFunc(ctx, shortURLs, userID)
	}

	err := u.delegate.DeleteUserURLs(ctx, shortURLs, userID)

	if err != nil {
		return fmt.Errorf("delete user urls from store delegate: %w", err)
	}

	return nil
}
