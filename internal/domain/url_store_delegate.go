package domain

import (
	"context"
	"fmt"
)

type URLStoreDelegate struct {
	GetFunc         func(ctx context.Context, shortURL string) (string, error)
	AddFunc         func(ctx context.Context, shortURL, url string) error
	AddBatchFunc    func(ctx context.Context, pairs []URLPair) error
	IsAvailableFunc func(ctx context.Context) bool
	delegate        URLStore
}

func NewURLStoreDelegate(delegate URLStore) *URLStoreDelegate {
	return &URLStoreDelegate{delegate: delegate}
}

func (u *URLStoreDelegate) Get(ctx context.Context, shortURL string) (string, error) {
	if u.GetFunc != nil {
		return u.GetFunc(ctx, shortURL)
	}
	url, err := u.delegate.Get(ctx, shortURL)

	if err != nil {
		return "", fmt.Errorf("get url from store delegate: %w", err)
	}

	return url, nil
}

func (u *URLStoreDelegate) Add(ctx context.Context, shortURL, url string) error {
	if u.AddFunc != nil {
		return u.AddFunc(ctx, shortURL, url)
	}
	err := u.delegate.Add(ctx, shortURL, url)

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

func (u *URLStoreDelegate) AddBatch(ctx context.Context, pairs []URLPair) error {
	if u.AddBatchFunc != nil {
		return u.AddBatchFunc(ctx, pairs)
	}

	err := u.delegate.AddBatch(ctx, pairs)

	if err != nil {
		return fmt.Errorf("add batch of urls to store delegate: %w", err)
	}

	return nil
}
