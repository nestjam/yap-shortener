package domain

import "fmt"

type URLStoreDelegate struct {
	GetFunc         func(shortURL string) (string, error)
	AddFunc         func(shortURL, url string) error
	AddBatchFunc    func(pairs []URLPair) error
	IsAvailableFunc func() bool
	delegate        URLStore
}

func NewURLStoreDelegate(delegate URLStore) *URLStoreDelegate {
	return &URLStoreDelegate{delegate: delegate}
}

func (u *URLStoreDelegate) Get(shortURL string) (string, error) {
	if u.GetFunc != nil {
		return u.GetFunc(shortURL)
	}
	url, err := u.delegate.Get(shortURL)

	if err != nil {
		return "", fmt.Errorf("get url from store delegate: %w", err)
	}

	return url, nil
}

func (u *URLStoreDelegate) Add(shortURL, url string) error {
	if u.AddFunc != nil {
		return u.AddFunc(shortURL, url)
	}
	err := u.delegate.Add(shortURL, url)

	if err != nil {
		return fmt.Errorf("add url to store delegate: %w", err)
	}

	return nil
}

func (u *URLStoreDelegate) IsAvailable() bool {
	if u.IsAvailableFunc != nil {
		return u.IsAvailableFunc()
	}
	return u.delegate.IsAvailable()
}

func (u *URLStoreDelegate) AddBatch(pairs []URLPair) error {
	panic("not implemented")
}
