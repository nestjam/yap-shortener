package client

import (
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	inner         *resty.Client
	serverAddress string
}

type Option func(*Client)

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

func WithServerAddress(addr string) Option {
	return func(client *Client) {
		client.serverAddress = addr
	}
}

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

func (c *Client) Shorten(url string) (string, error) {
	response, err := c.inner.R().
		SetBody(url).
		Post(c.serverAddress)

	if err != nil {
		return "", fmt.Errorf("shorten URL: %w", err)
	}

	return string(response.Body()), nil
}
