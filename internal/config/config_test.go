package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
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
				RunAddr:  defaultRunAddr,
				BaseAddr: defaultBaseAddr,
			},
		},
		{
			name: "args contain run address",
			args: []string{
				"app.exe",
				"-a",
				":8000",
			},
			want: config{
				RunAddr:  ":8000",
				BaseAddr: defaultBaseAddr,
			},
		},
		{
			name: "args contain run and base addresses",
			args: []string{
				"app.exe",
				"-a=:8000",
				"-b=http://localhost:8000",
			},
			want: config{
				RunAddr:  ":8000",
				BaseAddr: "http://localhost:8000",
			},
		},
		{
			name: "args contain base address",
			args: []string{
				"app.exe",
				"-b",
				"http://localhost:3000",
			},
			want: config{
				RunAddr:  defaultRunAddr,
				BaseAddr: "http://localhost:3000",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.args)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("failed to parse args", func(t *testing.T) {
		args := []string{
			"app.exe",
			"-b=",
			"-a",
		}

		assert.Panics(t, func() { _ = Parse(args) })
	})
}
