package client

import (
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
)

// Client представляет клиент сервса сокращения ссылок.
type Client struct {
	inner         *resty.Client
	serverAddress string
}

// Option определяет опцию настройки клиента.
type Option func(*Client)

// New созадет экземпляр клиента с переданными опциями.
func New(options ...Option) *Client {
	client := &Client{
		inner: resty.New(),
	}

	client.inner.SetRedirectPolicy(
		resty.RedirectPolicyFunc(func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}),
	)

	for _, opt := range options {
		opt(client)
	}

	return client
}

// WithServerAddress возвращает опцию клиента с указанным адресом сервера.
func WithServerAddress(addr string) Option {
	return func(client *Client) {
		client.serverAddress = addr
	}
}

// Expand возвращает исходный URL по сокращенному, иначе - ошибку.
func (c *Client) Expand(shortURL string) (string, error) {
	const op = "expand URL"
	response, err := c.inner.R().Get(shortURL)

	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return "", fmt.Errorf("%s: not found", op)
	}

	return response.Header().Get("Location"), nil
}

// Shorten выполняет сокращение URL.
// Возвращает сокращенный URL в случае успеха, иначе - ошибку.
func (c *Client) Shorten(url string) (string, error) {
	response, err := c.inner.R().
		SetBody(url).
		Post(c.serverAddress)

	if err != nil {
		return "", fmt.Errorf("shorten URL: %w", err)
	}

	return string(response.Body()), nil
}
