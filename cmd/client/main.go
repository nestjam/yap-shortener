package main

import (
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
)

func main() {
	shortenedURL, err := shortenURL("http://ya.ru")
	
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(shortenedURL)

	url, err := getFullURL(shortenedURL)
	
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(url)
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
		return "", err
	}
	
	fmt.Println(response.StatusCode())
	return response.Header().Get("Location"), nil
}

func shortenURL(url string) (string, error) {
	client := resty.New()
	response, err := client.R().
		SetBody(url).
		Post("http://localhost:8080/")
	
	if err != nil {
		return "", err
	}

	fmt.Println(response.StatusCode())
	shortURL := string(response.Body())
	
	return shortURL, nil
}
