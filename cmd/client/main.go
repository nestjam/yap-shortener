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
		wrongArgs         = "expand or shorten subcommand required"
		shortenSubcommand = "shorten"
		expandSubcommand  = "expand"
	)

	if len(os.Args) < minCount {
		exit(wrongArgs)
	}

	shortenSet := flag.NewFlagSet(shortenSubcommand, flag.ExitOnError)
	serverAddr := shortenSet.String("a", "http://localhost:8080/", "address of shortener server")

	expandSet := flag.NewFlagSet(expandSubcommand, flag.ExitOnError)

	switch os.Args[1] {
	case shortenSubcommand:
		err := shortenSet.Parse(os.Args[minCount:])

		if err != nil {
			exit(err)
		}

		shortenURLs(shortenSet.Args(), *serverAddr)
	case expandSubcommand:
		err := expandSet.Parse(os.Args[minCount:])

		if err != nil {
			exit(err)
		}

		expandURLs(expandSet.Args())
	default:
		exit(wrongArgs)
	}
}

func exit(msg any) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func expandURLs(urls []string) {
	client := client.New()

	for _, url := range urls {
		fullURL, err := client.Expand(url)

		if err != nil {
			exit(err)
		}

		fmt.Println(fullURL)
	}
}

func shortenURLs(urls []string, addr string) {
	client := client.New(client.WithServerAddress(addr))

	for _, url := range urls {
		shortenedURL, err := client.Shorten(url)

		if err != nil {
			exit(err)
		}

		fmt.Println(shortenedURL)
	}
}
