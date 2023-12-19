package domain

import (
	"context"
	"fmt"
)

// A URLStoreDelegate allows to extend the behavior of the test double for negative scenarios
// for URLStore consumers.
type URLStoreDelegate struct {
	GetOriginalURLFunc func(ctx context.Context, shortURL string) (string, error)
	AddURLFunc         func(ctx context.Context, shortURL, url string) error
	AddURLsFunc        func(ctx context.Context, pairs []URLPair) error
	IsAvailableFunc    func(ctx context.Context) bool
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

func (u *URLStoreDelegate) AddURL(ctx context.Context, shortURL, url string) error {
	if u.AddURLFunc != nil {
		return u.AddURLFunc(ctx, shortURL, url)
	}
	err := u.delegate.AddURL(ctx, shortURL, url)

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

func (u *URLStoreDelegate) AddURLs(ctx context.Context, pairs []URLPair) error {
	if u.AddURLsFunc != nil {
		return u.AddURLsFunc(ctx, pairs)
	}

	err := u.delegate.AddURLs(ctx, pairs)

	if err != nil {
		return fmt.Errorf("add batch of urls to store delegate: %w", err)
	}

	return nil
}
