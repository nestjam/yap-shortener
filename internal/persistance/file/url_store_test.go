package file

import (
	"bytes"
	"os"
	"testing"

	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLStore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test.")
	}
	domain.URLStoreContract{
		NewURLStore: func() (domain.URLStore, func()) {
			t.Helper()
			f, err := os.CreateTemp(os.TempDir(), "*")

			require.NoError(t, err)

			store, err := New(f)

			require.NoError(t, err)

			return store, func() {
				_ = os.Remove(f.Name())
			}
		},
	}.Test(t)
}

func TestGet(t *testing.T) {
	t.Run("invalid data", func(t *testing.T) {
		data := "invalid_data"
		rw := bytes.NewBuffer([]byte(data))
		_, err := New(rw)

		assert.Error(t, err)
	})
}
