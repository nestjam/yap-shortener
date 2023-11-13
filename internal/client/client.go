package client

import (
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	inner   *resty.Client
	baseURL string
}

func New(baseURL string) *Client {
	client := &Client{
		resty.New(),
		baseURL,
	}

	client.inner.SetRedirectPolicy(
		resty.RedirectPolicyFunc(func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}),
	)

	return client
}

func (c *Client) GetFull(shortURL string) (string, error) {
	const op = "get full URL"
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
		Post(c.baseURL)

	if err != nil {
		return "", fmt.Errorf("shorten URL: %w", err)
	}

	return string(response.Body()), nil
}
