package config

import "flag"

type Config struct {
	ServerAddress string
	BaseURL       string
}

const (
	defaultServerAddr = ":8080"
	defaultBaseURL    = "http://localhost:8080"
)

type Environment interface {
	LookupEnv(key string) (string, bool)
}

func New() Config {
	return Config{
		ServerAddress: defaultServerAddr,
		BaseURL:       defaultBaseURL,
	}
}

func (conf Config) FromArgs(args []string) Config {
	flagSet := flag.NewFlagSet("", flag.PanicOnError)
	flagSet.StringVar(&conf.ServerAddress, "a", conf.ServerAddress, "server address")
	flagSet.StringVar(&conf.BaseURL, "b", conf.BaseURL, "base URL")
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
	return conf
}
