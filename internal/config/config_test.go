package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testEnvironment struct {
	m map[string]string
}

func (t *testEnvironment) LookupEnv(key string) (string, bool) {
	v, ok := t.m[key]
	return v, ok
}

func TestConfigFromArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want config
	}{
		{
			name: "args contain only app name",
			args: []string{
				"app.exe",
			},
			want: config{
				ServerAddress: defaultServerAddr,
				BaseURL:       defaultBaseURL,
			},
		},
		{
			name: "args contain server address",
			args: []string{
				"app.exe",
				"-a",
				":8000",
			},
			want: config{
				ServerAddress: ":8000",
				BaseURL:       defaultBaseURL,
			},
		},
		{
			name: "args contain server address and base URL",
			args: []string{
				"app.exe",
				"-a=:8000",
				"-b=http://localhost:8000",
			},
			want: config{
				ServerAddress: ":8000",
				BaseURL:       "http://localhost:8000",
			},
		},
		{
			name: "args contain base URL",
			args: []string{
				"app.exe",
				"-b",
				"http://localhost:3000",
			},
			want: config{
				ServerAddress: defaultServerAddr,
				BaseURL:       "http://localhost:3000",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := New()
			got := conf.FromArgs(tt.args)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("failed to parse args", func(t *testing.T) {
		args := []string{
			"app.exe",
			"-b=",
			"-a",
		}

		conf := New()
		assert.Panics(t, func() { _ = conf.FromArgs(args) })
	})
}

func TestConfigFromEnv(t *testing.T) {
	tests := []struct {
		name string
		env  Environment
		want config
	}{
		{
			name: "env contains server address and base URL",
			want: config{
				BaseURL:       "shrt.ru",
				ServerAddress: ":8080",
			},
			env: &testEnvironment{
				m: map[string]string{
					"SERVER_ADDRESS": ":8080",
					"BASE_URL":       "shrt.ru",
				},
			},
		},
		{
			name: "env contains server address",
			want: config{
				BaseURL:       defaultBaseURL,
				ServerAddress: ":8080",
			},
			env: &testEnvironment{
				m: map[string]string{
					"SERVER_ADDRESS": ":8080",
				},
			},
		},
		{
			name: "env contains base URL",
			want: config{
				BaseURL:       "shrt.ru",
				ServerAddress: defaultServerAddr,
			},
			env: &testEnvironment{
				m: map[string]string{
					"BASE_URL": "shrt.ru",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New().FromEnv(tt.env)
			assert.Equal(t, tt.want, got)
		})
	}
}
