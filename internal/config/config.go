package config

import "flag"

type config struct {
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

func New() config {
	return config{
		ServerAddress: defaultServerAddr,
		BaseURL:       defaultBaseURL,
	}
}

func (conf config) FromArgs(args []string) config {
	flagSet := flag.NewFlagSet("", flag.PanicOnError)
	flagSet.StringVar(&conf.ServerAddress, "a", conf.ServerAddress, "server address")
	flagSet.StringVar(&conf.BaseURL, "b", conf.BaseURL, "base URL")
	_ = flagSet.Parse(args[1:]) // exclude command name
	return conf
}

func (conf config) FromEnv(env Environment) config {
	if servAddr, ok := env.LookupEnv("SERVER_ADDRESS"); ok {
		conf.ServerAddress = servAddr
	}
	if baseURL, ok := env.LookupEnv("BASE_URL"); ok {
		conf.BaseURL = baseURL
	}
	return conf
}
