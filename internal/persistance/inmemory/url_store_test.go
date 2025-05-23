package inmemory

import (
	"testing"

	"github.com/nestjam/yap-shortener/internal/domain"
)

func TestInmemoryURLStore(t *testing.T) {
	domain.URLStoreContract{
		NewURLStore: func() (domain.URLStore, func()) {
			t.Helper()
			store := New()

			return store, func() {
			}
		},
	}.Test(t)
}
