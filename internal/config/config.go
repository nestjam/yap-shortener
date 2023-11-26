package config

import "flag"

type Config struct {
	ServerAddress   string
	BaseURL         string
	FileStoragePath string
}

const (
	defaultServerAddr      = ":8080"
	defaultBaseURL         = "http://localhost:8080"
	defaultFileStoragePath = "/tmp/short-url-db.json"
)

type Environment interface {
	LookupEnv(key string) (string, bool)
}

func New() Config {
	return Config{
		ServerAddress:   defaultServerAddr,
		BaseURL:         defaultBaseURL,
		FileStoragePath: defaultFileStoragePath,
	}
}

func (conf Config) FromArgs(args []string) Config {
	flagSet := flag.NewFlagSet("", flag.PanicOnError)
	flagSet.StringVar(&conf.ServerAddress, "a", conf.ServerAddress, "server address")
	flagSet.StringVar(&conf.BaseURL, "b", conf.BaseURL, "base URL")
	flagSet.StringVar(&conf.FileStoragePath, "f", conf.FileStoragePath, "file storage path")

	_ = flagSet.Parse(args[1:]) // exclude command name
	return conf
}

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

	return conf
}
