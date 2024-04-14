package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrustedSubnet(t *testing.T) {
	t.Run("trusted subnet is not set", func(t *testing.T) {
		request := newRequestFromIP(t, "127.0.0.4")
		response := httptest.NewRecorder()
		noOpHandlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		})
		const subnet = ""
		sut := TrustedSubnet(subnet)(noOpHandlerFunc)

		sut.ServeHTTP(response, request)

		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	t.Run("ip address in x-real-ip header is in trusted subnet", func(t *testing.T) {
		request := newRequestFromIP(t, "127.0.0.4")
		response := httptest.NewRecorder()
		var callCount int
		spyHandlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
		})
		const subnet = "127.0.0.1/24"
		sut := TrustedSubnet(subnet)(spyHandlerFunc)

		sut.ServeHTTP(response, request)

		assert.Equal(t, http.StatusOK, response.Code)
		assert.Equal(t, 1, callCount)
	})

	t.Run("ip address in x-real-ip header is not in trusted subnet", func(t *testing.T) {
		request := newRequestFromIP(t, "192.169.1.11")
		response := httptest.NewRecorder()
		var callCount int
		spyHandlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
		})
		const subnet = "127.0.0.1/24"
		sut := TrustedSubnet(subnet)(spyHandlerFunc)

		sut.ServeHTTP(response, request)

		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	t.Run("failed to parse trusted subnet address", func(t *testing.T) {
		request := newRequestFromIP(t, "127.0.0.1")
		response := httptest.NewRecorder()
		var callCount int
		spyHandlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
		})
		const subnet = "12799.0.0.1/24"
		sut := TrustedSubnet(subnet)(spyHandlerFunc)

		sut.ServeHTTP(response, request)

		assert.Equal(t, http.StatusInternalServerError, response.Code)
	})
}

func newRequestFromIP(t *testing.T, ip string) *http.Request {
	t.Helper()

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Real-IP", ip)

	return r
}
