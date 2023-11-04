package config

import "flag"

type config struct {
	RunAddr  string
	BaseAddr string
}

const (
	defaultRunAddr  = ":8080"
	defaultBaseAddr = "http://localhost:8080"
)

func Parse(args []string) config {
	var config = config{}

	flagSet := flag.NewFlagSet("", flag.PanicOnError)
	flagSet.StringVar(&config.RunAddr, "a", defaultRunAddr, "run address")
	flagSet.StringVar(&config.BaseAddr, "b", defaultBaseAddr, "base address")
	_ = flagSet.Parse(args[1:]) // exclude command name
	
	return config
}
