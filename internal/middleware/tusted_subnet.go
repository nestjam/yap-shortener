package middleware

import (
	"net"
	"net/http"
)

const failedToParseCIDRMessage = "failed to parse trusted subnet address"

// TrustedSubnet возвращает посредника, который проверяет IP адрес пользователя на принадлежность доверенной сети.
func TrustedSubnet(subnet string) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		f := func(w http.ResponseWriter, r *http.Request) {
			if subnet == "" {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			_, ipNet, err := net.ParseCIDR(subnet)
			if err != nil {
				http.Error(w, failedToParseCIDRMessage, http.StatusInternalServerError)
				return
			}

			ip := net.ParseIP(r.Header.Get("X-Real-IP"))

			if !ipNet.Contains(ip) {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(f)
	}
}
