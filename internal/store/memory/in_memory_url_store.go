package memory

type InMemoryUrlStore struct {
	m map[string]string
}

func NewInMemoryUrlStore() *InMemoryUrlStore {
	return &InMemoryUrlStore{
		m: map[string]string{
			"EwHXdJfB": "https://practicum.yandex.ru/",
		},
	}
}

func (s *InMemoryUrlStore) Get(key string) string {
	return s.m[key]
}
