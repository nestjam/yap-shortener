package store

type inMemory struct {
	m map[string]string
}

func NewInMemory() *inMemory {
	return &inMemory{
		m: map[string]string{
			"EwHXdJfB": "https://practicum.yandex.ru/",
		},
	}
}

func (s *inMemory) Get(key string) string {
	return s.m[key]
}
