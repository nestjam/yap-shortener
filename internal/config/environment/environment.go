package environment

import "os"

type Environment struct {
}

func New() Environment {
	return Environment{}
}

func (env Environment) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}
