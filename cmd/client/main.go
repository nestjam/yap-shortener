package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/go-resty/resty/v2"
)

func main() {
	const (
		minCount          = 2
		wrongArgs         = "short or full subcommand required"
		shortenSubcommand = "short"
		getSubcommand     = "full"
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
	for _, url := range urls {
		fullURL, err := getFullURL(url)

		if err != nil {
			exit(err)
		}

		fmt.Println(fullURL)
	}
}

func shortenURLs(urls []string, addr string) {
	for _, url := range urls {
		shortenedURL, err := shortenURL(url, addr)

		if err != nil {
			exit(err)
		}

		fmt.Println(shortenedURL)
	}
}

func getFullURL(shortURL string) (string, error) {
	client := resty.New()
	client.SetRedirectPolicy(
		resty.RedirectPolicyFunc(func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}),
	)

	response, err := client.R().Get(shortURL)

	if err != nil {
		return "", fmt.Errorf("get full URL: %w", err)
	}

	return response.Header().Get("Location"), nil
}

func shortenURL(url string, addr string) (string, error) {
	client := resty.New()
	response, err := client.R().
		SetBody(url).
		Post(addr)

	if err != nil {
		return "", fmt.Errorf("shorten URL: %w", err)
	}

	return string(response.Body()), nil
}
