package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFull(t *testing.T) {
	type args struct {
		shortURL string
	}

	type test struct {
		name string
		args args
		want string
	}

	t.Run("get full url", func(t *testing.T) {
		tt := test{
			args: args{
				"/abc",
			},
			want: "http://ya.ru",
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.URL.String(), tt.args.shortURL)

			w.Header().Set("Location", tt.want)
			w.WriteHeader(http.StatusTemporaryRedirect)
		}))
		defer server.Close()

		client := New(WithServerAddress(server.URL))
		url, err := client.GetFull(server.URL + tt.args.shortURL)

		assert.NoError(t, err)
		assert.Equal(t, tt.want, url)
	})

	t.Run("full url not found", func(t *testing.T) {
		tt := test{
			args: args{
				"/abc",
			},
			want: "get full URL: not found",
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := New(WithServerAddress(server.URL))
		_, err := client.GetFull(server.URL + tt.args.shortURL)

		assert.Error(t, err)
		assert.Equal(t, tt.want, err.Error())
	})

	t.Run("server does not respond", func(t *testing.T) {
		tt := test{
			args: args{
				"/abc",
			},
			want: "get full URL: not found",
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		serverURL := server.URL
		server.Close()
		client := New(WithServerAddress(serverURL))

		_, err := client.GetFull(serverURL + tt.args.shortURL)

		assert.Error(t, err)
	})
}

func TestShorten(t *testing.T) {
	type args struct {
		url string
	}

	type test struct {
		name string
		args args
		want string
	}

	t.Run("shorten url", func(t *testing.T) {
		tt := test{
			args: args{
				"http://ya.ru",
			},
			want: "http://localhost:8080/123",
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			assert.Equal(t, string(body), tt.args.url)

			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(tt.want))
		}))
		defer server.Close()

		client := New(WithServerAddress(server.URL))
		shortURL, err := client.Shorten(tt.args.url)

		assert.NoError(t, err)
		assert.Equal(t, tt.want, shortURL)
	})

	t.Run("server does not respond", func(t *testing.T) {
		tt := test{
			args: args{
				"http://ya.ru",
			},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(tt.want))
		}))
		defer server.Close()

		serverURL := server.URL
		server.Close()
		client := New(WithServerAddress(serverURL))
		_, err := client.Shorten(tt.args.url)

		assert.Error(t, err)
	})
}
