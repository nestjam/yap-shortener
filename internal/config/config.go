package config

import (
	"encoding/json"
	"flag"
	"strconv"
)

// Config описывает конфигурацию сервера сокращения ссылок.
type Config struct {
	ServerAddress   string `json:"server_address"`    // адрес сервера
	BaseURL         string `json:"base_url"`          // базовый адрес сокращенной ссылки
	FileStoragePath string `json:"file_storage_path"` // путь к файловому хранилищу сокращенных ссылок
	DataSourceName  string `json:"database_dsn"`      // строка подключения к БД хранилища сокращенных ссылок
	EnableHTTPS     bool   `json:"enable_https"`      // включение HTTPS в веб-сервере
}

const (
	defaultServerAddr = ":8080"
	defaultBaseURL    = "http://localhost:8080"
)

// Environment определяет доступ к переменным среды.
type Environment interface {
	LookupEnv(key string) (string, bool)
}

// New создает экземпляр конфигурации с настройками по умолчанию.
func New() Config {
	return Config{
		ServerAddress: defaultServerAddr,
		BaseURL:       defaultBaseURL,
	}
}

// FromArgs заполняет параметры конфигурации из аргументов командной строки.
func (conf Config) FromArgs(args []string) Config {
	var path string
	parseArgs(&conf, &path, args)
	return conf
}

func parseArgs(conf *Config, confFilePath *string, args []string) {
	flagSet := flag.NewFlagSet("", flag.PanicOnError)

	flagSet.StringVar(&conf.ServerAddress, "a", conf.ServerAddress, "server address")
	flagSet.StringVar(&conf.BaseURL, "b", conf.BaseURL, "base URL")
	flagSet.StringVar(&conf.FileStoragePath, "f", conf.FileStoragePath, "file storage path")
	flagSet.StringVar(&conf.DataSourceName, "d", conf.DataSourceName, "data source name")
	flagSet.BoolVar(&conf.EnableHTTPS, "s", conf.EnableHTTPS, "enable HTTPS")
	flagSet.StringVar(confFilePath, "c", "", "config file path")

	_ = flagSet.Parse(args[1:]) // exclude command name
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

	if enableHTTPS, ok := env.LookupEnv("ENABLE_HTTPS"); ok {
		enable, err := strconv.ParseBool(enableHTTPS)

		if err != nil {
			panic(err)
		}

		conf.EnableHTTPS = enable
	}

	return conf
}

// FromJSON заполняет параметры конфигурации из файла JSON.
func (conf Config) FromJSON(data []byte) Config {
	err := json.Unmarshal(data, &conf)

	if err != nil {
		panic(err)
	}

	return conf
}

// GetConfigFileFromArgs возваращает путь к файлу конфигурации из аргументов.
// Если путь не указан в аргументах, возвращается пустая строка.
func GetConfigFileFromArgs(args []string) string {
	var path string
	var conf Config
	parseArgs(&conf, &path, args)

	return path
}

// GetConfigFileFromEnv возваращает путь к файлу конфигурации из переменной окружения CONFIG.
// Если переменная окружения отсутствует или путь не зада, возвращается пустая строка.
func GetConfigFileFromEnv(env Environment) string {
	if config, ok := env.LookupEnv("CONFIG"); ok {
		return config
	}

	return ""
}
