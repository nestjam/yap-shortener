package domain

import "fmt"

type URLStoreDelegate struct {
	GetFunc         func(shortURL string) (string, error)
	AddFunc         func(shortURL, url string)
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
		return "", fmt.Errorf("url store delegate: %w", err)
	}

	return url, nil
}

func (u *URLStoreDelegate) Add(shortURL, url string) {
	if u.AddFunc != nil {
		u.AddFunc(shortURL, url)
	}
	u.delegate.Add(shortURL, url)
}

func (u *URLStoreDelegate) IsAvailable() bool {
	if u.IsAvailableFunc != nil {
		return u.IsAvailableFunc()
	}
	return u.delegate.IsAvailable()
}
