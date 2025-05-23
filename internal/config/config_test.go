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
		want Config
		name string
		args []string
	}{
		{
			name: "args contain only app name",
			args: []string{
				"app.exe",
			},
			want: Config{},
		},
		{
			name: "args contain server address",
			args: []string{
				"app.exe",
				"-a",
				":8000",
			},
			want: Config{
				ServerAddress: ":8000",
			},
		},
		{
			name: "args contain server address and base URL",
			args: []string{
				"app.exe",
				"-a=:8000",
				"-b=http://localhost:8000",
			},
			want: Config{
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
			want: Config{
				BaseURL: "http://localhost:3000",
			},
		},
		{
			name: "args contain file storage path",
			args: []string{
				"app.exe",
				"-f",
				"tmp/urls.db",
			},
			want: Config{
				FileStoragePath: "tmp/urls.db",
			},
		},
		{
			name: "args contain database connection string",
			args: []string{
				"app.exe",
				"-d",
				"database_name",
			},
			want: Config{
				DataSourceName: "database_name",
			},
		},
		{
			name: "args contain enable https flag",
			args: []string{
				"app.exe",
				"-s",
			},
			want: Config{
				EnableHTTPS: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := Config{}
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
		want Config
	}{
		{
			name: "env contains server address and base URL",
			want: Config{
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
			want: Config{
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
			want: Config{
				BaseURL: "shrt.ru",
			},
			env: &testEnvironment{
				m: map[string]string{
					"BASE_URL": "shrt.ru",
				},
			},
		},
		{
			name: "env contains file storage path",
			want: Config{
				FileStoragePath: "tmp/file.db",
			},
			env: &testEnvironment{
				m: map[string]string{
					"FILE_STORAGE_PATH": "tmp/file.db",
				},
			},
		},
		{
			name: "env contains database connection string",
			want: Config{
				DataSourceName: "database_name",
			},
			env: &testEnvironment{
				m: map[string]string{
					"DATABASE_DSN": "database_name",
				},
			},
		},
		{
			name: "env contains enable HTTPS flag",
			want: Config{
				EnableHTTPS: true,
			},
			env: &testEnvironment{
				m: map[string]string{
					"ENABLE_HTTPS": "true",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Config{}.FromEnv(tt.env)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("failed to parse bool", func(t *testing.T) {
		env := &testEnvironment{
			m: map[string]string{
				"ENABLE_HTTPS": "enable",
			},
		}

		conf := New()
		assert.Panics(t, func() { _ = conf.FromEnv(env) })
	})
}

func TestNew(t *testing.T) {
	t.Run("new", func(t *testing.T) {
		want := Config{
			ServerAddress: defaultServerAddr,
			BaseURL:       defaultBaseURL,
		}

		got := New()

		assert.Equal(t, want, got)
	})
}

func TestFromJSON(t *testing.T) {
	t.Run("from json", func(t *testing.T) {
		want := Config{
			ServerAddress:   "localhost:8080",
			BaseURL:         "http://localhost",
			FileStoragePath: "/path/to/file.db",
			EnableHTTPS:     true,
		}
		const json = `{
	"server_address": "localhost:8080",
	"base_url": "http://localhost",
	"file_storage_path": "/path/to/file.db",
	"database_dsn": "",
	"enable_https": true
} `

		got := Config{}.FromJSON([]byte(json))

		assert.Equal(t, want, got)
	})

	t.Run("failed to parse json", func(t *testing.T) {
		const json = `{
	"server_address": "localhost:8080"{{,
	"base_url": "http://localhost",
	"file_storage_path": "/path/to/file.db",
	"database_dsn": "",
	"enable_
} `

		assert.Panics(t, func() { _ = Config{}.FromJSON([]byte(json)) })
	})
}

func TestGetConfigFileFromArgs(t *testing.T) {
	t.Run("file path from cmd arg", func(t *testing.T) {
		args := []string{
			"app.exe",
			"-c",
			"/path/to/file.json",
			"-d",
			"database_name",
		}
		const want = "/path/to/file.json"

		got := GetConfigFileFromArgs(args)

		assert.Equal(t, want, got)
	})

	t.Run("file path is not in cmd args", func(t *testing.T) {
		args := []string{
			"app.exe",
			"-d",
			"database_name",
		}

		path := GetConfigFileFromArgs(args)

		assert.Equal(t, "", path)
	})
}

func TestGetConfigFileFromEnv(t *testing.T) {
	t.Run("file path from env", func(t *testing.T) {
		env := &testEnvironment{
			m: map[string]string{
				"CONFIG": "/path/to/file.json",
			},
		}
		const want = "/path/to/file.json"

		got := GetConfigFileFromEnv(env)

		assert.Equal(t, want, got)
	})

	t.Run("file path is not in env", func(t *testing.T) {
		env := &testEnvironment{
			m: map[string]string{
				"PATH": "/path/to/file.json",
			},
		}

		got := GetConfigFileFromEnv(env)

		assert.Equal(t, "", got)
	})
}
