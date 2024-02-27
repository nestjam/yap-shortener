package config

import (
	"flag"
	"os"
	"path"
)

// Config описывает конфигурацию сервера сокращения ссылок.
type Config struct {
	ServerAddress   string // адрес сервера
	BaseURL         string // базовый адрес сокращенной ссылки
	FileStoragePath string // путь к файловому хранилищу сокращенных ссылок
	DataSourceName  string // строка подключения к БД хранилища сокращенных ссылок
}

const (
	defaultServerAddr = ":8080"
	defaultBaseURL    = "http://localhost:8080"
)

var defaultFileStoragePath string = path.Join(os.TempDir(), "short-url-db.json")

// Environment определяет доступ к переменным среды.
type Environment interface {
	LookupEnv(key string) (string, bool)
}

// New создает экземпляр конфигурации с настройками по умолчанию.
func New() Config {
	return Config{
		ServerAddress:   defaultServerAddr,
		BaseURL:         defaultBaseURL,
		FileStoragePath: defaultFileStoragePath,
	}
}

// FromArgs заполняет параметры конфигурации из аргументов командной строки.
func (conf Config) FromArgs(args []string) Config {
	flagSet := flag.NewFlagSet("", flag.PanicOnError)
	flagSet.StringVar(&conf.ServerAddress, "a", conf.ServerAddress, "server address")
	flagSet.StringVar(&conf.BaseURL, "b", conf.BaseURL, "base URL")
	flagSet.StringVar(&conf.FileStoragePath, "f", conf.FileStoragePath, "file storage path")
	flagSet.StringVar(&conf.DataSourceName, "d", conf.DataSourceName, "data source name")

	_ = flagSet.Parse(args[1:]) // exclude command name
	return conf
}

// FromEnv заполняет параметры конфигурации из переменных среды.
func (conf Config) FromEnv(env Environment) Config {
	if servAddr, ok := env.LookupEnv("SERVER_ADDRESS"); ok {
		conf.ServerAddress = servAddr
	}

	if baseURL, ok := env.LookupEnv("BASE_URL"); ok {
		conf.BaseURL = baseURL
	}

	if path, ok := env.LookupEnv("FILE_STORAGE_PATH"); ok {
		conf.FileStoragePath = path
	}

	if dsn, ok := env.LookupEnv("DATABASE_DSN"); ok {
		conf.DataSourceName = dsn
	}

	return conf
}
