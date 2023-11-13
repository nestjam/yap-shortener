package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nestjam/yap-shortener/internal/client"
)

func main() {
	const (
		minCount          = 2
		wrongArgs         = "get or shorten subcommand required"
		shortenSubcommand = "shorten"
		getSubcommand     = "get"
	)

	if len(os.Args) < minCount {
		exit(wrongArgs)
	}

	shortenSet := flag.NewFlagSet(shortenSubcommand, flag.ExitOnError)
	serverAddr := shortenSet.String("a", "http://localhost:8080/", "address of shortener server")

	getSet := flag.NewFlagSet(getSubcommand, flag.ExitOnError)

	switch os.Args[1] {
	case shortenSubcommand:
		err := shortenSet.Parse(os.Args[minCount:])

		if err != nil {
			exit(err)
		}

		shortenURLs(shortenSet.Args(), *serverAddr)
	case getSubcommand:
		err := getSet.Parse(os.Args[minCount:])

		if err != nil {
			exit(err)
		}

		getURLs(getSet.Args())
	default:
		exit(wrongArgs)
	}
}

func exit(msg any) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func getURLs(urls []string) {
	client := client.New("")
	
	for _, url := range urls {
		fullURL, err := client.GetFull(url)

		if err != nil {
			exit(err)
		}

		fmt.Println(fullURL)
	}
}

func shortenURLs(urls []string, addr string) {
	client := client.New(addr)

	for _, url := range urls {
		shortenedURL, err := client.Shorten(url)

		if err != nil {
			exit(err)
		}

		fmt.Println(shortenedURL)
	}
}