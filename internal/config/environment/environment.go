package environment

import "os"

// Environment определяет доступ к переменным среды. 
type Environment struct {
}

// New создает экземпляр Environment.
func New() Environment {
	return Environment{}
}

// LookupEnv возвращает значение переменной среды по ключу, если переменная существует.
func (env Environment) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}
